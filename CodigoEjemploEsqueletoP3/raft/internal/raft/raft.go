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

	CanalLatido   chan bool
	CanalSeguidor chan bool
	CanalLider    chan bool
	CanalVotos    chan bool

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

	fmt.Println("Creando nuevo nodo Raft")

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
	nr.Logger.Println("Iniciando configuración de estado y canales")

	nr.Estado.Log = make([]Entrada, 0)
	nr.Logger.Println("Estado inicializado1")
	nr.CanalLatido = make(chan bool)
	nr.Logger.Println("Estado inicializado2")
	nr.CanalSeguidor = make(chan bool)
	nr.Logger.Println("Estado inicializado3")
	nr.CanalLider = make(chan bool)
	nr.Logger.Println("Estado inicializado4")
	nr.CanalVotos = make(chan bool)
	nr.Logger.Println("Estado inicializado5")

	nr.Estado.HaVotadoA = -1
	nr.Estado.MandatoActual = 0
	nr.Estado.IndiceMayorComprometido = 0
	nr.Estado.IndiceMayorAplicado = 0

	nr.Rol = "Seguidor"
	nr.Estado.SiguienteIndice = Make(0, len(nr.Nodos))
	nr.Estado.IndiceUltimoConocido = Make(-1, len(nr.Nodos))

	fmt.Println("Log inicializado y nodo creado con éxito")
	//fmt.Printf("Estado inicial: %+v\n", nr.Estado)
	fmt.Printf("Rol inicial: %s\n", nr.Rol)
	go nr.bucle()
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
	nr.Logger.Println("Obteniendo estado")
	nr.Mux.Lock()
	var yo int = nr.Yo
	var mandato int
	var esLider bool
	var idLider int
	nr.Logger.Println(fmt.Sprintf("Estado obtenido, soy %d y soy %s. El ID lider es %d", nr.Yo, nr.Rol, nr.IdLider))

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
	esLider := nr.Yo == nr.IdLider
	idLider := -1
	valorADevolver := ""

	if esLider {
		indice = nr.Estado.IndiceMayorComprometido //+1?
		mandato = nr.Estado.MandatoActual
		entry := Entrada{indice, mandato, operacion}

		nr.Logger.Printf("(%d, %d, %s, %s, %s)", entry.Indice, entry.Mandato,
			entry.Operacion.Operacion, entry.Operacion.Clave,
			entry.Operacion.Valor)

		var results Results
		confirmados := 0

		for i := 0; i < len(nr.Nodos); i++ {
			if i != nr.Yo {
				nr.Nodos[i].CallTimeout(
					"NodoRaft.AppendEntries",
					ArgAppendEntries{
						MandatoLider:       mandato,
						IdLider:            nr.Yo,
						IndiceLogAnterior:  indice,
						MandatoLogAnterior: entry.Mandato,
						EntradasLog:        []Entrada{entry},
						IndiceComprometido: nr.Estado.IndiceMayorComprometido,
					},
					&results,
					20*time.Millisecond,
				)
			}

			if results.Exito {
				confirmados++
			}
		}

		if confirmados > len(nr.Nodos)/2 {
			nr.Estado.IndiceMayorComprometido++
		}

		idLider = nr.Yo
	}

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
	nr.Logger.Println("Obteniendo estado nodo")
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
	logOk := false
	if peticion.MandatoUltimaEntradaCandidato > MandatoLog {
		logOk = true // El log del candidato está en un mandato más reciente
	} else if peticion.MandatoUltimaEntradaCandidato == MandatoLog {
		if peticion.IndiceUltimaEntradaCandidato >= LongLog-1 {
			logOk = true // El candidato tiene el mismo mandato y al menos el mismo índice
		}
	}

	// Verificación del término del candidato y el estado de votación
	termOk := false
	if peticion.MandatoCandidato > nr.Estado.MandatoActual {
		termOk = true // El candidato tiene un término superior
	} else if peticion.MandatoCandidato == nr.Estado.MandatoActual {
		if nr.Estado.HaVotadoA == -1 || nr.Estado.HaVotadoA == peticion.IdCandidato {
			termOk = true // Mismo término y aún no ha votado, o ya votó por el candidato
		}
	}

	if logOk && termOk {
		nr.Logger.Printf("RECV RPC.PedirVoto: Voto concedido a %d\n",
			peticion.IdCandidato)
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
	nr.Logger.Printf("AppendEntries recibido de %d", args.IdLider)

	// Verificar si el mandato del líder es mayor al mandato actual
	if args.MandatoLider > nr.Estado.MandatoActual {
		nr.Logger.Println("RECV RPC.AppendEntries: Voy rezagado")
		nr.Estado.MandatoActual = args.MandatoLider
		nr.Estado.HaVotadoA = -1 // Reiniciar el voto en el nuevo mandato
		nr.CanalLatido <- true
	}

	// Procesar AppendEntries si el mandato es el actual y el log está alineado
	if args.MandatoLider == nr.Estado.MandatoActual && nr.isLogValido(args) {

		nr.IdLider = args.IdLider
		results.MandatoActual = nr.Estado.MandatoActual
		nr.Rol = "Seguidor"
		results.Exito = true
		results.IndiceCoincidente = args.IndiceLogAnterior + len(args.EntradasLog)

		nr.ActualizarLog(args)
		nr.Rol = "Seguidor"
		nr.CanalLatido <- true
	} else {
		nr.Logger.Printf("Error append entries. Mi mandato es %d y el del lider es %d", nr.Estado.MandatoActual, args.MandatoLider)
		results.MandatoActual = nr.Estado.MandatoActual
		results.Exito = false
		results.IndiceCoincidente = -1
	}

	return nil
}

