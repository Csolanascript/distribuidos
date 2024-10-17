package main

import (
	"bufio"
	"fmt"
	"os"
	"practica2/ra"
	"strconv"
	"time"
)

func contarLineas(filePath string) int {
	file, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return 0
	}

	return lineCount
}

// Función para escribir en ficheros compartidos en modo append
func escribirEnFicherosCompartidos(numFicheros int, ID int) error {
	for i := 1; i <= numFicheros; i++ {
		fileName := fmt.Sprintf("../fichero_compartido_%d.txt", i)

		// Abrir el archivo en modo append. Si no existe, se crea.
		file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("error al abrir %s: %v", fileName, err)
		}

		// Escribir en el archivo
		_, err = file.WriteString(fmt.Sprintf("Ha escrito el proceso %d\n", ID))
		if err != nil {
			file.Close()
			return fmt.Errorf("error al escribir en %s: %v", fileName, err)
		}

		// Cerrar el archivo después de escribir
		if err := file.Close(); err != nil {
			return fmt.Errorf("error al cerrar %s: %v", fileName, err)
		}
	}

	return nil
}

// Escritor representa un proceso escritor que escribe en su archivo
func Escritor(writerID int) {
	fmt.Printf("Empieza el proceso de escribir %d \n", writerID)
	// Crear una instancia del algoritmo Ricart-Agrawala
	raInstance := ra.New(writerID, "../users.txt", 1) // Se usa 1 para identificar escritores
	go raInstance.Listen()
	lineCount := contarLineas("../users.txt")
	fmt.Printf("El proceso %d ha contado %d lineas\n", writerID, lineCount)
	for {
		// Fase de preprotocolo: solicitar acceso a la sección crítica
		fmt.Printf("Escritor %d solicitando acceso a la sección crítica para escribir...\n", writerID)
		raInstance.PreProtocol()

		// Sección crítica: escribir en el archivo compartido
		fmt.Printf("Escritor %d accede a la sección crítica para escribir...\n", writerID)
		escribirEnFicherosCompartidos(lineCount, writerID)
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
