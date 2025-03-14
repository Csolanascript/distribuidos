package ra

import (
	"fmt"
	"practica2/ms"
	"sync"
)

type Request struct {
	Clock []int
	Pid   int
	op    int
}

type Reply struct {
	Clock []int
}

type RASharedDB struct {
	OutRepCnt int
	ReqCS     bool
	RepDefd   []bool
	ms        *ms.MessageSystem
	done      chan bool
	chrep     chan bool
	Mutex     *sync.Mutex // Mutex para proteger concurrencia sobre las variables
	op        int
	op_matrix [][]bool
	Clock     []int
}

// Constructor para inicializar la estructura RASharedDB
func New(me int, usersFile string, operacion int) *RASharedDB {
	fmt.Println("RASharedDB initizes")
	messageTypes := []ms.Message{Request{}, Reply{}}
	msgs := ms.New(me, usersFile, messageTypes)
	numProcesses := len(msgs.Peers()) // Asumimos que la cantidad de procesos es igual al número de peers
	Aux := make([]int, numProcesses)
	for i := 0; i < len(msgs.Peers()); i++ {
		Aux[i] = 0
	}
	ra := &RASharedDB{
		OutRepCnt: 0,
		ReqCS:     false,
		RepDefd:   make([]bool, numProcesses),
		ms:        &msgs,
		done:      make(chan bool),
		chrep:     make(chan bool, numProcesses-1),
		Mutex:     &sync.Mutex{},
		op:        operacion,
		op_matrix: [][]bool{
			{true, false},
			{false, false},
		},
		Clock: Aux,
	}
	fmt.Println("RASharedDB initizes correctly")
	return ra
}

// PreProtocol: Realiza el PreProtocol del algoritmo Ricart-Agrawala Generalizado
func (ra *RASharedDB) PreProtocol() {
	ra.Mutex.Lock()
	fmt.Println("Pide permiso para entrar en la sección crítica")
	ra.ReqCS = true
	ra.OutRepCnt = 0

	// Envía una solicitud a todos los procesos
	for i := 0; i < len(ra.ms.Peers()); i++ {
		if i != ra.ms.Me()-1 { // No se envía a sí mismo
			req := Request{Clock: ra.Clock, Pid: ra.ms.Me(), op: ra.op}
			ra.ms.Send(i+1, req)

		}
	}
	ra.Mutex.Unlock()

	fmt.Println("Se espera a que lleguen todas las respuestas")
	// Espera hasta recibir todas las respuestas
	for ra.OutRepCnt < len(ra.ms.Peers())-1 {
		<-ra.chrep
	}
	fmt.Println("Han llegado todas las respuestas")
	ra.OutRepCnt = 0
}

// PostProtocol: Realiza el PostProtocol del algoritmo Ricart-Agrawala Generalizado
func (ra *RASharedDB) PostProtocol() {
	ra.Mutex.Lock()
	ra.ReqCS = false
	fmt.Println("Voy a devolver reply a los diferidos pendientes")
	// Envía respuestas diferidas
	for i := 0; i < len(ra.ms.Peers()); i++ {
		if ra.RepDefd[i] {
			fmt.Println("Doy respuesta al diferido")
			reply := Reply{}
			ra.ms.Send(i+1, reply)
			ra.RepDefd[i] = false
		}
	}
	ra.Mutex.Unlock()
}

// Comprueba que un vector sea estrictamente menor que otro
func vectormenor(v1 []int, v2 []int) bool {
	for i := 0; i < len(v1); i++ {
		if v1[i] >= v2[i] {
			return false
		}
	}
	return true
}

// Comprueba que un vector sea estrictamente mayor que otro
func vectormayor(v1 []int, v2 []int) bool {
	for i := 0; i < len(v1); i++ {
		if v1[i] <= v2[i] {
			return false
		}
	}
	return true
}

// Manejador de solicitudes entrantes
func (ra *RASharedDB) HandleRequest(req Request) {
	ra.Mutex.Lock()
	defer ra.Mutex.Unlock()

	fmt.Println("Ha llegado una solicitud de sc")
	// Determina si debe enviar un Reply de inmediato o diferirlo
	if ra.ReqCS && ((vectormenor(ra.Clock, req.Clock) || (!vectormenor(ra.Clock, req.Clock) && !vectormayor(ra.Clock, req.Clock) && ra.ms.Me() < req.Pid)) && !ra.op_matrix[ra.op][req.op]) {
		fmt.Println(ra.op_matrix[ra.op][req.op])
		// Diferir respuesta
		ra.RepDefd[req.Pid-1] = true
		fmt.Println("Se ha diferido la respuesta")

	} else {
		// Enviar respuesta inmediata
		reply := Reply{}
		ra.ms.Send(req.Pid, reply)
		fmt.Println("Se ha enviado la respuesta")
	}
	ra.Clock[ra.ms.Me()-1]++
	for i := 0; i < len(ra.Clock); i++ {
		if req.Clock[i] > ra.Clock[i] {
			ra.Clock[i] = req.Clock[i]
		}
	}

}

// Manejador de respuestas entrantes
func (ra *RASharedDB) HandleReply() {
	ra.Mutex.Lock()
	defer ra.Mutex.Unlock()
	ra.OutRepCnt++
	ra.chrep <- true
	fmt.Println("Ha llegado una respuesta")
	// Señaliza que ha recibido una respuesta

}

// Función para detener el sistema
func (ra *RASharedDB) Stop() {
	ra.ms.Stop()
	ra.done <- true
}

// Función que escucha y maneja mensajes recibidos
func (ra *RASharedDB) Listen() {
	fmt.Println("Escuchando mensajes")
	for {
		// Verifica si el canal `ra.done` ha recibido una señal para detenerse
		select {
		case <-ra.done:
			return
		default:
			// Recibe el mensaje desde la función Receive+
			msg := ra.ms.Receive()
			fmt.Println("Mensaje recibido")

			// Maneja el tipo de mensaje recibido
			switch m := msg.(type) {
			case Request:
				ra.HandleRequest(m)
			case Reply:
				ra.HandleReply()
			}
		}
	}
}
