package main

import (
	"log"
	"mangahub/internal/api"
	"mangahub/internal/tcp"
)

func main() {
	// Create tcp server ONCE
	tcpServer := tcp.NewProgressSyncServer(":8082")

	// Start TCP concurrently
	go func() {
		log.Println("TCP server starting on :8082")
		if err := tcpServer.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	// Start API server and give it access to tcp instance
	log.Println("API server starting on :8080")
	api.StartHTTPServer(tcpServer)
}
