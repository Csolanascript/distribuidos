// Escribir vuestro código de funcionalidad Raft en este fichero
//

package raft

//
// API
// ===
// Este es el API que vuestra implementación debe exportar
//
// nodoRaft = NuevoNodo(...)
//   Crear un nuevo servidor del grupo de elección.
//
// nodoRaft.Para()
//   Solicitar la parado de un servidor
//
// nodo.ObtenerEstado() (yo, mandato, esLider)
//   Solicitar a un nodo de elección por "yo", su mandato en curso,
//   y si piensa que es el msmo el lider
//
// nodoRaft.SometerOperacion(operacion interface()) (indice, mandato, esLider)

// type AplicaOperacion

import (
	"fmt"
	"io"
	"log"
	"os"

	//"crypto/rand"
	"math/rand"
	"sync"
	"time"

	//"net/rpc"

	"raft/internal/comun/rpctimeout"
)

const (
	// Constante para fijar valor entero no inicializado
	IntNOINICIALIZADO = -1

	//  false deshabilita por completo los logs de depuracion
	// Aseguraros de poner kEnableDebugLogs a false antes de la entrega
	kEnableDebugLogs = true

	// Poner a true para logear a stdout en lugar de a fichero
	kLogToStdout = true

	// Cambiar esto para salida de logs en un directorio diferente
	kLogOutputDir = "./logs_raft/"
)

type TipoOperacion struct {
	Operacion string // La operaciones posibles son "leer" y "escribir"
	Clave     string
	Valor     string // en el caso de la lectura Valor = ""
}

// A medida que el nodo Raft conoce las operaciones de las  entradas de registro
// comprometidas, envía un AplicaOperacion, con cada una de ellas, al canal
// "canalAplicar" (funcion NuevoNodo) de la maquina de estados
type AplicaOperacion struct {
	Indice    int // en la entrada de registro
	Mandato   int
	Operacion TipoOperacion
}
type Entrada struct {
	Indice    int
	Mandato   int
	Operacion TipoOperacion
}

type State struct {
	MandatoActual           int
	HaVotadoA               int
	Log                     []Entrada
	IndiceMayorComprometido int
	IndiceMayorAplicado     int
	SiguienteIndice         []int // SiguienteIndice[0] = Siguiente entrada que se mandará al nodo 0
	IndiceUltimoConocido    []int // IndiceUltimoConocido[0] = Ultima entrada que el lider sabe que le ha llegado al nodo 0
}

// Tipo de dato Go que representa un solo nodo (réplica) de raft
type NodoRaft struct {
	Mux sync.Mutex // Mutex para proteger acceso a estado compartido
	// Host:Port de todos los nodos (réplicas) Raft, en mismo orden
	Nodos   []rpctimeout.HostPort
	Yo      int // indice de este nodos en campo array "nodos"
	IdLider int
	// Utilización opcional de este logger para depuración
	// Cada nodo Raft tiene su propio registro de trazas (logs)
	Logger *log.Logger

	Estado State

	CanalLatido           chan bool
	CanalSeguidor         chan bool
	CanalLider            chan bool
	CanalVotos            chan bool
	canalAplicarOperacion chan int
	Memoria               map[string]string

	Rol string
}

func Make(val, len int) []int {
	v := make([]int, len)
	for i := range v {
		v[i] = val
	}
	return v
}

// Creacion de un nuevo nodo de eleccion
//
// Tabla de <Direccion IP:puerto> de cada nodo incluido a si mismo.
//
// <Direccion IP:puerto> de este nodo esta en nodos[yo]
//
// Todos los arrays nodos[] de los nodos tienen el mismo orden

