package tcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

type ClientConn struct {
	conn     net.Conn
	lastPing time.Time
}

type ProgressSyncServer struct {
	Port      string
	Clients   map[string]*ClientConn
	Broadcast chan ProgressUpdate

	Buffer    []ProgressUpdate
	MaxBuffer int

	mu sync.Mutex
}

func NewProgressSyncServer(port string) *ProgressSyncServer {
	return &ProgressSyncServer{
		Port:      port,
		Clients:   make(map[string]*ClientConn),
		Broadcast: make(chan ProgressUpdate, 100),

		Buffer:    make([]ProgressUpdate, 0, 100),
		MaxBuffer: 100,
	}
}

func (s *ProgressSyncServer) Start() error {

	listener, err := net.Listen("tcp", s.Port)
	if err != nil {
		return err
	}

	fmt.Println("Progress Sync TCP Server running on", s.Port)

	go s.broadcastLoop()
	go s.reapDeadClients()

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
	addr := conn.RemoteAddr().String()

	client := &ClientConn{
		conn:     conn,
		lastPing: time.Now(),
	}

	s.mu.Lock()
	s.Clients[addr] = client
	for _, evt := range s.Buffer {
		data, _ := json.Marshal(evt)
		fmt.Fprintln(conn, string(data))
	}
	s.mu.Unlock()

	fmt.Println("Client connected:", addr)

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		raw := scanner.Bytes()

		var base struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &base); err != nil {
			continue
		}

		switch base.Type {
		case "PING":
			client.lastPing = time.Now()
			conn.Write([]byte(`{"type":"PONG"}` + "\n"))

		case "PROGRESS":
			var update ProgressUpdate
			if err := json.Unmarshal(raw, &update); err != nil {
				continue
			}
			update.Timestamp = time.Now().Unix()
			s.Broadcast <- update
		}
	}

	// disconnect
	s.mu.Lock()
	delete(s.Clients, addr)
	s.mu.Unlock()

	conn.Close()
	fmt.Println("Client disconnected:", addr)
}

func (s *ProgressSyncServer) broadcastLoop() {
	for update := range s.Broadcast {
		data, _ := json.Marshal(update)

		s.mu.Lock()
		// buffer always
		if len(s.Buffer) >= s.MaxBuffer {
			s.Buffer = s.Buffer[1:] // drop oldest
		}
		s.Buffer = append(s.Buffer, update)
		for _, client := range s.Clients {
			fmt.Fprintln(client.conn, string(data))
		}
		s.mu.Unlock()
	}
}
func (s *ProgressSyncServer) reapDeadClients() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		now := time.Now()

		s.mu.Lock()
		for addr, c := range s.Clients {
			if now.Sub(c.lastPing) > 15*time.Second {
				c.conn.Close()
				delete(s.Clients, addr)
			}
		}
		s.mu.Unlock()
	}
}
