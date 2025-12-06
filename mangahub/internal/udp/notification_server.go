package udp

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

type Notification struct {
	Type      string `json:"type"`
	MangaID   string `json:"manga_id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type NotificationServer struct {
	Port    string
	Clients []net.UDPAddr
	mu      sync.Mutex
}

func NewNotificationServer(port string) *NotificationServer {
	return &NotificationServer{
		Port:    port,
		Clients: make([]net.UDPAddr, 0),
	}
}

func (s *NotificationServer) Start() error {
	addr, err := net.ResolveUDPAddr("udp", s.Port)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	fmt.Println("UDP Notification Server running on", s.Port)

	buf := make([]byte, 1024)

	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Read error:", err)
			continue
		}

		msg := string(buf[:n])

		if msg == "REGISTER" {
			s.addClient(*clientAddr)
			fmt.Println("Client registered:", clientAddr)
			continue
		}

		fmt.Println("Unknown message from client:", msg)
	}
}

func (s *NotificationServer) addClient(addr net.UDPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range s.Clients {
		if c.String() == addr.String() {
			return
		}
	}

	s.Clients = append(s.Clients, addr)
}

func (s *NotificationServer) Broadcast(note Notification) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, _ := json.Marshal(note)

	for _, client := range s.Clients {
		conn, err := net.DialUDP("udp", nil, &client)
		if err != nil {
			fmt.Println("Broadcast error:", err)
			continue
		}

		_, _ = conn.Write(data)
		conn.Close()
	}
}
