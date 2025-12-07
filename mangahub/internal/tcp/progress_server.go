package tcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type ProgressUpdate struct {
	UserID    string `json:"user_id"`
	MangaID   string `json:"manga_id"`
	Chapter   int    `json:"chapter"`
	Timestamp int64  `json:"timestamp"`
}

type ProgressSyncServer struct {
	Port        string
	Connections map[string]net.Conn
	Broadcast   chan ProgressUpdate
	mu          sync.Mutex
}

func NewProgressSyncServer(port string) *ProgressSyncServer {
	return &ProgressSyncServer{
		Port:        port,
		Connections: make(map[string]net.Conn),
		Broadcast:   make(chan ProgressUpdate, 10),
	}
}

func (s *ProgressSyncServer) Start() error {
	listener, err := net.Listen("tcp", s.Port)
	if err != nil {
		return err
	}

	fmt.Println("Progress Sync TCP Server running on", s.Port)

	go s.broadcastLoop()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}

		go s.handleClient(conn)
	}
}

func (s *ProgressSyncServer) handleClient(conn net.Conn) {
	clientAddr := conn.RemoteAddr().String()

	s.mu.Lock()
	s.Connections[clientAddr] = conn
	s.mu.Unlock()

	fmt.Println("Client connected:", clientAddr)

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		msg := scanner.Bytes()

		var update ProgressUpdate
		if err := json.Unmarshal(msg, &update); err != nil {
			fmt.Fprintln(conn, "INVALID_JSON")
			continue
		}

		update.Timestamp = time.Now().Unix()

		// 1. Save progress using HTTP API
		go sendProgressToAPI(update)

		// 2. Broadcast to all TCP clients
		s.Broadcast <- update
	}

	// client disconnected
	s.mu.Lock()
	delete(s.Connections, clientAddr)
	s.mu.Unlock()

	fmt.Println("Client disconnected:", clientAddr)
}

func (s *ProgressSyncServer) broadcastLoop() {
	for update := range s.Broadcast {
		data, _ := json.Marshal(update)

		s.mu.Lock()
		for addr, conn := range s.Connections {
			fmt.Fprintln(conn, string(data))
			_ = addr
		}
		s.mu.Unlock()
	}
}

func sendProgressToAPI(update ProgressUpdate) {
	jsonData, _ := json.Marshal(map[string]interface{}{
		"user_id":  update.UserID,
		"manga_id": update.MangaID,
		"chapter":  update.Chapter,
	})

	_, err := http.Post(
		"http://localhost:8080/users/progress", // your API
		"application/json",
		bytes.NewBuffer(jsonData),
	)

	if err != nil {
		fmt.Println("Failed to sync with API:", err)
	}
}
