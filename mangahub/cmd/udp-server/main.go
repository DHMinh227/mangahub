package main

import (
	"log"
	"mangahub/internal/udp"
	"time"
)

func main() {
	server := udp.NewNotificationServer(":9091")

	// Start UDP listener
	go func() {
		if err := server.Start(); err != nil {
			log.Fatal("UDP server error:", err)
		}
	}()

	log.Println("UDP Notification Server running on :9091")

	// TEST BROADCAST EVERY 10s
	for {
		server.Broadcast(udp.Notification{
			Type:      "chapter_release",
			MangaID:   "30001",
			Message:   "New chapter released!",
			Timestamp: time.Now().Unix(),
		})

		time.Sleep(100 * time.Second)
	}
}
