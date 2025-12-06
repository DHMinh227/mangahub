package main

import (
	"log"
	"mangahub/internal/tcp"
)

func main() {
	server := tcp.NewProgressSyncServer(":8082")
	log.Println("Starting TCP Sync Server on :8082")
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
