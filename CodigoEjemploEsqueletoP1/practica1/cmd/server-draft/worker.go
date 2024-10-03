package main

import (
	"bufio"
	"encoding/gob"
	"log"
	"net"
	"os"
	"os/exec"
	"practica1/com"
	"strings"
)

var availableWorkers int

func processRequest(tasks <-chan com.Task, workerAddr string) {
	i := 0
	for task := range tasks {
		log.Println("Ha llegado una conexón al master para conectarse")

		workerConn, err := net.Dial("tcp", workerAddr)
		if err != nil {
			log.Println("Error connecting to worker:", err)
			task.Conn.Close()
			continue
		}

		encoder := gob.NewEncoder(workerConn)
		err = encoder.Encode(&task.Request)
		if err != nil {
			log.Println("Error encoding request to worker:", err)
			workerConn.Close()
			task.Conn.Close()
			continue
		}

		var reply com.Reply
		decoder := gob.NewDecoder(workerConn)
		err = decoder.Decode(&reply)
		if err != nil {
			log.Println("Error decoding reply from worker:", err)
			workerConn.Close()
			task.Conn.Close()
			continue
		}

		encoder = gob.NewEncoder(task.Conn)
		err = encoder.Encode(&reply)
		if err != nil {
			log.Println("Error encoding reply to client:", err)
		}
		log.Println("Ha mandado una respuesta")
		workerConn.Close()
		task.Conn.Close()
		i++
		log.Println(i)

	}
}

func startWorkers(workers []string, workersFile string) error {
	for _, workerAddr := range workers {
		// Separa la IP y el puerto del worker
		parts := strings.Split(workerAddr, ":")
		if len(parts) != 2 {
			log.Printf("Invalid worker address: %s", workerAddr)
			continue
		}
		ip := parts[0]
		port := parts[1]

		// Comando SSH para ejecutar el worker en la máquina remota
		sshCmd := exec.Command("ssh", ip,"cd","/misc/alumnos/sd/sd2425/a872815/practica1/cmd/server-draft","&&","go","run","workers.go", ip+":"+port)
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr

		// Inicia el comando SSH
		err := sshCmd.Start()
		if err != nil {
			log.Printf("Error starting worker at %s: %v", workerAddr, err)
			return err
		}

		log.Printf("Worker started at %s via SSH", workerAddr)
	}
	return nil
}

func loadWorkers(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var workers []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		workers = append(workers, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return workers, nil
}

func main() {
	args := os.Args
	if len(args) != 3 {
		log.Println("Error: usage: go run server.go ip:port workers.txt")
		os.Exit(1)
	}
	endpoint := args[1]
	workersFile := args[2]
	workers, err := loadWorkers(workersFile)
	if err != nil {
		log.Fatalf("Error loading workers: %v", err)
	}
	longitud := len(workers)
	if longitud == 0 {
		log.Fatalf("No workers found in %s", workersFile)
	}
	startWorkers(workers, workersFile)
	tasks := make(chan com.Task)
	for i := 0; i < longitud; i++ {
		go processRequest(tasks, workers[i])
	}

	listener, err := net.Listen("tcp", endpoint)
	com.CheckError(err)

	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	i := 0
	log.Println("***** Listening for new connection in endpoint ", endpoint)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}

		log.Println("Ha llegado una conexón al master")
		var request com.Request
		decoder := gob.NewDecoder(conn)
		err = decoder.Decode(&request)
		com.CheckError(err)
		tasks <- com.Task{Conn: conn, Request: request}
		i++
		log.Println(i)
	}
}
a872815@r20:/misc/alumnos/sd/sd2425/a872815/practica1/cmd/server-draft$ cat workers.go
package main

import (
    "encoding/gob"
    "log"
    "net"
    "os"
    "practica1/com"
)

func isPrime(n int) bool {
    if n <= 1 {
        return false
    }
    for i := 2; i*i <= n; i++ {
        if n%i == 0 {
            return false
        }
    }
    return true
}

func findPrimes(interval com.TPInterval) []int {
    var primes []int
    for i := interval.Min; i <= interval.Max; i++ {
        if isPrime(i) {
            primes = append(primes, i)
        }
    }
    return primes
}

func worker(conn net.Conn) {
    defer conn.Close()
    var request com.Request
    decoder := gob.NewDecoder(conn)
    err := decoder.Decode(&request)
    if err != nil {
        log.Println("Error decoding request:", err)
        return
    }

    log.Println("Ha llegado una conexón al worker")
    primes := findPrimes(request.Interval)
    reply := com.Reply{Id: request.Id, Primes: primes}
    encoder := gob.NewEncoder(conn)
    err = encoder.Encode(&reply)
    if err != nil {
        log.Println("Error encoding reply:", err)
    }
    log.Println("Ha mandado una respuesta")
}

func main() {
    args := os.Args
    if len(args) != 2 {
        log.Println("Error: usage: go run worker.go ip:port")
        os.Exit(1)
    }
    endpoint := args[1]

    listener, err := net.Listen("tcp", endpoint)
    if err != nil {
        log.Fatalf("Error starting TCP listener: %v", err)
    }
    defer listener.Close()

    log.SetFlags(log.Lshortfile | log.Lmicroseconds)
    log.Println("***** Listening for new connection in endpoint", endpoint)

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Println("Error accepting connection:", err)
            continue
        }
        worker(conn)
    }
}
