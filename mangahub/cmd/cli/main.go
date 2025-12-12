// cmd/cli/main.go
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	accessToken  string
	refreshToken string
	currentUser  string // username used as user_id in gRPC calls
	grpcClient   pb.MangaServiceClient
	lastMangaID  string
)

// ==================================
// clear screen
// ==================================
func clearScreen() {
	fmt.Print("\033[H\033[2J")
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
			currentUser = username
			fmt.Println("Logged in as:", username)
		}
	} else {
		fmt.Println("Register failed:", reply["error"])
	}
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
		currentUser = username
		fmt.Println("Login successful!")
	} else if t2, ok := reply["token"].(string); ok {
		accessToken = t2
		currentUser = username
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

	fmt.Println("\nOptions:")
	fmt.Println("1) UPDATE PROGRESS")
	fmt.Println("2) MAIN MENU")

	choice := input("> ")

	switch choice {
	case "1":
		updateProgressGRPC(client)
	default:
		return
	}
}

func updateProgressGRPC(client pb.MangaServiceClient) {
	clearScreen()
	printHeader("UPDATE PROGRESS")

	if lastMangaID == "" {
		lastMangaID = input("Enter manga ID: ")
	}

	chapterStr := input("Enter current chapter: ")
	ch, _ := strconv.Atoi(chapterStr)

	req := &pb.ProgressRequest{
		UserId:         currentUser,
		MangaId:        lastMangaID,
		CurrentChapter: int32(ch),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.UpdateProgress(ctx, req)
	if err != nil {
		fmt.Println("Error:", err)
		time.Sleep(1 * time.Second)
		return
	}

	fmt.Println(resp.Message)
	time.Sleep(1 * time.Second)
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
		case "2", "info", "mangainfo":
			mangaInfoGRPC(grpcClient)
		case "3", "progress":
			updateProgressGRPC(grpcClient)
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

func main() {
	initGRPC()
	welcomeMenu()

}
