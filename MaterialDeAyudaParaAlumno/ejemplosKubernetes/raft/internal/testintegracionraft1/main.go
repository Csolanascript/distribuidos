package testintegracionraft1

import (
	"fmt"
	"raft/internal/comun/check"
	"sync"

	//"log"
	//"crypto/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"raft/internal/comun/rpctimeout"
	"raft/internal/raft"
)

const (
	//nodos replicas
	REPLICA1 = "127.0.0.1:29987"
	REPLICA2 = "127.0.0.1:29888"
	REPLICA3 = "127.0.0.1:29889"
	//REPLICA1 = "192.168.3.17:29298"
	//REPLICA2 = "192.168.3.18:29299"
	//REPLICA3 = "192.168.3.19:29291"
	// paquete main de ejecutables relativos a directorio raiz de modulo
	EXECREPLICA = "cmd/srvraft/main.go"

	// comando completo a ejecutar en máquinas remota con ssh. Ejemplo :
	// 				cd $HOME/raft; go run cmd/srvraft/main.go 127.0.0.1:29001
)

// PATH de los ejecutables de modulo golang de servicio Raft
var cwd, _ = os.Getwd()
var PATH string = filepath.Dir(filepath.Dir(cwd))

// go run cmd/srvraft/main.go 0 127.0.0.1:29001 127.0.0.1:29002 127.0.0.1:29003
var EXECREPLICACMD string = "cd " + PATH + ";  /usr/local/go/bin/go run " + EXECREPLICA

//////////////////////////////////////////////////////////////////////////////
///////////////////////			 FUNCIONES TEST
/////////////////////////////////////////////////////////////////////////////

// TEST primer rango
func TestPrimerasPruebas(t *testing.T) { // (m *testing.M) {
	// <setup code>
	// Crear canal de resultados de ejecuciones ssh en maquinas remotas

	dns := "raft-service.default.svc.cluster.local"
	name := "raft"
	puerto := "6000"
	var direcciones []string
	for i := 0; i < 3; i++ {
		nodo := name + "-" + strconv.Itoa(i) + "." + dns + ":" + puerto
		direcciones = append(direcciones, nodo)
	}
	var nodos []rpctimeout.HostPort
	for _, endPoint := range direcciones {
		nodos = append(nodos, rpctimeout.HostPort(endPoint))
	}
	time.Sleep(10 * time.Second)
	cfg := makeCfgDespliegue(t,
		3,
		[]string{REPLICA1, REPLICA2, REPLICA3},
		[]bool{true, true, true})

	// tear down code
	// eliminar procesos en máquinas remotas
	defer cfg.stop()

	// Run test sequence

	// Test1 : No debería haber ningun primario, si SV no ha recibido aún latidos
	t.Run("T1:soloArranqueYparada",
		func(t *testing.T) { cfg.soloArranqueYparadaTest1(t) })

	// Test2 : No debería haber ningun primario, si SV no ha recibido aún latidos
	t.Run("T2:ElegirPrimerLider",
		func(t *testing.T) { cfg.elegirPrimerLiderTest2(t) })

	// Test3: tenemos el primer primario correcto
	t.Run("T3:FalloAnteriorElegirNuevoLider",
		func(t *testing.T) { cfg.falloAnteriorElegirNuevoLiderTest3(t) })

	// Test4: Tres operaciones comprometidas en configuración estable
	t.Run("T4:tresOperacionesComprometidasEstable",
		func(t *testing.T) { cfg.tresOperacionesComprometidasEstable(t) })

	t.Run("T5:AcuerdoEntradasSinSeguidor",
		func(t *testing.T) { cfg.AcuerdoEntradasSinSeguidor(t) })

	t.Run("T6:NoAcuerdoEntradasSinSeguidores",
		func(t *testing.T) { cfg.NoAcuerdoEntradasSinSeguidores(t) })

	t.Run("T7:SometerVariasOperaciones",
		func(t *testing.T) { cfg.SometerVariasOperaciones(t) })
}

