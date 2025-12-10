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
)

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
	printHeader("MANGA SEARCH (gRPC)")
	fmt.Println(`Format:
search <query> --genre <genre> --status <status> --limit <number>`)

	raw := input("\nEnter search command: ")

	query := ""
	genre := ""
	status := ""
	limit := int32(50)

	parts := strings.Split(raw, " ")

	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "search":
			if i+1 < len(parts) {
				query = strings.Trim(parts[i+1], `"`)
			}
		case "--genre":
			if i+1 < len(parts) {
				genre = parts[i+1]
			}
		case "--status":
			if i+1 < len(parts) {
				status = parts[i+1]
			}
		case "--limit":
			if i+1 < len(parts) {
				n, _ := strconv.Atoi(parts[i+1])
				limit = int32(n)
			}
		}
	}

	req := &pb.SearchRequest{
		Query:  query,
		Genre:  genre,
		Status: status,
		Limit:  limit,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.SearchManga(ctx, req)
	if err != nil {
		fmt.Println("gRPC error:", err.Error())
		return
	}

	if len(resp.Results) == 0 {
		fmt.Println("No results found.")
		return
	}

	fmt.Println()
	fmt.Println("ID                 Title                    Author            Status")
	fmt.Println("------------------------------------------------------------------------")

	for _, m := range resp.Results {
		fmt.Printf("%-18s %-24s %-18s %-10s\n",
			m.Id, m.Title, m.Author, m.Status)
	}
}

func mangaInfoGRPC(client pb.MangaServiceClient) {
	id := input("Enter manga ID: ")

	req := &pb.GetMangaRequest{Id: id}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	m, err := client.GetManga(ctx, req)
	if err != nil {
		fmt.Println("gRPC error:", err.Error())
		return
	}

	fmt.Println("┌───────────────────────────────────────────────────────────┐")
	fmt.Printf("│ %-57s │\n", strings.ToUpper(m.Title))
	fmt.Println("└───────────────────────────────────────────────────────────┘")

	fmt.Println("Basic Information:")
	fmt.Println("ID:", m.Id)
	fmt.Println("Title:", m.Title)
	fmt.Println("Author:", m.Author)
	fmt.Println("Genres:", m.Genres)
	fmt.Println("Status:", m.Status)
	fmt.Println("Chapters:", m.TotalChapters)

	fmt.Println("\nDescription:")
	fmt.Println(m.Description)
}

func updateProgressGRPC(client pb.MangaServiceClient) {
	mangaId := input("Manga ID: ")
	chapterStr := input("Current chapter: ")

	ch, _ := strconv.Atoi(chapterStr)

	req := &pb.ProgressRequest{
		UserId:         currentUser,
		MangaId:        mangaId,
		CurrentChapter: int32(ch),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.UpdateProgress(ctx, req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(resp.Message)
}

// ==================================
// Helpers
// ==================================
func tokenizeArgs(raw string) []string {
	// very simple tokenizer that keeps quoted strings together
	out := []string{}
	current := ""
	inQuote := false
	for _, r := range raw {
		switch r {
		case ' ', '\t':
			if inQuote {
				current += string(r)
			} else {
				if current != "" {
					out = append(out, current)
					current = ""
				}
			}
		case '"':
			inQuote = !inQuote
		default:
			current += string(r)
		}
	}
	if current != "" {
		out = append(out, current)
	}
	return out
}

// ==================================
// Menus
// ==================================
func mainMenu() {
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
		case "search", "1":
			searchMangaGRPC(grpcClient)

		case "info", "mangainfo", "2":
			mangaInfoGRPC(grpcClient)

		case "progress", "3":
			updateProgressGRPC(grpcClient)

		case "logout", "4":
			logoutUser()
			return
		case "exit", "5":
			os.Exit(0)
		default:
			fmt.Println("Unknown command.")
		}
	}
}

func welcomeMenu() {
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
