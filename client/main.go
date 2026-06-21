package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

// currentToken holds the most recently received YOUR TURN token.
// It is cleared after use so it is only sent once.
var (
	currentToken string
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Please provide host:port")
		os.Exit(1)
	}

	network := "udp"
	udpAddr, err := net.ResolveUDPAddr(network, os.Args[1])
	exitOnError(err)

	conn, err := net.DialUDP(network, nil, udpAddr)
	exitOnError(err)
	defer conn.Close()

	fmt.Println("Connected to server at", os.Args[1])
	fmt.Println("Type messages to send to the server (or 'quit' to exit)")

	// Send initial REGISTER message
	player := "Bob"
	_, err = conn.Write([]byte("REGISTER;" + player))
	exitOnError(err)
	fmt.Println("Sent: REGISTER;" + player)

	// Start goroutine to listen for incoming messages
	go listenForMessages(conn)

	// Read user input and send to server
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		message := strings.TrimSpace(scanner.Text())
		if message == "" {
			continue
		}

		if message == "quit" {
			fmt.Println("Exiting...")
			break
		}

		// Append the pending action token if not already present.
		// The token is kept until a new YOUR TURN message replaces it,
		// so an invalid/rejected command does not lose the token.
		if currentToken != "" && !strings.Contains(message, currentToken) {
			message = message + ";" + currentToken
		}

		_, err := conn.Write([]byte(message))
		if err != nil {
			log.Printf("Error sending message: %v\n", err)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading input: %v\n", err)
	}
}

func listenForMessages(conn *net.UDPConn) {
	buffer := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error receiving message: %v\n", err)
			continue
		}

		message := string(buffer[:n])
		fmt.Printf("\n[Server] %s\n> ", message)

		if strings.HasPrefix(message, "ROUND STARTING;") {
			handleRoundStarting(message, conn)
		}

		if strings.HasPrefix(message, "YOUR TURN;") {
			handleYourTurn(message)
		}
	}
}

func handleRoundStarting(message string, conn *net.UDPConn) {
	parts := strings.Split(message, ";")
	if len(parts) < 2 {
		log.Println("Invalid ROUND STARTING message format")
		return
	}

	token := strings.TrimSpace(parts[1])
	joinMessage := "JOIN;" + token
	_, err := conn.Write([]byte(joinMessage))
	if err != nil {
		log.Printf("Error sending JOIN message: %v\n", err)
	} else {
		fmt.Printf("[Client] Automatically sent: %s\n> ", joinMessage)
	}
}

// handleYourTurn stores the action token from a YOUR TURN message.
func handleYourTurn(message string) {
	parts := strings.SplitN(message, ";", 2)
	if len(parts) < 2 {
		log.Println("Invalid YOUR TURN message format")
		return
	}

	token := strings.TrimSpace(parts[1])
	currentToken = token
}

func exitOnError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