// ---------------------------------------------------------------------
//
// Canal de resultados de ejecución de comandos ssh remotos
type canalResultados chan string

func (cr canalResultados) stop() {
	close(cr)

	// Leer las salidas obtenidos de los comandos ssh ejecutados
	for s := range cr {
		fmt.Println(s)
	}
}

// ---------------------------------------------------------------------
// Operativa en configuracion de despliegue y pruebas asociadas
type configDespliegue struct {
	t           *testing.T
	conectados  []bool
	numReplicas int
	nodosRaft   []rpctimeout.HostPort
	cr          canalResultados
}

// Crear una configuracion de despliegue
func makeCfgDespliegue(t *testing.T, n int, nodosraft []string,
	conectados []bool) *configDespliegue {
	cfg := &configDespliegue{}
	cfg.t = t
	cfg.conectados = conectados
	cfg.numReplicas = n
	cfg.nodosRaft = rpctimeout.StringArrayToHostPortArray(nodosraft)
	cfg.cr = make(canalResultados, 2000)

	return cfg
}

func (cfg *configDespliegue) stop() {
	//cfg.stopDistributedProcesses()

	time.Sleep(50 * time.Millisecond)

	cfg.cr.stop()
}

// --------------------------------------------------------------------------
// FUNCIONES DE SUBTESTS

// Se ponen en marcha replicas - 3 NODOS RAFT
func (cfg *configDespliegue) soloArranqueYparadaTest1(t *testing.T) {
	t.Skip("SKIPPED soloArranqueYparadaTest1")

	fmt.Println(t.Name(), ".....................")

	cfg.t = t // Actualizar la estructura de datos de tests para errores

	// Poner en marcha replicas en remoto con un tiempo de espera incluido
	cfg.startDistributedProcesses()

	// Comprobar estado replica 0
	cfg.comprobarEstadoRemoto(0)

	// Comprobar estado replica 1
	cfg.comprobarEstadoRemoto(1)

	// Comprobar estado replica 2
	cfg.comprobarEstadoRemoto(2)

	// Parar réplicas almacenamiento en remoto
	cfg.stopDistributedProcesses()

	fmt.Println(".............", t.Name(), "Superado")
}

// Primer lider en marcha - 3 NODOS RAFT
func (cfg *configDespliegue) elegirPrimerLiderTest2(t *testing.T) {
	//t.Skip("SKIPPED ElegirPrimerLiderTest2")

	fmt.Println(t.Name(), ".....................")

	cfg.startDistributedProcesses()

	// Se ha elegido lider ?
	fmt.Printf("Probando lider en curso\n")
	leader := cfg.pruebaUnLider(3)
	fmt.Printf("El leader es el nodo %d\n", leader)

	// Parar réplicas alamcenamiento en remoto
	cfg.stopDistributedProcesses() // Parametros

	fmt.Println(".............", t.Name(), "Superado")
}

// Fallo de un primer lider y reeleccion de uno nuevo - 3 NODOS RAFT
func (cfg *configDespliegue) falloAnteriorElegirNuevoLiderTest3(t *testing.T) {
	//t.Skip("SKIPPED FalloAnteriorElegirNuevoLiderTest3")

	fmt.Println(t.Name(), ".....................")

	// Iniciar los procesos distribuidos
	cfg.startDistributedProcesses()

	// Encontrar el líder actual
	lider := cfg.pruebaUnLider(3)
	fmt.Printf("El líder es el nodo %d\n", lider)

	// Detener el nodo líder
	fmt.Printf("Se detiene el nodo líder %d\n", lider)
	cfg.pararLider(lider)

	// Esperar a que se elija un nuevo líder
	nuevoLider := cfg.esperarNuevoLider(lider)
	fmt.Printf("El nuevo líder es el nodo %d\n", nuevoLider)

	// Detener todos los procesos distribuidos
	cfg.stopDistributedProcesses()

	fmt.Println(".............", t.Name(), "Superado")

}

