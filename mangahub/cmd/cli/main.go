package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	fmt.Println("📟 MangaHub CLI Client")
	fmt.Println("Connecting to TCP server at 127.0.0.1:8081...")

	conn, err := net.Dial("tcp", "127.0.0.1:8081")
	if err != nil {
		fmt.Println("❌ Failed to connect:", err)
		return
	}
	defer conn.Close()

	fmt.Println("✅ Connected. Type commands (ping, hello, exit)")

	reader := bufio.NewReader(os.Stdin)
	serverReader := bufio.NewReader(conn)

	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "exit" {
			fmt.Println("Bye!")
			return
		}

		// Send to server
		conn.Write([]byte(input + "\n"))

		// Read server response
		response, _ := serverReader.ReadString('\n')
		fmt.Println("Server:", strings.TrimSpace(response))
	}
}
