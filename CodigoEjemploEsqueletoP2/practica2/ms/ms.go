/*
* AUTOR: Rafael Tolosana Calasanz
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: septiembre de 2021
* FICHERO: ms.go
* DESCRIPCIÓN: Implementación de un sistema de mensajería asíncrono, insipirado en el Modelo Actor
 */
package ms

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"net"
	"os"
)

type Message interface{}

type MessageSystem struct {
	mbox  chan Message
	peers []string
	done  chan bool
	me    int
}

const (
	MAXMESSAGES = 10000
)

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

// Método público para acceder a la lista de peers
func (ms *MessageSystem) Peers() []string {
	return ms.peers
}

// Método para acceder al valor de `me` desde fuera del paquete
func (ms *MessageSystem) Me() int {
	return ms.me
}

func parsePeers(path string) (lines []string) {
	file, err := os.Open(path)
	checkError(err)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

// Pre: pid en {1..n}, el conjunto de procesos del SD
// Post: envía el mensaje msg a pid
func (ms *MessageSystem) Send(pid int, msg Message) {
	// Establecer la conexión TCP
	conn, err := net.Dial("tcp", ms.peers[pid-1])
	if err != nil {
		// Manejo del error de conexión
		fmt.Fprintf(os.Stderr, "Error al conectar con el proceso %d: %s\n", pid, err.Error())
		return
	}
	defer conn.Close()

	// Codificar y enviar el mensaje
	encoder := gob.NewEncoder(conn)
	err = encoder.Encode(&msg)
	if err != nil {
		// Manejo del error al codificar el mensaje
		fmt.Fprintf(os.Stderr, "Error al codificar mensaje para el proceso %d: %s\n", pid, err.Error())
		return
	}

	// Cerrar la conexión (el defer conn.Close() ya se encargará de cerrar la conexión)
	err = conn.Close() // Esto es opcional, ya que `defer conn.Close()` lo cerrará de todas maneras
	if err != nil {
		// Manejo del error al cerrar la conexión
		fmt.Fprintf(os.Stderr, "Error al cerrar la conexión con el proceso %d: %s\n", pid, err.Error())
		return
	}
}

// Pre: True
// Post: el mensaje msg de algún Proceso P_j se retira del mailbox y se devuelve
//
//	Si mailbox vacío, Receive bloquea hasta que llegue algún mensaje
func (ms *MessageSystem) Receive() (msg Message) {
	msg = <-ms.mbox
	return msg
}

//	messageTypes es un slice con tipos de mensajes que los procesos se pueden intercambiar a través de este ms
//
// Hay que registrar un mensaje antes de poder utilizar (enviar o recibir)
// Notar que se utiliza en la función New
func Register(messageTypes []Message) {
	for _, msgTp := range messageTypes {
		gob.Register(msgTp)
	}
}

// Pre: whoIam es el pid del proceso que inicializa este ms
// usersFile es la ruta a un fichero de texto que en cada línea contiene IP:puerto de cada participante
// messageTypes es un slice con todos los tipos de mensajes que los procesos se pueden intercambiar a través de este ms
func New(whoIam int, usersFile string, messageTypes []Message) (ms MessageSystem) {
	ms.me = whoIam
	ms.peers = parsePeers(usersFile)
	ms.mbox = make(chan Message, MAXMESSAGES)
	ms.done = make(chan bool)
	Register(messageTypes)

	go func() {
		listener, err := net.Listen("tcp", ms.peers[ms.me-1])
		if err != nil {
			// Manejo del error al abrir el listener
			fmt.Fprintf(os.Stderr, "Error al iniciar listener en %s: %s\n", ms.peers[ms.me-1], err.Error())
			return
		}
		fmt.Println("Process listening at " + ms.peers[ms.me-1])
		defer close(ms.mbox)

		for {
			select {
			case <-ms.done:
				return
			default:
				conn, err := listener.Accept()
				if err != nil {
					// Manejo del error al aceptar la conexión
					fmt.Fprintf(os.Stderr, "Error al aceptar conexión: %s\n", err.Error())
					continue
				}

				decoder := gob.NewDecoder(conn)
				var msg Message
				err = decoder.Decode(&msg)
				if err != nil {
					// Manejo del error al decodificar el mensaje
					fmt.Fprintf(os.Stderr, "Error al decodificar el mensaje: %s\n", err.Error())
					conn.Close() // Asegúrate de cerrar la conexión incluso si hay un error
					continue
				}

				conn.Close()   // Cierra la conexión normalmente
				ms.mbox <- msg // Enviar el mensaje al canal mbox
			}
		}
	}()
	return ms
}

// Pre: True
// Post: termina la ejecución de este ms
func (ms *MessageSystem) Stop() {
	ms.done <- true
}
