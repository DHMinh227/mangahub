package udp

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
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

// Broadcast sends a notification to all registered clients.
// It copies the client list under lock, then sends to each client without holding the lock.
// Sends are done concurrently with a small goroutine pool to avoid flood when many clients exist.
func (s *NotificationServer) Broadcast(note Notification) {
	data, err := json.Marshal(note)
	if err != nil {
		fmt.Println("UDP: failed to marshal notification:", err)
		return
	}

	// copy clients under lock
	s.mu.Lock()
	clients := make([]net.UDPAddr, len(s.Clients))
	copy(clients, s.Clients)
	s.mu.Unlock()

	// send to each client concurrently but bounded
	// bounded concurrency avoids creating thousands of goroutines at once
	const maxWorkers = 10
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for _, client := range clients {
		wg.Add(1)
		sem <- struct{}{}
		go func(c net.UDPAddr) {
			defer wg.Done()
			defer func() { <-sem }()

			conn, err := net.DialUDP("udp", nil, &c)
			if err != nil {
				fmt.Println("UDP: dial error to", c.String(), ":", err)
				return
			}
			defer conn.Close()

			// set a write deadline so a slow client cannot stall forever
			_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))

			if _, err := conn.Write(data); err != nil {
				fmt.Println("UDP: write error to", c.String(), ":", err)
				return
			}
		}(client)
	}
	wg.Wait()
}
