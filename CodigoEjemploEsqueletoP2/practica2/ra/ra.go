package ra

import (
	"practica2/ms"
	"sync"
)

type Request struct {
	Clock []int
	Pid   int
	SeqNum int 
	op    int
}

type Reply struct{
	Clock []int
}

type RASharedDB struct {
	OurSeqNum int
	HigSeqNum int
	OutRepCnt int
	ReqCS     bool
	RepDefd   []bool
	ms        *ms.MessageSystem
	done      chan bool
	chrep     chan bool
	Mutex     *sync.Mutex // Mutex para proteger concurrencia sobre las variables
	op int
	op_matrix [][]bool
	Clock []int
}



// Constructor para inicializar la estructura RASharedDB
func New(me int, usersFile string, operacion int) *RASharedDB {
	messageTypes := []ms.Message{Request{}, Reply{}}
	msgs := ms.New(me, usersFile, messageTypes)
	numProcesses := len(msgs.Peers()) // Asumimos que la cantidad de procesos es igual al número de peers
	Aux := make([]int,len(msgs.Peers()))
	for i := 0; i < len(msgs.Peers()); i++ {
		Aux[i] = 0
	}
	ra := &RASharedDB{
		OurSeqNum: 0,
		HigSeqNum: 0,
		OutRepCnt: 0,
		ReqCS:     false,
		RepDefd:   make([]bool, numProcesses),
		ms:        &msgs,
		done:      make(chan bool),
		chrep:     make(chan bool),
		Mutex:     &sync.Mutex{},
		op: operacion,
		op_matrix: [][]bool{
			{true, false,}, 
			{false, false},
		},
		Clock: Aux
	}
	return ra
}

// PreProtocol: Realiza el PreProtocol del algoritmo Ricart-Agrawala Generalizado
func (ra *RASharedDB) PreProtocol() {
	ra.Mutex.Lock()

	// Incrementa el número de secuencia y marca la intención de entrar en la sección crítica
	ra.OurSeqNum = ra.HigSeqNum + 1
	ra.ReqCS = true
	ra.OutRepCnt = 0

	// Envía una solicitud a todos los procesos
	for i := 0; i < len(ra.ms.Peers()); i++ {
		if i != ra.ms.Me()-1 { // No se envía a sí mismo
			req := Request{Clock: ra.Clock, Pid: ra.ms.Me(), SeqNum: ra.OurSeqNum, op: ra.op}
			ra.ms.Send(i+1, req)

		}
	}
	ra.Mutex.Unlock()

	// Espera hasta recibir todas las respuestas
	for ra.OutRepCnt < len(ra.ms.Peers())-1 {
		<-ra.chrep
	}
}

// PostProtocol: Realiza el PostProtocol del algoritmo Ricart-Agrawala Generalizado
func (ra *RASharedDB) PostProtocol() {
	ra.Mutex.Lock()
	ra.ReqCS = false

	// Envía respuestas diferidas
	for i := 0; i < len(ra.ms.Peers()); i++ {
		if ra.RepDefd[i] {
			reply := Reply{}
			ra.ms.Send(i+1, reply)
			ra.RepDefd[i] = false
		}
	}
	ra.Mutex.Unlock()
}

// Comprueba que un vector sea estrictamente menor que otro
func vectormenor (v1 []int, v2 []int) bool {
	for i:=0; i < len(v1); i++ {
		if(v1[i] >= v2[i]) {
			return false
		}
	}
	return true;
}

// Comprueba que un vector sea estrictamente mayor que otro
func vectormayor (v1 []int, v2 []int) bool {
	for i:=0; i < len(v1); i++ {
		if(v1[i] <= v2[i]) {
			return false
		}
	}
	return true;
}

// Manejador de solicitudes entrantes
func (ra *RASharedDB) HandleRequest(req Request) {
	ra.Mutex.Lock()


	// Determina si debe enviar un Reply de inmediato o diferirlo
	if (ra.ReqCS && ((vectormenor(ra.Clock, req.Clock) || (!vectormenor(ra.Clock, req.Clock) && !vectormayor(ra.Clock, req.Clock) && ra.HigSeqNum < req.SeqNum)) || !ra.op_matrix[ra.op][req.op])) {
		// Diferir respuesta
		ra.RepDefd[req.Pid-1] = true

	} else {
		// Enviar respuesta inmediata
		reply := Reply{}
		ra.ms.Send(req.Pid, reply)
	}

	// Actualiza el número de secuencia más alto conocido
	if req.SeqNum > ra.HigSeqNum {
		ra.HigSeqNum = req.SeqNum
	}

	ra.Mutex.Unlock()
}

// Manejador de respuestas entrantes
func (ra *RASharedDB) HandleReply() {
	ra.Mutex.Lock()
	ra.OutRepCnt++
	ra.Mutex.Unlock()

	// Señaliza que ha recibido una respuesta
	ra.chrep <- true
}

// Función para detener el sistema
func (ra *RASharedDB) Stop() {
	ra.ms.Stop()
	ra.done <- true
}

// Función que escucha y maneja mensajes recibidos
func (ra *RASharedDB) Listen() {
	for {
		// Verifica si el canal `ra.done` ha recibido una señal para detenerse
		select {
		case <-ra.done:
			return
		default:
			// Recibe el mensaje desde la función Receive
			msg := ra.ms.Receive()

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
