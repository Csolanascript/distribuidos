package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

func readEndpoints(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var endpoints []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			endpoints = append(endpoints, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return endpoints, nil
}

func handleConnection(conn net.Conn, barrierChan chan<- bool, received *map[string]bool, mu *sync.Mutex, n int) {
	defer conn.Close()
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading from connection:", err)
		return
	}
	msg := string(buf)
	mu.Lock()
	(*received)[msg] = true
	fmt.Println("Received ", len(*received), " elements")
	if len(*received) == n-1 {
		barrierChan <- true
	}
	mu.Unlock()
}

// Get enpoints (IP adresse:port for each distributed process)
func getEndpoints() ([]string, int, error) {
	endpointsFile := os.Args[1]
	var endpoints []string // Por qué esta declaración ?
	lineNumber, err := strconv.Atoi(os.Args[2])
	if err != nil || lineNumber < 1 {
		fmt.Println("Invalid line number")
	} else if endpoints, err = readEndpoints(endpointsFile); err != nil {
		fmt.Println("Error reading endpoints:", err)
	} else if lineNumber > len(endpoints) {
		fmt.Printf("Line number %d out of range\n", lineNumber)
		err = errors.New("Line number out of range")
	}

	return endpoints, lineNumber, err
}

func acceptAndHandleConnections(listener net.Listener, quitChannel chan bool,
	barrierChan chan bool, receivedMap *map[string]bool, mu *sync.Mutex, n int) {
	for {
		select {
		case <-quitChannel:
			fmt.Println("Stopping the listener...")
			break
		default:
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Error accepting connection:", err)
				continue
			}
			go handleConnection(conn, barrierChan, receivedMap, mu, n)
		}
	}
}

func notifyOtherDistributedProcesses(endPoints []string, lineNumber int, mutex *sync.Mutex, finalChannel chan bool, n int) {
	contador := 0
	for i, ep := range endPoints {
		if i+1 != lineNumber {
			go func(ep string) {
				for {
					conn, err := net.Dial("tcp", ep)
					if err != nil {
						fmt.Println("Error connecting to", ep, ":", err)
						time.Sleep(1 * time.Second)
						continue
					}
					_, err = conn.Write([]byte(strconv.Itoa(lineNumber)))
					if err != nil {
						fmt.Println("Error sending message:", err)
						conn.Close()
						continue
					}
					mutex.Lock()
					contador++
					if contador == n-1 {
						finalChannel <- true
					}
					mutex.Unlock()

					conn.Close()
					break
				}
			}(ep)
		}
	}
}

func main() {
	var listener net.Listener

	if len(os.Args) != 3 {
		fmt.Println("Usage: go run main.go <endpoints_file> <line_number>")
		return
	}

	endPoints, lineNumber, err := getEndpoints()
	if err != nil {
		fmt.Println("Error getting endpoints:", err)
		return
	}

	// Get the endpoint for current process
	localEndpoint := endPoints[lineNumber-1]
	//listener, err := net.Listen("tcp", localEndpoint)
	//if err != nil {
	//	fmt.Println("Error creating listener:", err)
	//	return
	//}
	//fmt.Println("Listening on", localEndpoint)

	if listener, err = net.Listen("tcp", localEndpoint); err != nil {
		fmt.Println("Error creating listener:", err)
	} else {
		fmt.Println("Listening on", localEndpoint)
	}

	// Barrier synchronization
	var mu sync.Mutex
	var mu2 sync.Mutex
	quitChannel := make(chan bool)
	receivedMap := make(map[string]bool)
	barrierChan := make(chan bool)
	finalChannel := make(chan bool)

	go acceptAndHandleConnections(listener, quitChannel, barrierChan, &receivedMap, &mu, len(endPoints))

	notifyOtherDistributedProcesses(endPoints, lineNumber, &mu2, finalChannel, len(endPoints))

	fmt.Println("Waiting for all the processes to reach the barrier")

	<-barrierChan
	<-finalChannel

	listener.Close()
	fmt.Println("Listener closed")
	quitChannel <- true
}