// Verificar si el log contiene la entrada en IndiceLogAnterior con el mandato correcto
func (nr *NodoRaft) isLogValido(args *ArgAppendEntries) bool {
	if args.IndiceLogAnterior >= 0 && args.IndiceLogAnterior <= len(nr.Estado.Log)-1 {
		nr.Logger.Println("ENTRÉEEEE")
		return args.MandatoLogAnterior == nr.Estado.Log[args.IndiceLogAnterior].Mandato
	}
	nr.Logger.Printf("NO HE ENTRADO. IndiceLogAnterior es %d y el ttamaño del log de nr es %d\n", args.IndiceLogAnterior, len(nr.Estado.Log))
	return false
}

// Lógica de actualización del log
func (nr *NodoRaft) ActualizarLog(args *ArgAppendEntries) {

	// Verificar si el log contiene la entrada en IndiceLogAnterior con el mandato correcto
	if args.IndiceLogAnterior >= 0 && args.IndiceLogAnterior < len(nr.Estado.Log) {
		if nr.Estado.Log[args.IndiceLogAnterior].Mandato != args.MandatoLogAnterior {
			nr.Logger.Println("Log inconsistente, truncando...")
			nr.Estado.Log = nr.Estado.Log[:args.IndiceLogAnterior]
		}
	} else if args.IndiceLogAnterior >= len(nr.Estado.Log) {
		nr.Logger.Println("IndiceLogAnterior fuera de rango, truncando...")
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

	// Actualizar el índice mayor comprometido si es necesario
	if args.IndiceComprometido > nr.Estado.IndiceMayorComprometido {
		nr.Estado.IndiceMayorComprometido = min(args.IndiceComprometido, len(nr.Estado.Log)-1)
	}

	nr.Logger.Println("Log actualizado:", nr.Estado.Log)
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
	nr.Logger.Printf("RPC.PedirVoto: Solicitud de Voto [%d]{%v} -> Respuesta [%d]{%v}", nr.Yo, args, nodo, respuesta)

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
		nr.Logger.Println("Error en la solicitud de envío de solicitud")
	}

	return err == nil // Retorna true si no hubo error en la solicitud, false en caso contrario
}

