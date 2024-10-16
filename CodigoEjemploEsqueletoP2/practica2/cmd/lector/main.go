package main

import (
	"fmt"
	"os"
	"practica2/ra"
	"time"
)

func Lector(processID int) {
	raInstance := ra.New(processID, "lectores.txt", 0)
	go raInstance.Listen(raInstance)
	for {
		// Fase de preprotocolo: solicitar acceso a la sección crítica (lectura)
		fmt.Printf("Lector %d solicitando acceso a la sección crítica para leer...\n", processID)
		raInstance.PreProtocol()

		// Sección crítica: lectura del archivo compartido
		fmt.Printf("Lector %d accede a la sección crítica para leer...\n", processID)
		content := LeerFichero("archivo_compartido.txt")
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
	// Crear instancia del algoritmo Ricart-Agrawala
	int NLectores := ContarLineas("lectores.txt")
	int NEscritores := ContarLineas("../escritor/escritores.txt")
	for i := Nescritores; i <= Nlectores + Nescritores; i++ {  // Crear NLectores lectores con IDs diferentes
		go Lector(i)
	}

	while(1)
}

