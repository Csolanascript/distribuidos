package main

import (
	//"errors"
	"fmt"
	//"log"
	"net"
	"net/rpc"
	"os"
	"raft/internal/comun/check"
	"raft/internal/comun/rpctimeout"
	"raft/internal/raft"
	"strconv"
	"strings"
	//"time"
)

func main() {
	// Obtener entero de índice de este nodo
	meStr := os.Args[1]
	name, me, err := parseNodeIndex(meStr)
	check.CheckError(err, "Main, mal índice de nodo:")

	dns := "raft-service.default.svc.cluster.local:6000"
	direcciones := generateAddresses(name, dns, 3)

	nodos := convertToHostPort(direcciones)

	// Parte Servidor
	nr := raft.NuevoNodo(nodos, me, make(chan raft.AplicaOperacion, 1000))
	rpc.Register(nr)

	fmt.Println("Replica escucha en :", me, " de ", direcciones)

	l, err := net.Listen("tcp", direcciones[me])
	check.CheckError(err, "Main listen error:")

	for {
		rpc.Accept(l)
	}
}

func parseNodeIndex(meStr string) (string, int, error) {
	parts := strings.Split(meStr, "-")
	name := parts[0]
	me, err := strconv.Atoi(parts[1])
	return name, me, err
}

func generateAddresses(name, dns string, count int) []string {
	var direcciones []string
	for i := 0; i < count; i++ {
		nodo := fmt.Sprintf("%s-%d.%s", name, i, dns)
		direcciones = append(direcciones, nodo)
	}
	return direcciones
}

func convertToHostPort(direcciones []string) []rpctimeout.HostPort {
	var nodos []rpctimeout.HostPort
	for _, endPoint := range direcciones {
		nodos = append(nodos, rpctimeout.HostPort(endPoint))
	}
	return nodos
}
