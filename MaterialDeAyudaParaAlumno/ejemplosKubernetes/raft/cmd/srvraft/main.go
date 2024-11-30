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
	// obtener entero de indice de este nodo
	meStr := os.Args[1]
	name := strings.Split(meStr, "-")[0]
	me, err := strconv.Atoi(strings.Split(meStr, "-")[1])
	check.CheckError(err, "Main, mal índice de nodo:")

	dns := "raft-service.default.svc.cluster.local:6000"
	var direcciones []string
	for i := 0; i < 3; i++ {
		nodo := name + "-" + strconv.Itoa(i) + "." + dns
		direcciones = append(direcciones, nodo)
	}

	var nodos []rpctimeout.HostPort
	// Resto de argumento son los end points como strings
	// De todas la replicas-> pasarlos a HostPort
	for _, endPoint := range direcciones {
		nodos = append(nodos, rpctimeout.HostPort(endPoint))
	}

	// Parte Servidor
	nr := raft.NuevoNodo(nodos, me, make(chan raft.AplicaOperacion, 1000))
	rpc.Register(nr)

	fmt.Println("Replica escucha en :", me, " de ", direcciones)

	l, err := net.Listen("tcp", direcciones[me])
	check.CheckError(err, "Main listen error:")

	for {
		rpc.Accept(l)
	}

	//METER LO DE FUNCION APLICAR OPERACION?????
}