// canalAplicar es un canal donde, en la practica 5, se recogerán las
// operaciones a aplicar a la máquina de estados. Se puede asumir que
// este canal se consumira de forma continúa.
//
// NuevoNodo() debe devolver resultado rápido, por lo que se deberían
// poner en marcha Gorutinas para trabajos de larga duracion
func NuevoNodo(nodos []rpctimeout.HostPort, yo int,
	canalAplicarOperacion chan AplicaOperacion) *NodoRaft {
	time.Sleep(5 * time.Second)

	nr := &NodoRaft{}
	nr.Nodos = nodos
	nr.Yo = yo
	nr.IdLider = -1

	if kEnableDebugLogs {
		nombreNodo := nodos[yo].Host() + "_" + nodos[yo].Port()
		fmt.Println("nombreNodo: ", nombreNodo)

		if kLogToStdout {
			nr.Logger = log.New(os.Stdout, nombreNodo+" -->> ",
				log.Lmicroseconds|log.Lshortfile)
		} else {
			err := os.MkdirAll(kLogOutputDir, os.ModePerm)
			if err != nil {
				panic(err.Error())
			}
			logOutputFile, err := os.OpenFile(
				fmt.Sprintf("%s/%s.txt", kLogOutputDir, nombreNodo),
				os.O_RDWR|os.O_CREATE|os.O_TRUNC,
				0755)
			if err != nil {
				panic(err.Error())
			}
			nr.Logger = log.New(logOutputFile,
				nombreNodo+" -> ", log.Lmicroseconds|log.Lshortfile)
		}
		nr.Logger.Println("logger initialized")
	} else {
		nr.Logger = log.New(io.Discard, "", 0)
	}

	nr.Estado.Log = make([]Entrada, 0)
	nr.CanalLatido = make(chan bool)
	nr.CanalSeguidor = make(chan bool)
	nr.CanalLider = make(chan bool)
	nr.CanalVotos = make(chan bool)
	nr.canalAplicarOperacion = make(chan int)

	nr.Estado.HaVotadoA = -1
	nr.Estado.MandatoActual = 0
	nr.Estado.IndiceMayorComprometido = -1
	nr.Estado.IndiceMayorAplicado = 0

	nr.Rol = "Seguidor"
	nr.Estado.SiguienteIndice = Make(0, len(nr.Nodos))
	nr.Estado.IndiceUltimoConocido = Make(-1, len(nr.Nodos))

	nr.Memoria = make(map[string]string)
	//fmt.Printf("Estado inicial: %+v\n", nr.Estado)
	go nr.MaquinaDeEstados()
	go func() {
		for indice := range nr.canalAplicarOperacion {
			nr.aplicarOperacion(indice)
		}
	}()
	return nr
}

// Metodo Para() utilizado cuando no se necesita mas al nodo
//
// Quizas interesante desactivar la salida de depuracion
// de este nodo
func (nr *NodoRaft) para() {
	go func() { time.Sleep(5 * time.Millisecond); os.Exit(0) }()
}

// Devuelve "yo", mandato en curso y si este nodo cree ser lider
//
// Primer valor devuelto es el indice de este  nodo Raft el el conjunto de nodos
// la operacion si consigue comprometerse.
// El segundo valor es el mandato en curso
// El tercer valor es true si el nodo cree ser el lider
// Cuarto valor es el lider, es el indice del líder si no es él
func (nr *NodoRaft) obtenerEstado() (int, int, bool, int) {
	nr.Mux.Lock()
	var yo int = nr.Yo
	var mandato int
	var esLider bool
	var idLider int

	mandato = nr.Estado.MandatoActual
	esLider = (nr.Rol == "Lider")

	if esLider {
		idLider = nr.Yo
	} else {
		idLider = nr.IdLider
	}
	nr.Mux.Unlock()
	return yo, mandato, esLider, idLider
}

// El servicio que utilice Raft (base de datos clave/valor, por ejemplo)
// Quiere buscar un acuerdo de posicion en registro para siguiente operacion
// solicitada por cliente.

