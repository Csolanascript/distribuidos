package main

import (
	"fmt"
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
	type Operation struct {
		opcion int
		nodo   int
		clave  string
		valor  string
	}

	operations := []Operation{
		{1, 0, "", ""},             // Obtener estado del nodo 0
		{1, 1, "", ""},             // Obtener estado del nodo 1
		{1, 2, "", ""},             // Obtener estado del nodo 2
		{4, 1, "clave1", ""},       // Someter operación de lectura en el nodo 0
		{5, 1, "clave1", "valor1"}, // Someter operación de escritura en el nodo lider
		{3, 1, "", ""},             // Detener nodo lider
	}

	var lider int
	i := 0

	for {
		time.Sleep(2000 * time.Millisecond)
		i = i % len(operations)
		op := operations[i]
		i = (i + 1)
		fmt.Printf("%+v", op)
		fmt.Printf("\n========================================\n")
		fmt.Printf("Opción seleccionada: %d\n", op.opcion)
		if op.opcion > 0 && op.opcion < 6 {
			if op.nodo >= 0 && op.nodo < len(REPLICAS) {
				fmt.Printf("Nodo seleccionado: %d\n", op.nodo)
			} else {
				fmt.Printf("Nodo seleccionado %d está fuera de rango\n", op.nodo)
				continue
			}
		}

		switch op.opcion {
		case 1:
			fmt.Printf("Obteniendo estado del nodo %d...\n", op.nodo)
			var reply raft.EstadoRemoto
			err := REPLICAS[op.nodo].CallTimeout("NodoRaft.ObtenerEstadoNodo",
				raft.Vacio{}, &reply, 500*time.Millisecond)
			if err != nil {
				fmt.Printf("Error en llamada RPC ObtenerEstadoNodo: %v\n", err)
				continue
			}
			printEstadoRemoto(reply)
			if reply.EsLider {
				lider = op.nodo
				fmt.Printf("Nodo %d es el líder.\n", lider)
			}
		case 3:
			fmt.Printf("Deteniendo nodo líder %d...\n", lider)
			time.Sleep(5000 * time.Millisecond)
			var reply raft.Vacio
			err := REPLICAS[lider].CallTimeout("NodoRaft.ParaNodo",
				raft.Vacio{}, &reply, 500*time.Millisecond)
			if err != nil {
				fmt.Printf("Error en llamada RPC Para nodo: %v\n", err)
				continue
			}
			fmt.Printf("Nodo líder %d detenido exitosamente.\n", lider)
			fmt.Printf("Obteniendo estado del nodo %d...\n", op.nodo)
			var reply2 raft.EstadoRemoto
			err = REPLICAS[op.nodo].CallTimeout("NodoRaft.ObtenerEstadoNodo",
				raft.Vacio{}, &reply2, 500*time.Millisecond)
			if err != nil {
				fmt.Printf("Error en llamada RPC ObtenerEstadoNodo: %v\n", err)
				continue
			}
			printEstadoRemoto(reply2)

		case 4, 5:
			if op.clave == "" {
				fmt.Println("Debes especificar una clave para la operación.")
				continue
			}
			var operacion raft.TipoOperacion
			operacion.Clave = op.clave
			if op.opcion == 4 {
				fmt.Printf("Sometiendo operación de lectura en el nodo líder %d con clave '%s'...\n", lider, op.clave)
				operacion.Operacion = "lectura"
			} else {
				if op.valor == "" {
					fmt.Println("Debes especificar un valor para la escritura.")
					continue
				}
				fmt.Printf("Sometiendo operación de escritura en el nodo líder %d con clave '%s' y valor '%s'...\n", lider, op.clave, op.valor)
				operacion.Operacion = "escritura"
				operacion.Valor = op.valor
			}
			var reply raft.ResultadoRemoto
			err := REPLICAS[lider].CallTimeout("NodoRaft.SometerOperacionRaft",
				operacion, &reply, 5000*time.Millisecond)
			if err != nil {
				fmt.Printf("Error en llamada RPC SometerOperacionRaft: %v\n", err)
				continue
			}
			printSometerOperacion(reply)
		default:
			fmt.Println("Opción inválida. Introduce una opción válida.")
		}
		fmt.Printf("========================================\n")
	}
}
