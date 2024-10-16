package main

import (
	"fmt"
	"os"
	"practica2/ra"
	"strconv"
	"time"
)

func Lector(processID int) {
	raInstance := ra.New(processID, "../users.txt", 0)
	go raInstance.Listen()
	for {
		// Fase de preprotocolo: solicitar acceso a la sección crítica (lectura)
		fmt.Printf("Lector %d solicitando acceso a la sección crítica para leer...\n", processID)
		raInstance.PreProtocol()

		// Sección crítica: lectura del archivo compartido
		fmt.Printf("Lector %d accede a la sección crítica para leer...\n", processID)
		content := LeerFichero("../archivo_compartido.txt")
		fmt.Printf("Lector %d leyó: %s\n", processID, content)

		// Fase de postprotocolo: liberar la sección crítica
		raInstance.PostProtocol()
		fmt.Printf("Lector %d ha liberado la sección crítica\n", processID)

		// Pausa para simular la actividad de los lectores
		time.Sleep(time.Second * 3)
	}
}

// Función para leer el archivo
func LeerFichero(filename string) string {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error al leer el fichero %s: %v\n", filename, err)
		return ""
	}
	return string(data)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: escritor <writerID>")
		return
	}

	writerID, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Printf("Error: writerID debe ser un número entero. %v\n", err)
		return
	}

	Lector(writerID)
}