// Si el nodo no es el lider, devolver falso
// Sino, comenzar la operacion de consenso sobre la operacion y devolver en
// cuanto se consiga
//
// No hay garantía que esta operación consiga comprometerse en una entrada de
// de registro, dado que el lider puede fallar y la entrada ser reemplazada
// en el futuro.
// Resultado de este método :
// - Primer valor devuelto es el indice del registro donde se va a colocar
// - la operacion si consigue comprometerse.
// - El segundo valor es el mandato en curso
// - El tercer valor es true si el nodo cree ser el lider
// - Cuarto valor es el lider, es el indice del líder si no es él
// - Quinto valor es el resultado de aplicar esta operación en máquina de estados
func (nr *NodoRaft) someterOperacion(operacion TipoOperacion) (int, int,
	bool, int, string) {
	indice := -1
	mandato := -1
	esLider := false
	idLider := -1
	valorADevolver := ""

	if nr.Rol != "Lider" {
		return indice, mandato, esLider, idLider, valorADevolver
	}

	if operacion.Operacion == "Lectura" {
		valorADevolver = nr.Memoria[operacion.Clave]
	} else {
		valorADevolver = operacion.Valor
	}
	nr.Logger.Printf("El log en este momento mide %d", len(nr.Estado.Log))
	indice = len(nr.Estado.Log)
	mandato = nr.Estado.MandatoActual
	esLider = true
	idLider = nr.Yo
	nr.Estado.Log = append(nr.Estado.Log, Entrada{Indice: indice, Mandato: mandato, Operacion: operacion})

	return indice, mandato, esLider, idLider, valorADevolver

}

// -----------------------------------------------------------------------
// LLAMADAS RPC al API
//
// Si no tenemos argumentos o respuesta estructura vacia (tamaño cero)
type Vacio struct{}

func (nr *NodoRaft) ParaNodo(args Vacio, reply *Vacio) error {
	defer nr.para()
	return nil
}

type EstadoParcial struct {
	Mandato int
	EsLider bool
	IdLider int
}

type EstadoRemoto struct {
	IdNodo int
	EstadoParcial
}

func (nr *NodoRaft) ObtenerEstadoNodo(args Vacio, reply *EstadoRemoto) error {
	reply.IdNodo, reply.Mandato, reply.EsLider, reply.IdLider = nr.obtenerEstado()
	return nil
}

type ResultadoRemoto struct {
	ValorADevolver string
	IndiceRegistro int
	EstadoParcial
}

func (nr *NodoRaft) SometerOperacionRaft(operacion TipoOperacion,
	reply *ResultadoRemoto) error {
	reply.IndiceRegistro, reply.Mandato, reply.EsLider,
		reply.IdLider, reply.ValorADevolver = nr.someterOperacion(operacion)
	return nil
}

// Definir una estructura para la respuesta RPC
type IndiceCommit struct {
	IndiceCommit int
}

// Método RPC para obtener el índice de compromiso de un nodo
func (nr *NodoRaft) ObtenerIndiceComprometido(args Vacio, reply *IndiceCommit) error {
	nr.Mux.Lock()
	defer nr.Mux.Unlock()
	reply.IndiceCommit = nr.Estado.IndiceMayorComprometido
	nr.Logger.Printf("ObtenerIndiceComprometido: Nodo %d devolviendo índice %d", nr.Yo, nr.Estado.IndiceMayorComprometido)
	return nil
}

// -----------------------------------------------------------------------
// LLAMADAS RPC protocolo RAFT
//
// Structura de ejemplo de argumentos de RPC PedirVoto.
//
// Recordar
// -----------
// Nombres de campos deben comenzar con letra mayuscula !
// Argumentos para la solicitud de voto
type ArgsPeticionVoto struct {
	MandatoCandidato              int // Mandato del candidato solicitando el voto
	IdCandidato                   int // ID del candidato solicitando el voto
	IndiceUltimaEntradaCandidato  int // Índice de la última entrada en el log del candidato
	MandatoUltimaEntradaCandidato int // Mandato de la última entrada en el log del candidato
}

