package udp

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/google/uuid"
)

type Notification struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	MangaID   string `json:"manga_id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type NotificationServer struct {
	Port    string
	Clients []net.UDPAddr
	conn    *net.UDPConn // keep the listening connection for broadcasting
	mu      sync.Mutex
}

func NewNotificationServer(port string) *NotificationServer {
	return &NotificationServer{
		Port:    port,
		Clients: make([]net.UDPAddr, 0),
	}
}

// Start runs the UDP server for client registration.
func (s *NotificationServer) Start() error {
	addr, err := net.ResolveUDPAddr("udp", s.Port)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	s.conn = conn // store the connection for broadcasting

	fmt.Println("UDP Notification Server running on", s.Port)

	buf := make([]byte, 2048)

	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("UDP read error:", err)
			continue
		}

		var msg struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}

		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "REGISTER":
			s.addClient(*clientAddr)
			conn.WriteToUDP([]byte(`{"type":"REGISTER_ACK"}`), clientAddr)

		case "ACK":
			fmt.Println("✅ ACK received from", clientAddr)
		case "NOTIFY":

		default:
			fmt.Println("Unknown UDP message:", msg.Type)
		}
	}
}

// thread-safe add client
func (s *NotificationServer) addClient(addr net.UDPAddr) {

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range s.Clients {
		if c.String() == addr.String() {
			return
		}
	}
	fmt.Println("📥 Total UDP clients:", len(s.Clients))

	s.Clients = append(s.Clients, addr)
	fmt.Println("📥 Total UDP clients:", len(s.Clients))

}

// Broadcast sends notifications to all clients safely.
func (s *NotificationServer) Broadcast(note Notification) {
	fmt.Println("📢 Broadcast called")
	fmt.Println("📢 Clients count:", len(s.Clients))
	fmt.Println("📢 Payload:", note)
	note.ID = uuid.NewString()

	data, err := json.Marshal(note)
	if err != nil {
		fmt.Println("UDP marshal error:", err)
		return
	}

	// copy clients under lock
	s.mu.Lock()
	clients := make([]net.UDPAddr, len(s.Clients))
	copy(clients, s.Clients)
	conn := s.conn // get the listening connection
	s.mu.Unlock()

	if conn == nil {
		fmt.Println("UDP server not started, cannot broadcast")
		return
	}

	// Send to all clients using the same listening connection
	// This ensures the source port is :9091 which the CLI expects
	for _, client := range clients {
		fmt.Println("📤 Sending to client:", client.String())
		_, err := conn.WriteToUDP(data, &client)
		if err != nil {
			fmt.Println("UDP write error to", client.String(), ":", err)
		} else {
			fmt.Println("✅ Sent to", client.String())
		}
	}
}
