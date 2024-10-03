package main

import (
	"encoding/gob"
	"log"
	"net"
	"os"
	"practica1/com"
)

// PRE: verdad = !foundDivisor
// POST: IsPrime devuelve verdad si n es primo y falso en caso contrario
func isPrime(n int) (foundDivisor bool) {
	foundDivisor = false
	for i := 2; (i < n) && !foundDivisor; i++ {
		foundDivisor = (n%i == 0)
	}
	return !foundDivisor
}

// PRE: interval.A < interval.B
// POST: FindPrimes devuelve todos los números primos comprendidos en el
// intervalo [interval.A, interval.B]
func findPrimes(interval com.TPInterval) (primes []int) {
	for i := interval.Min; i <= interval.Max; i++ {
		if isPrime(i) {
			primes = append(primes, i)
		}
	}
	return primes
}

// Procesa la solicitud recibida en la conexión
func processRequest(conn net.Conn) {
	log.Println("A connection has arrived to the goroutines pool")
	defer conn.Close() // Asegura que la conexión se cierre al finalizar la función
	var request com.Request
	decoder := gob.NewDecoder(conn)
	err := decoder.Decode(&request)
	com.CheckError(err)
	primes := findPrimes(request.Interval)
	reply := com.Reply{Id: request.Id, Primes: primes}
	encoder := gob.NewEncoder(conn)
	encoder.Encode(&reply)
}

func main() {
	args := os.Args
	if len(args) != 2) {
		log.Println("Error: endpoint missing: go run server.go ip:port")
		os.Exit(1)
	}
	endpoint := args[1]
	listener, err := net.Listen("tcp", endpoint)
	com.CheckError(err)

	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	log.Println("***** Listening for new connection in endpoint ", endpoint)
	for {
		conn, err := listener.Accept()
		defer conn.Close() // Asegura que la conexión se cierre al finalizar la función
		com.CheckError(err)
		go processRequest(conn) // Procesa la solicitud en una nueva goroutine
	}
}
