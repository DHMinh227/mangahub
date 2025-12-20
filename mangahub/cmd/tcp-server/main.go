package main

import (
	"log"
	"mangahub/internal/tcp"
)

func main() {
	server := tcp.NewProgressSyncServer(":9090")
	log.Println("Starting TCP Sync Server on :9090")
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}

}