// 3 operaciones comprometidas con situacion estable y sin fallos - 3 NODOS RAFT
func (cfg *configDespliegue) tresOperacionesComprometidasEstable(t *testing.T) {
	//t.Skip("SKIPPED tresOperacionesComprometidasEstable")
	fmt.Println(t.Name(), ".....................")

	cfg.startDistributedProcesses()

	idLeader := cfg.pruebaUnLider(3)
	cfg.comprobarOperacion(idLeader, 0, "leer", "", "")
	cfg.comprobarOperacion(idLeader, 1, "escribir", "", "pepitosabemuchodelinux")
	cfg.comprobarOperacion(idLeader, 2, "leer", "", "pepitosabemuchodelinux")

	// Parar réplicas almacenamiento en remoto
	cfg.stopDistributedProcesses() //parametros
	fmt.Println(".............", t.Name(), "Superado")

}

func (cfg *configDespliegue) AcuerdoEntradasSinSeguidor(t *testing.T) {
	//t.Skip("SKIPPED AcuerdoEntradasSinSeguidor")
	fmt.Println("Iniciando Test 5: AcuerdoEntradasSinSeguidor")

	cfg.startDistributedProcesses()

	idLeader := cfg.pruebaUnLider(3)

	fmt.Printf("Arrancamos el lider %d\n", idLeader)

	nodoCaido := (idLeader + 1) % 3
	cfg.pararLider(nodoCaido)
	fmt.Printf("Paramos el nodo %d\n", nodoCaido)

	cfg.comprobarOperacion(idLeader, 0, "leer", "", "")
	cfg.comprobarOperacion(idLeader, 1, "escribir", "", "hola mundo")
	cfg.comprobarOperacion(idLeader, 2, "leer", "", "")

	cfg.iniciarNodo(nodoCaido)

	//Tiempo para iniciar nodo al no haber barrera
	time.Sleep(1000 * time.Millisecond)
	fmt.Printf("Despues del timeout, miramos la sincronicazion")

	ultimoIndiceComrometido, err := cfg.obtenerIndiceComprometidoRemoto(idLeader)
	if err != nil {
		cfg.t.Fatalf("Error al obtener índice comprometido del líder: %v", err)
	}

	for i := 0; i < 3; i++ {
		// Obtener el índice comprometido de cada nodo mediante la llamada RPC
		indice, err := cfg.obtenerIndiceComprometidoRemoto(i)
		if err != nil {
			cfg.t.Fatalf("Error al obtener índice comprometido del nodo %d: %v", i, err)
		}

		// Comprobar que el índice comprometido coincide con el del líder
		if indice != ultimoIndiceComrometido {
			cfg.t.Fatalf("El índice comprometido no coincide en nodo %d, esperado: %d, obtenido: %d", i, ultimoIndiceComrometido, indice)
		}
	}

	fmt.Println("Test 5: AcuerdoEntradasSinSeguidor superado")
	cfg.stopDistributedProcesses()
}