// Structura de ejemplo de respuesta de RPC PedirVoto,
//
// Recordar
// -----------
// Nombres de campos deben comenzar con letra mayuscula !
// Respuesta a la solicitud de voto
type RespuestaPeticionVoto struct {
	IDNodo        int  // ID del nodo que responde
	MandatoPropio int  // Mandato del nodo que responde
	VotoRecibido  bool // Indicador de si el voto fue concedido
}

// Metodo para RPC PedirVoto
func (nr *NodoRaft) PedirVoto(peticion *ArgsPeticionVoto,
	reply *RespuestaPeticionVoto) error {

	// Vuestro codigo aqui
	nr.Mux.Lock()
	defer nr.Mux.Unlock()
	MandatoLog := 0
	LongLog := len(nr.Estado.Log)
	if LongLog > 0 {
		MandatoLog = nr.Estado.Log[LongLog-1].Mandato
	}
	// Inicializamos la respuesta con el mandato actual del nodo

	// Verificación de la actualidad del log del candidato
	logActualizado := false
	if peticion.MandatoUltimaEntradaCandidato > MandatoLog {
		logActualizado = true // El log del candidato está en un mandato más reciente
	} else if peticion.MandatoUltimaEntradaCandidato == MandatoLog {
		if peticion.IndiceUltimaEntradaCandidato >= LongLog-1 {
			logActualizado = true // El candidato tiene el mismo mandato y al menos el mismo índice
		}
	}

	// Verificación del término del candidato y el estado de votación
	terminoActualizado := false
	if peticion.MandatoCandidato > nr.Estado.MandatoActual {
		terminoActualizado = true // El candidato tiene un término superior
	} else if peticion.MandatoCandidato == nr.Estado.MandatoActual {
		if nr.Estado.HaVotadoA == -1 || nr.Estado.HaVotadoA == peticion.IdCandidato {
			terminoActualizado = true // Mismo término y aún no ha votado, o ya votó por el candidato
		}
	}

	if logActualizado && terminoActualizado {
		nr.Estado.MandatoActual = peticion.MandatoCandidato
		nr.Estado.HaVotadoA = peticion.IdCandidato
		nr.Rol = "Seguidor"
		reply.VotoRecibido = true
		reply.MandatoPropio = nr.Estado.MandatoActual
		nr.CanalLatido <- true

	} else {
		reply.VotoRecibido = false
		reply.MandatoPropio = nr.Estado.MandatoActual
	}

	return nil
}

type ArgAppendEntries struct {
	MandatoLider       int
	IdLider            int
	IndiceLogAnterior  int
	MandatoLogAnterior int
	EntradasLog        []Entrada
	IndiceComprometido int
}

type Results struct {
	MandatoActual     int
	Exito             bool
	IndiceCoincidente int // Indice de la última entrada coincidente
}

// Metodo de tratamiento de llamadas RPC AppendEntries
func (nr *NodoRaft) AppendEntries(args *ArgAppendEntries, results *Results) error {
	// Verificar si el mandato del líder es mayor al mandato actual
	if args.MandatoLider > nr.Estado.MandatoActual {
		nr.Estado.MandatoActual = args.MandatoLider
		nr.Estado.HaVotadoA = -1 // Reiniciar el voto en el nuevo mandato
		nr.CanalLatido <- true
	}

	isLogOk := len(nr.Estado.Log)-1 >= args.IndiceLogAnterior
	if isLogOk && args.IndiceLogAnterior >= 0 {
		isLogOk = args.IndiceLogAnterior == nr.Estado.Log[args.IndiceLogAnterior].Mandato
	}
	// Procesar AppendEntries si <el mandato es el actual y el log está alineado
	if args.MandatoLider == nr.Estado.MandatoActual && isLogOk {
		nr.IdLider = args.IdLider
		nr.Rol = "Seguidor"
		args.IndiceLogAnterior++
		nr.ActualizarLog(args)
		nr.CanalLatido <- true
		results.Exito = true
		results.MandatoActual = nr.Estado.MandatoActual
		results.IndiceCoincidente = args.IndiceLogAnterior + len(args.EntradasLog) - 1
		//nr.Logger.Printf("Indice coincidente ahora es %d\n", results.IndiceCoincidente)
	} else {
		results.MandatoActual = nr.Estado.MandatoActual
		results.Exito = false
		results.IndiceCoincidente = -1
	}

	return nil
}

