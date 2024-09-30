package main

import (
    "encoding/gob"
    "log"
    "net"
    "os"
    "practica1/com"
    "strconv"
)

func isPrime(n int) (foundDivisor bool) {
    foundDivisor = false
    for i := 2; (i < n) && !foundDivisor; i++ {
        foundDivisor = (n%i == 0)
    }
    return !foundDivisor
}

func findPrimes(interval com.TPInterval) (primes []int) {
    for i := interval.Min; i <= interval.Max; i++ {
        if isPrime(i) {
            primes = append(primes, i)
        }
    }
    return primes
}

func worker(tasks <-chan com.Task) {
    for task := range tasks {
		log.Println("Ha llegado una conexón al worker")
        primes := findPrimes(task.Request.Interval)
        reply := com.Reply{Id: task.Request.Id, Primes: primes}
        encoder := gob.NewEncoder(task.Conn)
        encoder.Encode(&reply)
        task.Conn.Close()
    }
}

func processRequest(conn net.Conn, tasks chan<- com.Task) {
    log.Println("Ha llegado una conexón al master")
    var request com.Request
    decoder := gob.NewDecoder(conn)
    err := decoder.Decode(&request)
    com.CheckError(err)
    tasks <- com.Task{Conn: conn, Request: request}
}

func main() {
    args := os.Args
    if len(args) != 3 {
        log.Println("Error: usage: go run server.go ip:port poolSize")
        os.Exit(1)
    }
    endpoint := args[1]
    poolSize, err := strconv.Atoi(args[2])
    com.CheckError(err)

    listener, err := net.Listen("tcp", endpoint)
    com.CheckError(err)

    log.SetFlags(log.Lshortfile | log.Lmicroseconds)

    tasks := make(chan com.Task)

    for i := 0; i < poolSize; i++ {
        go worker(tasks)
    }

    log.Println("***** Listening for new connection in endpoint ", endpoint)
    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Println("Error accepting connection:", err)
            continue
        }
        processRequest(conn, tasks)
    }
}