package main

import (
	"log"
	"mangahub/internal/udp"
)

func main() {
	server := udp.NewNotificationServer(":9001")
	log.Println("Starting UDP Notification Server on :9001")
	server.Start()
}
