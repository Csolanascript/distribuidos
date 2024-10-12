package ra

import (
	"practica2/ms"
	"sync"
)

type Request struct {
	Clock int
	Pid   int
}

type Reply struct{}

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
}

// Constructor para inicializar la estructura RASharedDB
func New(me int, usersFile string) *RASharedDB {
	messageTypes := []ms.Message{Request{}, Reply{}}
	msgs := ms.New(me, usersFile, messageTypes)
	numProcesses := len(msgs.Peers()) // Asumimos que la cantidad de procesos es igual al número de peers

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
			req := Request{Clock: ra.OurSeqNum, Pid: ra.ms.Me()}
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

// Manejador de solicitudes entrantes
func (ra *RASharedDB) HandleRequest(req Request) {
	ra.Mutex.Lock()

	// Actualiza el número de secuencia más alto conocido
	if req.Clock > ra.HigSeqNum {
		ra.HigSeqNum = req.Clock
	}

	// Determina si debe enviar un Reply de inmediato o diferirlo
	if ra.ReqCS && (ra.OurSeqNum < req.Clock || (ra.OurSeqNum == req.Clock && ra.ms.Me() < req.Pid)) {
		// Diferir respuesta
		ra.RepDefd[req.Pid-1] = true
	} else {
		// Enviar respuesta inmediata
		reply := Reply{}
		ra.ms.Send(req.Pid, reply)
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
		select {
		case msg := ra.ms.Receive(): // Recibe el mensaje desde la función
			switch m := msg.(type) {
			case Request:
				ra.HandleRequest(m)
			case Reply:
				ra.HandleReply()
			}
		case <-ra.done:
			return
		}
	}
}
