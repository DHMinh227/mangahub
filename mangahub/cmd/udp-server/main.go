package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"mangahub/internal/udp"
)

func main() {
	server := udp.NewNotificationServer(":9091")

	// Start UDP server (client registration)
	go func() {
		if err := server.Start(); err != nil {
			log.Fatal("UDP server error:", err)
		}
	}()

	// HTTP control server on 127.0.0.1:9094
	http.HandleFunc("/broadcast", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, _ := io.ReadAll(r.Body)

		var note udp.Notification
		if err := json.Unmarshal(body, &note); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if note.Timestamp == 0 {
			note.Timestamp = time.Now().Unix()
		}

		server.Broadcast(note)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	log.Println("UDP registration port :9091")
	log.Println("UDP broadcast HTTP control :9094")

	log.Fatal(http.ListenAndServe(":9094", nil))
}
