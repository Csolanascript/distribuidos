package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"practica2/ra"
	"strconv"
	"strings"
	"time"
)

// Escritor representa un proceso escritor que escribe en su archivo
func Escritor(writerID int) {
	// Crear una instancia del algoritmo Ricart-Agrawala
	raInstance := ra.New(writerID, "../users.txt", 1) // Se usa 1 para identificar escritores
	go raInstance.Listen()

	for i := 0; i < 3; i++ {
		// Fase de preprotocolo: solicitar acceso a la sección crítica
		fmt.Printf("Escritor %d solicitando acceso a la sección crítica para escribir...\n", writerID)
		raInstance.PreProtocol()

		// Sección crítica: escribir en el archivo compartido
		fmt.Printf("Escritor %d accede a la sección crítica para escribir...\n", writerID)
		AppendToFile("../archivo_compartido.txt", fmt.Sprintf("Escritura del escritor %d", writerID))
		time.Sleep(time.Second * 3)
		transferFileToMultipleIPs("../users.txt", "../archivo_compartido.txt", "/home/usuario/")
		// Fase de postprotocolo: liberar la sección crítica
		raInstance.PostProtocol()
		fmt.Printf("Escritor %d ha liberado la sección crítica\n", writerID)

		// Pausa para simular la concurrencia
		time.Sleep(time.Second * 3)
	}
}

func transferFileToMultipleIPs(usersFile, fileToTransfer, remotePath string) error {
	// Abrir el archivo users.txt
	file, err := os.Open(usersFile)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de usuarios: %v", err)
	}
	defer file.Close()

	// Leer el archivo línea por línea
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Leer la línea actual (ip:puerto)
		line := scanner.Text()

		// Separar IP y puerto, y quedarnos solo con la IP
		ipPort := strings.Split(line, ":")
		ip := ipPort[0]

		// Comando scp usando la IP, ignoramos el puerto
		cmd := exec.Command("scp", fileToTransfer, fmt.Sprintf("usuario@%s:%s", ip, remotePath))

		// Ejecutar el comando
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Error al transferir archivo a %s: %v\n", ip, err)
		} else {
			fmt.Printf("Archivo transferido exitosamente a %s\n", ip)
		}
	}

	// Manejo de errores de lectura del archivo
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error al leer el archivo de usuarios: %v", err)
	}

	return nil
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
