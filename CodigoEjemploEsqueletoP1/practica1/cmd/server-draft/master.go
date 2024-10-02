package main

import (
    "bufio"
    "encoding/gob"
    "log"
    "net"
    "os"
    "practica1/com"
)

var availableWorkers int

func processRequest(tasks <-chan com.Task, workerAddr string) {
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
    }
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
    if len(workers) == 0 {
        log.Fatalf("No workers found in %s", workersFile)
    }


    tasks := make(chan com.Task)
    for i:=0; i < len(workers); i++ {
        go processRequest(tasks, workers[i])
    }

    listener, err := net.Listen("tcp", endpoint)
    com.CheckError(err)

    log.SetFlags(log.Lshortfile | log.Lmicroseconds)

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
    }
}