// cmd/cli/main.go
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "mangahub/proto/manga" // <-- adjust if your generated proto package path differs

	"google.golang.org/grpc"
)

const (
	HTTP_API     = "http://localhost:8080"
	GRPC_ADDRESS = "localhost:9092"
)

// global session
var (
	accessToken    string
	refreshToken   string
	currentUser    string // username used as user_id in gRPC calls
	grpcClient     pb.MangaServiceClient
	lastMangaID    string
	notifyCancelCh chan struct{} // channel to cancel previous notification timer
	notifyMu       sync.Mutex    // mutex for notification state
	notifyShowing  bool          // track if notification is currently showing
)

// ==================================
// clear screen
// ==================================
func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func authHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
}

func refreshAccessToken() error {
	body, _ := json.Marshal(map[string]string{
		"refresh_token": refreshToken,
	})

	resp, err := http.Post(HTTP_API+"/auth/refresh", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh failed")
	}

	var res map[string]string
	json.NewDecoder(resp.Body).Decode(&res)

	if res["access_token"] == "" {
		return fmt.Errorf("no access token")
	}

	accessToken = res["access_token"]
	return nil
}

func doAuthRequest(req *http.Request) (*http.Response, error) {
	authHeader(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	resp.Body.Close()

	if err := refreshAccessToken(); err != nil {
		return nil, err
	}

	authHeader(req)
	return http.DefaultClient.Do(req)
}

// ==================================
// Utilities
// ==================================
func input(prompt string) string {
	fmt.Print(prompt)
	r := bufio.NewReader(os.Stdin)
	s, _ := r.ReadString('\n')
	return strings.TrimSpace(s)
}

func printHeader(title string) {
	fmt.Println()
	fmt.Println("====================================================")
	fmt.Println("  " + title)
	fmt.Println("====================================================")
}

// ==================================
// HTTP Auth (register/login/logout)
// ==================================
func registerUser() {
	clearScreen()
	printHeader("REGISTER")

	username := input("Enter username: ")
	password := input("Enter password: ")

	payload := map[string]string{
		"username": username,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	res, err := http.Post(HTTP_API+"/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer res.Body.Close()

	var reply map[string]interface{}
	json.NewDecoder(res.Body).Decode(&reply)

	if res.StatusCode == http.StatusCreated || res.StatusCode == http.StatusOK {
		fmt.Println("Account created successfully!")
		// some server return tokens on register; try extract
		if t, ok := reply["access_token"].(string); ok {
			accessToken = t
			if rt, ok2 := reply["refresh_token"].(string); ok2 {
				refreshToken = rt
			}
			currentUser = getUserIDFromJWT(accessToken)
			fmt.Println("Logged in as:", username)
		}
	} else {
		fmt.Println("Register failed:", reply["error"])
	}
}

func getUserIDFromJWT(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	var claims struct {
		UserID string `json:"user_id"`
	}

	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}

	return claims.UserID
}

func loginUser() {
	clearScreen()
	printHeader("LOGIN")

	username := input("Username: ")
	password := input("Password: ")

	payload := map[string]string{
		"username": username,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	res, err := http.Post(HTTP_API+"/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer res.Body.Close()

	var reply map[string]interface{}
	json.NewDecoder(res.Body).Decode(&reply)

	// support both {"token": "..."} and {"access_token":"..."}
	if t, ok := reply["access_token"].(string); ok {
		accessToken = t
		if rt, ok2 := reply["refresh_token"].(string); ok2 {
			refreshToken = rt
		}
		currentUser = getUserIDFromJWT(accessToken)

		fmt.Println("Login successful!")
	} else if t2, ok := reply["token"].(string); ok {
		accessToken = t2
		currentUser = getUserIDFromJWT(accessToken)
		fmt.Println("Login successful!")
	} else {
		fmt.Println("Login failed:", reply["error"])
	}
}

func logoutUser() {
	clearScreen()
	if refreshToken == "" {
		fmt.Println("No refresh token stored; you may be logged out already.")
		accessToken = ""
		currentUser = ""
		refreshToken = ""
		return
	}

	payload := map[string]string{"refresh_token": refreshToken}
	body, _ := json.Marshal(payload)
	res, err := http.Post(HTTP_API+"/auth/logout", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Println("Error calling logout:", err)
		// still clear local session
		accessToken = ""
		currentUser = ""
		refreshToken = ""
		return
	}
	defer res.Body.Close()
	accessToken = ""
	currentUser = ""
	refreshToken = ""
	fmt.Println("Logged out.")
}

// ==================================
// gRPC calls (Search, GetManga, UpdateProgress)
// ==================================

func searchMangaGRPC(client pb.MangaServiceClient) {
	clearScreen()
	printHeader("MANGA SEARCH (gRPC)")

	query := input("Enter keyword (empty = all): ")
	genre := input("Genre filter (empty = ignore): ")
	status := input("Status filter (empty = ignore): ")
	limit := input("Limit (default 50): ")

	lim := int32(50)
	if limit != "" {
		n, _ := strconv.Atoi(limit)
		lim = int32(n)
	}

	req := &pb.SearchRequest{
		Query:  query,
		Genre:  genre,
		Status: status,
		Limit:  lim,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.SearchManga(ctx, req)
	if err != nil {
		fmt.Println("gRPC error:", err.Error())
		return
	}

	clearScreen()
	printHeader("SEARCH RESULTS")

	if len(resp.Results) == 0 {
		fmt.Println("No results found.")
		time.Sleep(1 * time.Second)
		return
	}

	for _, m := range resp.Results {
		fmt.Printf("[%s] %s (%s)\n", m.Id, m.Title, m.Status)
	}

	fmt.Println("\nOptions:")
	fmt.Println("1) MANGA INFO")
	fmt.Println("2) MAIN MENU")

	choice := input("> ")

	switch choice {
	case "1":
		lastMangaID = input("Enter manga ID: ")
		mangaInfoGRPC(client) // pass through
	}
}

func mangaInfoGRPC(client pb.MangaServiceClient) {
	clearScreen()

	id := lastMangaID
	if id == "" {
		id = input("Enter manga ID: ")
		lastMangaID = id
	}

	req := &pb.GetMangaRequest{Id: id}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	m, err := client.GetManga(ctx, req)
	if err != nil {
		fmt.Println("gRPC error:", err.Error())
		return
	}

	// FETCH PROGRESS
	progReq := &pb.GetProgressRequest{
		UserId:  currentUser,
		MangaId: m.Id,
	}

	progResp, _ := client.GetProgress(ctx, progReq)

	clearScreen()
	printHeader("MANGA INFO")

	fmt.Println("ID:", m.Id)
	fmt.Println("Title:", m.Title)
	fmt.Println("Author:", m.Author)
	fmt.Println("Genres:", m.Genres)
	fmt.Println("Status:", m.Status)
	fmt.Println("Total Chapters:", m.TotalChapters)

	fmt.Println("\nDescription:")
	fmt.Println(m.Description)

	if progResp.Exists {
		fmt.Println("\nYour Progress: Chapter", progResp.CurrentChapter)
	}

	fmt.Println("\nOptions:")
	fmt.Println("1) UPDATE PROGRESS")
	fmt.Println("2) MAIN MENU")

	choice := input("> ")

	switch choice {
	case "1", "progress":
		updateProgressHTTP()
		lastMangaID = "" // reset after update
	default:
		lastMangaID = "" // reset when returning to menu
		return
	}

}

func updateProgressHTTP() {
	clearScreen()
	printHeader("UPDATE PROGRESS")

	if lastMangaID == "" {
		lastMangaID = input("Enter manga ID: ")
	}

	chapterStr := input("Enter current chapter: ")
	ch, _ := strconv.Atoi(chapterStr)

	payload := map[string]interface{}{
		"manga_id": lastMangaID,
		"chapter":  ch,
	}

	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest(
		"POST",
		HTTP_API+"/users/progress",
		bytes.NewBuffer(data),
	)

	resp, err := doAuthRequest(req)
	if err != nil {
		fmt.Println("Request failed:", err)
		time.Sleep(time.Second)
		return
	}
	defer resp.Body.Close()

	var res map[string]string
	json.NewDecoder(resp.Body).Decode(&res)

	fmt.Println(res["message"])
	time.Sleep(time.Second)
}

// ==================================
// Menus
// ==================================
func mainMenu() {
	clearScreen()
	for {
		printHeader("MAIN MENU")
		fmt.Println("Options:")
		fmt.Println("1) SEARCH")
		fmt.Println("2) MANGA INFO")
		fmt.Println("3) UPDATE_PROGRESS")
		fmt.Println("4) LOGOUT")
		fmt.Println("5) EXIT")

		cmd := strings.ToLower(input("> "))

		switch cmd {
		case "1", "search":
			searchMangaGRPC(grpcClient)
			lastMangaID = ""
		case "2", "info", "mangainfo":
			lastMangaID = ""
			mangaInfoGRPC(grpcClient)
		case "3", "progress":
			updateProgressHTTP()
			lastMangaID = ""
		case "4", "logout":
			logoutUser()
			return
		case "5", "exit":
			os.Exit(0)
		}
	}
}

func welcomeMenu() {
	clearScreen()

	for {
		printHeader("WELCOME TO MANGAHUB CLI")
		fmt.Println("Options:")
		fmt.Println("1) REGISTER")
		fmt.Println("2) LOGIN")
		fmt.Println("3) EXIT")

		cmd := strings.ToLower(input("> "))

		switch cmd {
		case "register", "1":
			registerUser()
		case "login", "2":
			loginUser()
			if accessToken != "" {
				mainMenu()
			}
		case "exit", "3":
			os.Exit(0)
		default:
			fmt.Println("Invalid choice.")
		}
	}
}

// ==================================
// bootstrap gRPC client
// ==================================

func initGRPC() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	grpcClient = pb.NewMangaServiceClient(conn)
}

var udpConn *net.UDPConn

func startUDP() {
	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:9091")
	if err != nil {
		fmt.Println("UDP resolve error:", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		fmt.Println("UDP dial error:", err)
		return
	}
	udpConn = conn

	fmt.Println("UDP local addr:", conn.LocalAddr()) // <-- see your port

	// keep sending REGISTER until ACK
	ackCh := make(chan struct{}, 1)
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ackCh:
				return
			case <-t.C:
				registerMsg := map[string]string{"type": "REGISTER"}
				data, _ := json.Marshal(registerMsg)
				conn.Write(data)
			}
		}
	}()

	go func() {
		buf := make([]byte, 2048)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				continue
			}

			var msg struct {
				Type    string `json:"type"`
				ID      string `json:"id"`
				Message string `json:"message"`
				Title   string `json:"title"`
			}
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				continue
			}

			switch msg.Type {
			case "REGISTER_ACK":
				fmt.Println("🔔 UDP registered successfully")
				select {
				case ackCh <- struct{}{}:
				default:
				}

			case "NOTIFY", "NEW_MANGA", "manga_added", "chapter_release":
				text := msg.Message
				if text == "" && msg.Title != "" {
					text = msg.Title
				}
				if text == "" {
					text = "(no message)"
				}

				// Cancel any previous notification timer and clear old notification
				notifyMu.Lock()
				if notifyCancelCh != nil {
					close(notifyCancelCh)
				}
				// If notification is showing, clear the 4 lines (notification + prompt)
				if notifyShowing {
					// Move up 4 lines and clear them
					fmt.Print("\033[4A") // Move up 4 lines
					fmt.Print("\033[J")  // Clear from cursor to end of screen
				}
				notifyCancelCh = make(chan struct{})
				cancelCh := notifyCancelCh
				notifyShowing = true
				notifyMu.Unlock()

				// Print notification inline
				fmt.Println("\n🔔 ============================================")
				fmt.Println("   NEW MANGA: " + text)
				fmt.Println("🔔 ============================================")
				fmt.Print("> ")

				// Send ACK immediately
				ack := map[string]string{"type": "ACK", "id": msg.ID}
				ackData, _ := json.Marshal(ack)
				conn.Write(ackData)

				// Wait 10 seconds then clear notification
				go func(ch chan struct{}) {
					select {
					case <-time.After(10 * time.Second):
						notifyMu.Lock()
						if notifyCancelCh == ch && notifyShowing {
							// Move up 4 lines and clear them
							fmt.Print("\033[4A") // Move up 4 lines
							fmt.Print("\033[J")  // Clear from cursor to end of screen
							fmt.Print("> ")      // Restore prompt
							notifyShowing = false
							notifyCancelCh = nil
						}
						notifyMu.Unlock()
					case <-ch:
						// New notification came in, do nothing
					}
				}(cancelCh)

			default:
				// Silently ignore unknown message types
			}
		}
	}()
}

func main() {
	initGRPC()
	startUDP()
	welcomeMenu()

}