// Lógica de actualización del log
func (nr *NodoRaft) ActualizarLog(args *ArgAppendEntries) {

	// Verificar si el log contiene la entrada en IndiceLogAnterior con el mandato correcto
	if args.IndiceLogAnterior >= 0 && args.IndiceLogAnterior < len(nr.Estado.Log) {
		if nr.Estado.Log[args.IndiceLogAnterior].Mandato != args.MandatoLogAnterior {
			nr.Estado.Log = nr.Estado.Log[:args.IndiceLogAnterior]
		}
	} else if args.IndiceLogAnterior >= len(nr.Estado.Log) {
		nr.Estado.Log = nr.Estado.Log[:len(nr.Estado.Log)] //
	}

	// Añadir las nuevas entradas al log
	for i, entry := range args.EntradasLog {
		if args.IndiceLogAnterior+1+i < len(nr.Estado.Log) {
			nr.Estado.Log[args.IndiceLogAnterior+1+i] = entry
		} else {
			nr.Estado.Log = append(nr.Estado.Log, entry)
		}
	}
	if args.IndiceComprometido > nr.Estado.IndiceMayorComprometido {
		nr.Logger.Println("Aplicamos a la maquina de estados")
		next := nr.Estado.IndiceMayorComprometido + 1

		nr.Estado.IndiceMayorComprometido = min(args.IndiceComprometido, len(nr.Estado.Log)-1)

		for i := next; i <= nr.Estado.IndiceMayorComprometido; i++ {
			nr.canalAplicarOperacion <- i
		}
	}

}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// --------------------------------------------------------------------------
// ----- METODOS/FUNCIONES desde nodo Raft, como cliente, a otro nodo Raft
// --------------------------------------------------------------------------

// Ejemplo de código enviarPeticionVoto
//
// nodo int -- indice del servidor destino en nr.nodos[]
//
// args *RequestVoteArgs -- argumentos par la llamada RPC
//
// reply *RequestVoteReply -- respuesta RPC
//
// Los tipos de argumentos y respuesta pasados a CallTimeout deben ser
// los mismos que los argumentos declarados en el metodo de tratamiento
// de la llamada (incluido si son punteros)
//
// Si en la llamada RPC, la respuesta llega en un intervalo de tiempo,
// la funcion devuelve true, sino devuelve false
//
// la llamada RPC deberia tener un timeout adecuado.
//
// Un resultado falso podria ser causado por una replica caida,
// un servidor vivo que no es alcanzable (por problemas de red ?),
// una petición perdida, o una respuesta perdida
//
// Para problemas con funcionamiento de RPC, comprobar que la primera letra
// del nombre de todo los campos de la estructura (y sus subestructuras)
// pasadas como parametros en las llamadas RPC es una mayuscula,
// Y que la estructura de recuperacion de resultado sea un puntero a estructura
// y no la estructura misma.

