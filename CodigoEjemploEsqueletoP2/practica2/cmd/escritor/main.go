package main

import (
	"fmt"
	"os"
	"practica2/ra"
	"strconv"
	"time"
)

// Escritor representa un proceso escritor que escribe en su archivo
func Escritor(writerID int) {
	// Crear una instancia del algoritmo Ricart-Agrawala
	raInstance := ra.New(writerID, "../users.txt", 1) // Se usa 1 para identificar escritores
	go raInstance.Listen()

	for {
		// Fase de preprotocolo: solicitar acceso a la sección crítica
		fmt.Printf("Escritor %d solicitando acceso a la sección crítica para escribir...\n", writerID)
		raInstance.PreProtocol()

		// Sección crítica: escribir en el archivo compartido
		fmt.Printf("Escritor %d accede a la sección crítica para escribir...\n", writerID)
		AppendToFile("../archivo_compartido.txt", fmt.Sprintf("Escritura del escritor %d\n", writerID))
		time.Sleep(time.Second * 3)
		// Fase de postprotocolo: liberar la sección crítica
		raInstance.PostProtocol()
		fmt.Printf("Escritor %d ha liberado la sección crítica\n", writerID)

		// Pausa para simular la concurrencia
		time.Sleep(time.Second * 3)
	}
}

func AppendToFile(filename string, content string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error al abrir el fichero %s: %v\n", filename, err)
		return
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		fmt.Printf("Error al escribir en el fichero %s: %v\n", filename, err)
	}
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

	Escritor(writerID)
}