// Método para iniciar una nueva elección
func (nr *NodoRaft) eleccion() {
	nr.Logger.Printf("Nueva elección")

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

	if nr.Rol != "Líder" || nr.Estado.MandatoActual != reply.MandatoActual {
		return
	}

	if reply.Exito {
		nr.Estado.SiguienteIndice[nodo] = reply.IndiceCoincidente + 1
		nr.Estado.IndiceUltimoConocido[nodo] = reply.IndiceCoincidente
	} else if nr.Estado.SiguienteIndice[nodo] > 0 {
		nr.Estado.SiguienteIndice[nodo]--
	}
}

// Función para enviar nuevos latidos a otros nodos
func (nr *NodoRaft) enviarLatido() {
	nr.Logger.Println("Nuevo latido")
	args := ArgAppendEntries{
		MandatoLider:       nr.Estado.MandatoActual,
		IdLider:            nr.Yo,
		IndiceLogAnterior:  -1,
		MandatoLogAnterior: 0,
		EntradasLog:        nil,
		IndiceComprometido: nr.Estado.IndiceMayorComprometido,
	}

	for nodo := range nr.Nodos {
		if nodo != nr.Yo {
			var respuesta Results
			args.IndiceLogAnterior = nr.Estado.SiguienteIndice[nodo] - 1
			if args.IndiceLogAnterior > -1 {
				args.MandatoLogAnterior = nr.Estado.Log[args.IndiceLogAnterior].Mandato
			}
			if nr.Estado.SiguienteIndice[nodo] < len(nr.Estado.Log) {
				args.EntradasLog = nr.Estado.Log[nr.Estado.SiguienteIndice[nodo]:]
			}
			go nr.peticionLatido(nodo, &args, &respuesta)
		}
	}
}

// =============================================================================
// Maquina de estados
// =============================================================================
//                                                      timeout
//                                           ______
//                                          v      |
//   --------    timeout    -----------  recv majority votes   -----------
//  |Seguidor| ----------> | Candidato |--------------------> |  Lider   |
//   --------               -----------                        -----------
//        ^          higher term/ |                         higher term |
//         |            new leader |                                     |
//         |_______________________|____________________________________ |
// =============================================================================

func (nr *NodoRaft) bucle() {
	for {
		switch nr.Rol {
		case "Seguidor":
			nr.Logger.Printf("ESTADO: Seguidor | Mandato %d\n", nr.Estado.MandatoActual)
			nr.bucleSeguidor()
		case "Candidato":
			nr.Logger.Printf("ESTADO: Candidato | Mandato %d\n", nr.Estado.MandatoActual+1)
			nr.bucleCandidato()
		case "Lider":
			nr.Logger.Printf("ESTADO: Lider | Mandato %d\n", nr.Estado.MandatoActual)
			nr.bucleLider()
		}
	}
}

/**
 * @brief Caso de que el nodo sea un seguidor.
 */

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
	startTime := time.Now()
	totalElectionTimeout := 2500 * time.Millisecond

	votosRecibidos := 1
	for nr.Rol == "Candidato" {

		// Temporizador de elección entre 150ms y 300ms
		electionTimeout := time.Duration(tiempoEleccion(100, 500)) * time.Millisecond
		temporizador := time.NewTimer(electionTimeout)
		nr.eleccion()

		select {
		case <-nr.CanalVotos:
			votosRecibidos++
			temporizador.Reset(time.Duration(tiempoEleccion(100, 500)) * time.Millisecond)
			if votosRecibidos > len(nr.Nodos)/2 {
				nr.Rol = "Lider"
				nr.IdLider = nr.Yo
				nr.bucleLider()
			}
			temporizador.Stop()
		case <-temporizador.C:
			// Expiró el temporizador de elección
			if time.Since(startTime) > totalElectionTimeout {
				fmt.Println("Error fatal: No se pudo elegir líder en 2.5 segundos")
				nr.Rol = "Error"
				temporizador.Stop()

			}
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
	nr.Logger.Printf("Inicio Lider: IndiceComprometido:%d Log: %v\n",
		nr.Estado.IndiceMayorComprometido, nr.Estado.Log)

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