func (nr *NodoRaft) enviarPeticionVoto(nodo int, args *ArgsPeticionVoto, respuesta *RespuestaPeticionVoto) bool {
	// Enviar la solicitud de voto al nodo especificado con un tiempo límite de 25ms
	err := nr.Nodos[nodo].CallTimeout("NodoRaft.PedirVoto", args, respuesta, 25*time.Millisecond)

	// Bloquear el acceso concurrente y procesar la respuesta
	nr.Mux.Lock()
	defer nr.Mux.Unlock()

	if err == nil { // Si la solicitud se ha enviado con éxito
		if respuesta.MandatoPropio == nr.Estado.MandatoActual && respuesta.VotoRecibido && nr.Rol == "Candidato" {
			nr.CanalVotos <- true
		} else if respuesta.MandatoPropio > nr.Estado.MandatoActual {
			nr.Estado.MandatoActual = respuesta.MandatoPropio
			nr.Estado.HaVotadoA = -1
			nr.CanalLatido <- true
		}
	} else {
	}

	return err == nil // Retorna true si no hubo error en la solicitud, false en caso contrario
}

// Método para iniciar una nueva elección
func (nr *NodoRaft) eleccion() {

	// Incrementar el mandato actual y votar por sí mismo
	nr.Estado.MandatoActual++
	nr.Estado.HaVotadoA = nr.Yo

	// Preparar los argumentos de la petición de voto
	args := nr.prepararArgsPeticionVoto()

	// Enviar peticiones de voto a otros nodos
	nr.enviarPeticionesVoto(args)
}

// Preparar los argumentos para la petición de voto
func (nr *NodoRaft) prepararArgsPeticionVoto() ArgsPeticionVoto {
	args := ArgsPeticionVoto{
		MandatoCandidato:              nr.Estado.MandatoActual,
		IdCandidato:                   nr.Yo,
		MandatoUltimaEntradaCandidato: 0,
		IndiceUltimaEntradaCandidato:  -1,
	}

	if len(nr.Estado.Log) > 0 {
		indiceUltimaEntrada := len(nr.Estado.Log) - 1
		args.IndiceUltimaEntradaCandidato = indiceUltimaEntrada
		args.MandatoUltimaEntradaCandidato = nr.Estado.Log[indiceUltimaEntrada].Mandato
	}

	return args
}

// Enviar peticiones de voto a todos los demás nodos
func (nr *NodoRaft) enviarPeticionesVoto(args ArgsPeticionVoto) {
	var nodoEnvio int
	for nodoID := range nr.Nodos {
		if nodoID != nr.Yo {
			nodoEnvio = nodoID
			go func() {
				var respuesta RespuestaPeticionVoto
				nr.enviarPeticionVoto(nodoEnvio, &args, &respuesta)
			}()
		}
	}
}

func (nr *NodoRaft) peticionLatido(nodo int, args *ArgAppendEntries, reply *Results) {
	err := nr.Nodos[nodo].CallTimeout("NodoRaft.AppendEntries", args, reply, 25*time.Millisecond)
	if err != nil {
		nr.Logger.Printf("Error en la petición de latido: %v", err)
		return
	}

	if reply.MandatoActual > nr.Estado.MandatoActual {
		nr.Estado.MandatoActual = reply.MandatoActual
		nr.Estado.HaVotadoA = -1
		nr.Rol = "Seguidor"
		return
	}

	if nr.Rol != "Lider" || nr.Estado.MandatoActual != reply.MandatoActual {
		nr.Logger.Printf("NO SOY LIDER O MI MANDATO NO COINCIDE")
		return
	}

	if reply.Exito {
		if len(args.EntradasLog) > 0 {
			nr.Logger.Printf("Recive RPC.AppendEntries: OK -> seguimos "+
				"NextIndex:%d -> NextIndex:%d y MatchIndex:%d -> MatchIndex:%d\n",
				nr.Estado.SiguienteIndice[nodo], reply.IndiceCoincidente+1,
				nr.Estado.IndiceUltimoConocido[nodo], reply.IndiceCoincidente)
		} else {
			//nr.Logger.Println("Recive RPC.AppendEntries: OK")
		}
		nr.Estado.SiguienteIndice[nodo] = reply.IndiceCoincidente + 1
		nr.Estado.IndiceUltimoConocido[nodo] = reply.IndiceCoincidente
		//nr.Logger.Printf("El siguiente indice de %d es %d\n", nodo, nr.Estado.SiguienteIndice[nodo])
		nr.CommitEntries()
	} else if nr.Estado.SiguienteIndice[nodo] > 0 {
		nr.Estado.SiguienteIndice[nodo]--

		if nr.Estado.SiguienteIndice[nodo] < 0 {
			return
		}
		args.IndiceLogAnterior = nr.Estado.SiguienteIndice[nodo] - 1
		if args.IndiceLogAnterior >= 0 && args.IndiceLogAnterior < len(nr.Estado.Log) {
			args.IndiceLogAnterior = nr.Estado.Log[args.IndiceLogAnterior].Mandato
		}
		args.EntradasLog = nr.Estado.Log[nr.Estado.SiguienteIndice[nodo]:]

	}
}

