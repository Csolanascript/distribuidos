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