func (cfg *configDespliegue) NoAcuerdoEntradasSinSeguidores(t *testing.T) {
	//t.Skip("SKIPPED NoAcuerdoEntradasSinSeguidores")
	fmt.Println("Iniciando Test 6: NoAcuerdoEntradasSinSeguidores")

	cfg.startDistributedProcesses()
	idLeader := cfg.pruebaUnLider(3)
	fmt.Printf("El lider es el nodo %d\n", idLeader)

	nodo1 := (idLeader + 1) % 3
	nodo2 := (idLeader + 2) % 3

	cfg.pararLider(nodo1)
	cfg.pararLider(nodo2)
	fmt.Printf("Paramos nodos %d y %d. Ahora vamos a intentar comprometer las entradas sin ellos\n", nodo1, nodo2)

	indiceLider, _ := cfg.obtenerIndiceComprometidoRemoto(idLeader)
	fmt.Printf("Cuando paramos nodos, el indice comprometido del lider vale %d\n", indiceLider)

	cfg.comprobarOperacion(idLeader, 0, "leer", "", "")
	cfg.comprobarOperacion(idLeader, 1, "escribir", "", "pepitosabemuchodelinux")
	cfg.comprobarOperacion(idLeader, 2, "leer", "", "pepitosabemuchodelinux")
	fmt.Printf("Intentamos comprometer entradas, no deberia poder\n")

	indiceLider, _ = cfg.obtenerIndiceComprometidoRemoto(idLeader)

	if indiceLider != -1 {
		t.Fatalf("El líder no debería haber comprometido ninguna entrada todavía. El indice es %d", indiceLider)
	}

	cfg.iniciarNodo(nodo1)
	cfg.iniciarNodo(nodo2)
	time.Sleep(1000 * time.Millisecond)

	indiceLider, _ = cfg.obtenerIndiceComprometidoRemoto(idLeader)
	if indiceLider != 2 {
		t.Fatalf("El líder debería haber comprometido las 3 entradas. El indice es %d\n", indiceLider)
	}

	for i := 0; i < 3; i++ {
		commitIndex, _ := cfg.obtenerIndiceComprometidoRemoto(i)
		if commitIndex != 2 {
			t.Fatalf("El nodo %d no está sincronizado correctamente", i)
		}
	}

	fmt.Println("Test 6: AcuerdoEntradasSinSeguidores superado")
	cfg.stopDistributedProcesses()
}

// TODO EL TEST 7
func (cfg *configDespliegue) SometerVariasOperaciones(t *testing.T) {
	//t.Skip("SKIPPED SometerVariasOperaciones")
	fmt.Println("Iniciando Test 7: SometerVariasOperaciones")
	cfg.startDistributedProcesses()

	leader := cfg.pruebaUnLider(3)
	var wg sync.WaitGroup
	numOperaciones := 5
	confirmaciones := make(chan int, numOperaciones)

	for i := 0; i < numOperaciones; i++ {
		wg.Add(1)
		go func(operacionID int) {
			defer wg.Done()
			clave := fmt.Sprintf("clave%d", operacionID)
			valor := fmt.Sprintf("valor%d", operacionID)
			indice, _, _, idLider, _ := cfg.someterOperacion(leader, "escribir", clave, valor)

			if indice != operacionID || idLider != leader {
				fmt.Printf("Error al someter operación %d: se obtuvo índice %d\n", operacionID, indice)
				return
			}

			confirmaciones <- indice
		}(i)
	}

	wg.Wait()
	close(confirmaciones)

	time.Sleep(1000 * time.Millisecond)

	for nodo := 0; nodo < 3; nodo++ {
		commitIndex, err := cfg.obtenerIndiceComprometidoRemoto(nodo)
		if err != nil {
			t.Fatalf("Error al obtener índice comprometido del nodo %d: %v", nodo, err)
		}

		if commitIndex != numOperaciones-1 {
			t.Fatalf("El nodo %d no está sincronizado correctamente. Se esperaba commitIndex = %d, obtenido = %d", nodo, numOperaciones-1, commitIndex)
		}
	}

	fmt.Println("Test 7: SometerVariasOperaciones superado")
	cfg.stopDistributedProcesses()
}

// --------------------------------------------------------------------------
// FUNCIONES DE APOYO
// --------------------------------------------------------------------------
func (cfg *configDespliegue) someterOperacion(idLeader int, operation string,
	clave string, valor string) (int, int, bool, int, string) {
	operacion := raft.TipoOperacion{
		Operacion: operation,
		Clave:     clave,
		Valor:     valor,
	}
	var reply raft.ResultadoRemoto
	err := cfg.nodosRaft[idLeader].CallTimeout("NodoRaft.SometerOperacionRaft",
		operacion, &reply, 100*time.Millisecond)

	// Manejo de error en la llamada RPC
	if err != nil {
		check.CheckError(err, "Error en llamada RPC SometerOperacionRaft")
	}

	// Devuelve el resultado de la operación
	return reply.IndiceRegistro, reply.Mandato, reply.EsLider, reply.IdLider, reply.ValorADevolver
}