func (nr *NodoRaft) CommitEntries() {
	minAcks := len(nr.Nodos) / 2
	for i := nr.Estado.IndiceMayorComprometido + 1; i < len(nr.Estado.Log); i++ {
		acks := 1 // Contar el propio líder
		for j := range nr.Nodos {
			if j != nr.Yo && nr.Estado.IndiceUltimoConocido[j] >= i {
				acks++
			}
		}
		nr.Logger.Printf("ACKS valen %d. MinAcks vale %d", acks, minAcks)
		if acks > minAcks && nr.Estado.Log[i].Mandato <= nr.Estado.MandatoActual {
			nr.Estado.IndiceMayorComprometido = i
			nr.canalAplicarOperacion <- i
		}
	}

}

// Función para enviar nuevos latidos a otros nodos
func (nr *NodoRaft) enviarLatido() {

	// Itera sobre los nodos y configura las entradas específicas para cada nodo
	for nodo := range nr.Nodos {
		baseArgs := ArgAppendEntries{
			MandatoLider:       nr.Estado.MandatoActual,
			IdLider:            nr.Yo,
			IndiceComprometido: nr.Estado.IndiceMayorComprometido,
			IndiceLogAnterior:  -1,
			MandatoLogAnterior: 0,
			EntradasLog:        nil,
		}

		if nodo == nr.Yo {
			continue
		}

		// Clona los argumentos base y personaliza por nodo
		args := baseArgs
		args.IndiceLogAnterior = nr.Estado.SiguienteIndice[nodo] - 1
		if args.IndiceLogAnterior >= 0 && args.IndiceLogAnterior < len(nr.Estado.Log) {
			args.MandatoLogAnterior = nr.Estado.Log[args.IndiceLogAnterior].Mandato
		}
		// Asigna las entradas del log si el índice está dentro del rango
		if nr.Estado.SiguienteIndice[nodo] < len(nr.Estado.Log) {
			args.EntradasLog = nr.Estado.Log[nr.Estado.SiguienteIndice[nodo]:] //Convert to tipoOperacion
		} else {

			args.EntradasLog = nil
		}

		go func(destino int, argumentos ArgAppendEntries) {
			var respuesta Results
			nr.peticionLatido(destino, &argumentos, &respuesta)
		}(nodo, args)
	}

}

func (nr *NodoRaft) MaquinaDeEstados() {
	for {
		nr.logEstadoActual()
		switch nr.Rol {
		case "Seguidor":
			nr.Logger.Println("Soy seguidor")
			nr.bucleSeguidor()
		case "Candidato":
			nr.Logger.Println("Soy candidato")
			nr.bucleCandidato()
		case "Lider":
			nr.Logger.Println("Soy lider")
			nr.bucleLider()
		}
	}
}

// Función auxiliar para loggear el estado actual
func (nr *NodoRaft) logEstadoActual() {
	//nr.Logger.Printf("Soy %s y mi mandato es %d\n", nr.Rol, nr.Estado.MandatoActual)
}

