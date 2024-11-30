package main

import (
	"fmt"
	"raft/internal/comun/check"
	"raft/internal/comun/rpctimeout"
	"raft/internal/raft"
	"time"
)

const (
	// Nodos replicas
	REPLICA1 = "raft-0.raft-service.default.svc.cluster.local:6000"
	REPLICA2 = "raft-1.raft-service.default.svc.cluster.local:6000"
	REPLICA3 = "raft-2.raft-service.default.svc.cluster.local:6000"
)

var REPLICAS = rpctimeout.StringArrayToHostPortArray([]string{REPLICA1, REPLICA2, REPLICA3})

func printEstadoRemoto(reply raft.EstadoRemoto) {
	fmt.Println()
	fmt.Printf("IdNodo:  %d\n", reply.IdNodo)
	fmt.Printf("Mandato: %d\n", reply.Mandato)
	fmt.Printf("EsLider: %v\n", reply.EsLider)
	fmt.Printf("IdLider: %d\n", reply.IdLider)
	fmt.Println()
}

func printSometerOperacion(reply raft.ResultadoRemoto) {
	fmt.Println()
	fmt.Printf("IndiceRegistro: %d\n", reply.IndiceRegistro)
	fmt.Printf("Mandato: %d\n", reply.Mandato)
	fmt.Printf("EsLider: %v\n", reply.EsLider)
	fmt.Printf("IdLider: %v\n", reply.IdLider)
	fmt.Printf("ValorADevolver: %s\n", reply.ValorADevolver)
	fmt.Println()
	fmt.Println("========================================")
}

func main() {
	// Sequence of operations
	operations := []struct {
		opcion int
		nodo   int
		clave  string
		valor  string
	}{
		{1, 0, "", ""},             // Obtener estado del nodo 0
		{1, 1, "", ""},             // Obtener estado del nodo 1
		{1, 2, "", ""},             // Obtener estado del nodo 2
		{4, 0, "clave1", ""},       // Someter operación de lectura en el nodo 0
		{5, 0, "clave1", "valor1"}, // Someter operación de escritura en el nodo 0
		{3, 1, "", ""},             // Detener nodo 1
		{1, 0, "", ""},             // Obtener estado del nodo 0
	}

	var lider int
	for _, op := range operations {
		time.Sleep(1000 * time.Millisecond)
		fmt.Printf("Opción seleccionada: %d\n", op.opcion)
		if op.opcion > 0 && op.opcion < 6 {
			fmt.Printf("Nodo seleccionado: %d\n", op.nodo)
		}

		switch op.opcion {
		case 1:
			var reply raft.EstadoRemoto
			err := REPLICAS[op.nodo].CallTimeout("NodoRaft.ObtenerEstadoNodo",
				raft.Vacio{}, &reply, 500*time.Millisecond)
			check.CheckError(err, "Error en llamada RPC ObtenerEstadoNodo")
			printEstadoRemoto(reply)
			if reply.EsLider {
				lider = op.nodo
			}
		case 3:
			time.Sleep(5000 * time.Millisecond)
			var reply raft.Vacio
			err := REPLICAS[lider].CallTimeout("NodoRaft.ParaNodo",
				raft.Vacio{}, &reply, 500*time.Millisecond)
			check.CheckError(err, "Error en llamada RPC Para nodo")
		case 4, 5:
			if op.clave == "" {
				fmt.Println("Debes especificar una clave para la operación.")
				continue
			}
			var operacion raft.TipoOperacion
			operacion.Clave = op.clave
			if op.opcion == 4 {
				operacion.Operacion = "lectura"
			} else {
				if op.valor == "" {
					fmt.Println("Debes especificar un valor para la escritura.")
					continue
				}
				operacion.Operacion = "escritura"
				operacion.Valor = op.valor
			}
			var reply raft.ResultadoRemoto
			err := REPLICAS[lider].CallTimeout("NodoRaft.SometerOperacionRaft",
				operacion, &reply, 5000*time.Millisecond)
			check.CheckError(err, "Error en llamada RPC SometerOperacionRaft")
			printSometerOperacion(reply)
		default:
			fmt.Println("Opción inválida. Introduce una opción válida.")
		}
	}
}