func (cfg *configDespliegue) comprobarOperacion(idLeader int, index int,
	operation string, clave string, valor string) {
	// Se somete la operación al nodo Raft
	indice, _, _, idLider, _ := cfg.someterOperacion(idLeader, operation, clave, valor)

	// Verifica si el índice de la operación sometida es igual al índice esperado
	if indice != index || idLider != idLeader {
		cfg.t.Fatalf("No se ha soemetido correctamente la operación con índice %d, se obtuvo índice %d", index, indice)
	}
}

func (cfg *configDespliegue) pararLider(idLeader int) {
	if idLeader < 0 || idLeader >= len(cfg.nodosRaft) {
		fmt.Printf("ID de líder no válido: %d\n", idLeader)
		return
	}

	var reply raft.Vacio
	endPoint := cfg.nodosRaft[idLeader]

	// Intentar detener el nodo líder
	err := endPoint.CallTimeout("NodoRaft.ParaNodo", raft.Vacio{}, &reply, 10*time.Millisecond)
	if err != nil {
		check.CheckError(err, "Error en llamada RPC Para nodo")
		return
	}

	// Actualizar el estado del nodo como desconectado
	cfg.conectados[idLeader] = false
	fmt.Printf("Nodo líder %d detenido exitosamente\n", idLeader)
}

// Comprobar que hay un solo lider
// probar varias veces si se necesitan reelecciones
func (cfg *configDespliegue) pruebaUnLider(numreplicas int) int {
	for iters := 0; iters < 10; iters++ {
		time.Sleep(100 * time.Millisecond)
		mapaLideres := make(map[int][]int)
		for i := 0; i < numreplicas; i++ {
			if cfg.conectados[i] {
				if _, mandato, eslider, _ := cfg.obtenerEstadoRemoto(i); eslider {
					mapaLideres[mandato] = append(mapaLideres[mandato], i)
				}
			}
		}

		ultimoMandatoConLider := -1
		for mandato, lideres := range mapaLideres {
			if len(lideres) > 1 {
				fmt.Printf("mandato %d tiene %d (>1) lideres",
					mandato, len(lideres))
				cfg.t.Fatalf("mandato %d tiene %d (>1) lideres",
					mandato, len(lideres))
			}
			if mandato > ultimoMandatoConLider {
				ultimoMandatoConLider = mandato
			}
		}

		if len(mapaLideres) != 0 {

			return mapaLideres[ultimoMandatoConLider][0] // Termina

		}
	}
	cfg.t.Fatalf("un lider esperado, ninguno obtenido")
	fmt.Printf("un lider esperado, ninguno obtenido")

	return -1 // Termina
}

func (cfg *configDespliegue) obtenerEstadoRemoto(
	indiceNodo int) (int, int, bool, int) {
	var reply raft.EstadoRemoto
	err := cfg.nodosRaft[indiceNodo].CallTimeout("NodoRaft.ObtenerEstadoNodo",
		raft.Vacio{}, &reply, 500*time.Millisecond)
	if err != nil {
		cfg.t.Logf("Error en llamada RPC ObtenerEstadoRemoto al nodo %d: %v", indiceNodo, err)
		return -1, -1, false, -1 // Devuelve valores por defecto en caso de error
	}

	return reply.IdNodo, reply.Mandato, reply.EsLider, reply.IdLider
}

// start  gestor de vistas; mapa de replicas y maquinas donde ubicarlos;
// y lista clientes (host:puerto)
func (cfg *configDespliegue) startDistributedProcesses() {
	//cfg.t.Log("Before start following distributed processes: ", cfg.nodosRaft)

	for i, endPoint := range cfg.nodosRaft {
		despliegue.ExecMutipleHosts(EXECREPLICACMD+
			" "+strconv.Itoa(i)+" "+
			rpctimeout.HostPortArrayToString(cfg.nodosRaft),
			[]string{endPoint.Host()}, cfg.cr)

		// dar tiempo para se establezcan las replicas
		time.Sleep(100 * time.Millisecond)
	}

	// aproximadamente 500 ms para cada arranque por ssh en portatil
	time.Sleep(1500 * time.Millisecond)
}