func tiempoEleccion(min, max int) int {
	return rand.Intn(max) + min
}

func (nr *NodoRaft) bucleSeguidor() {
	// Inicializar el temporizador con un tiempo de elección entre 150ms y 500ms
	temporizador := time.NewTimer(time.Duration(tiempoEleccion(100, 500)) * time.Millisecond)

	defer temporizador.Stop()

	for nr.Rol == "Seguidor" {
		select {
		case <-nr.CanalLatido:
			// Reiniciar el temporizador con un nuevo tiempo de elección entre 150ms y 500ms
			temporizador.Reset(time.Duration(tiempoEleccion(100, 500)) * time.Millisecond)
		case <-temporizador.C:
			// Cambiar el rol a "Candidato" y llamar a bucleCandidato
			nr.Rol = "Candidato"
			nr.IdLider = -1
			return
		}
	}
}

/**
 * @brief Caso de que el nodo sea un candidato.
 */
func (nr *NodoRaft) bucleCandidato() {

	votosRecibidos := 1
	for nr.Rol == "Candidato" {

		// Temporizador de elección entre 150ms y 300ms
		electionTimeout := time.Duration(tiempoEleccion(100, 500)) * time.Millisecond
		temporizador := time.NewTimer(electionTimeout)
		defer temporizador.Stop()
		nr.eleccion()

		select {
		case <-nr.CanalVotos:
			votosRecibidos++
			temporizador.Reset(time.Duration(tiempoEleccion(100, 500)) * time.Millisecond)
			if votosRecibidos > len(nr.Nodos)/2 {
				nr.Rol = "Lider"
				nr.IdLider = nr.Yo
				nr.Logger.Println("Soy lider")
				nr.bucleLider()
			}
			temporizador.Stop()
		case <-temporizador.C:
			// Expiró el temporizador de elección
			votosRecibidos = 1
			// Detener temporizador y comenzar nueva elección
			//temporizador.Stop()
			//temporizador.Reset(time.Duration(tiempoEleccion(150, 300)) * time.Millisecond)
		case <-nr.CanalLatido:
			nr.Rol = "Seguidor"

		}

	}
}

/**
 * @brief Caso de que el nodo sea un líder.
 */
func (nr *NodoRaft) bucleLider() {

	temporizador := time.NewTicker(time.Duration(25) * time.Millisecond)
	defer temporizador.Stop()

	nr.IdLider = nr.Yo
	nr.Estado.SiguienteIndice = make([]int, len(nr.Nodos))
	for i := range nr.Estado.SiguienteIndice {
		nr.Estado.SiguienteIndice[i] = len(nr.Estado.Log)
	}
	nr.Estado.IndiceUltimoConocido = make([]int, len(nr.Nodos))
	for i := range nr.Estado.IndiceUltimoConocido {
		nr.Estado.IndiceUltimoConocido[i] = -1
	}

	for nr.Rol == "Lider" {
		temporizador = time.NewTicker(time.Duration(25) * time.Millisecond)
		nr.enviarLatido()

		select {
		case <-temporizador.C:
		case <-nr.CanalLatido:
			nr.Rol = "Seguidor"
		}
	}
}

func (nr *NodoRaft) aplicarOperacion(indice int) {
	nr.Mux.Lock()
	defer nr.Mux.Unlock()

	operacion := nr.Estado.Log[indice].Operacion
	nr.Memoria[operacion.Clave] = operacion.Valor
	nr.Estado.IndiceMayorAplicado++

	nr.Logger.Printf("Aplicada operación: Clave=%s, Valor=%v", operacion.Clave, operacion.Valor)
	nr.Logger.Printf("ESTADO NODO: CI=%d, Log=%v, Almacen=%v",
		nr.Estado.IndiceMayorComprometido, nr.Estado.Log, nr.Memoria)
}
