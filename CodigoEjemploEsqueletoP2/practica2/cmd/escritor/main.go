package main

import (
	"fmt"
	"os"
	"time"
	"practica2/ra"
)

// Escritor representa un proceso escritor que escribe en su archivo
func Escritor(writerID int) {
	// Crear una instancia del algoritmo Ricart-Agrawala
	raInstance := ra.New(writerID, "lectores.txt", 1) // Se usa 1 para identificar escritores
	go raInstance.Listen(raInstance)

	for {
		// Fase de preprotocolo: solicitar acceso a la sección crítica
		fmt.Printf("Escritor %d solicitando acceso a la sección crítica para escribir...\n", writerID)
		raInstance.PreProtocol()

		// Sección crítica: escribir en el archivo compartido
		fmt.Printf("Escritor %d accede a la sección crítica para escribir...\n", writerID)
		EscribirFichero(fmt.Sprintf("fichero_escritor_%d.txt", writerID), fmt.Sprintf("Escritura del escritor %d", writerID))

		// Fase de postprotocolo: liberar la sección crítica
		raInstance.PostProtocol()
		fmt.Printf("Escritor %d ha liberado la sección crítica\n", writerID)

		// Pausa para simular la concurrencia
		time.Sleep(time.Second * 5)
	}
}

// Función para escribir en el archivo
func EscribirFichero(filename string, content string) {
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		fmt.Printf("Error al escribir en el fichero %s: %v\n", filename, err)
	}
}

// ContarLineas cuenta el número de líneas en el archivo especificado sin manejar errores explícitamente
func ContarLineas(filename string) int {
	file, err := os.Open(filename)
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	return lineCount
}

func main() {
	int NEscritores := ContarLineas("escritores.txt")
	for i := 1; i <= Nescritores; i++ {
		go Escritor(i)
	}
	while(1)
}