func (cfg *configDespliegue) stopDistributedProcesses() {
	var reply raft.Vacio

	for i, endPoint := range cfg.nodosRaft {
		if cfg.conectados[i] {
			err := endPoint.CallTimeout("NodoRaft.ParaNodo",
				raft.Vacio{}, &reply, 10*time.Millisecond)
			check.CheckError(err, "Error en llamada RPC Para nodo")
		} else {
			cfg.conectados[i] = true
		}
	}
}

// Comprobar estado remoto de un nodo con respecto a un estado prefijado
func (cfg *configDespliegue) comprobarEstadoRemoto(idNodoDeseado int) {
	idNodo, mandato, esLider, idLider := cfg.obtenerEstadoRemoto(idNodoDeseado)

	cfg.t.Log("Estado replica : ", idNodo, mandato, esLider, idLider, "\n")

	if idNodo != idNodoDeseado {
		cfg.t.Fatalf("Estado incorrecto en replica %d en subtest %s",
			idNodoDeseado, cfg.t.Name())
	}
}

// esperarNuevoLider espera hasta que se elija un nuevo líder, diferente del anterior.
func (cfg *configDespliegue) esperarNuevoLider(idLiderAnterior int) int {
	fmt.Println("Esperando un nuevo líder...")
	nuevoLider := idLiderAnterior

	for nuevoLider == idLiderAnterior {
		i := (idLiderAnterior + 1) % 3
		_, _, _, nuevoLider = cfg.obtenerEstadoRemoto(i)
		time.Sleep(100 * time.Millisecond)
		fmt.Println("Esperando un nuevo líder...")
	}

	return nuevoLider
}

func (cfg *configDespliegue) iniciarNodo(nodo int) {
	if nodo >= len(cfg.nodosRaft) || nodo < 0 {
		cfg.t.Fatalf("Error: El siguiente nodo no corresponde: %d", nodo)
	}

	nodoIniciado := cfg.nodosRaft[nodo]
	cfg.t.Logf("Nodo %d ha sido inciado en la máquina %s", nodo, nodoIniciado.Host())

	// Iniciamos el nodo nodoIniciado
	despliegue.ExecMutipleHosts(
		EXECREPLICACMD+" "+strconv.Itoa(nodo)+" "+rpctimeout.HostPortArrayToString(cfg.nodosRaft),
		[]string{nodoIniciado.Host()}, cfg.cr)

	// Hay que dar tiempo para que el nodo se inicie correctamente al no tener una barrera
	time.Sleep(1000 * time.Millisecond)

	var estadoRemoto raft.EstadoRemoto
	// Comprobamos que el nodo ha sido iniciado correctamente
	err := nodoIniciado.CallTimeout("NodoRaft.ObtenerEstadoNodo", raft.Vacio{}, &estadoRemoto, 500*time.Millisecond)

	if err != nil {
		cfg.t.Fatalf("Error: El nodo %d no se ha iniciado correctamente: %v", nodo, err)
	} else {
		cfg.t.Logf("Nodo %d arrancado correctamente.",
			nodo)
		cfg.conectados[nodo] = true
	}
}

func (cfg *configDespliegue) obtenerIndiceComprometidoRemoto(nodo int) (int, error) {
	var reply raft.IndiceCommit
	err := cfg.nodosRaft[nodo].CallTimeout("NodoRaft.ObtenerIndiceComprometido",
		raft.Vacio{}, &reply, 100*time.Millisecond)
	if err != nil {
		fmt.Printf("Error en ObtenerIndiceComprometido: %v\n", err)
		return -1, err
	}

	return reply.IndiceCommit, nil
}
