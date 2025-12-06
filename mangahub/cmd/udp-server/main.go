package main

import (
	"log"
	"mangahub/internal/udp"
)

func main() {
	server := udp.NewNotificationServer(":9091")
	log.Println("Starting UDP Notification Server on :9091")
	server.Start()
}
