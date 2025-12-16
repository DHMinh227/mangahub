package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const API = "http://localhost:8080"

var (
	accessToken  string
	refreshToken string
)

func input(prompt string) string {
	fmt.Print(prompt)
	r := bufio.NewReader(os.Stdin)
	s, _ := r.ReadString('\n')
	return strings.TrimSpace(s)
}

func authHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
}
func getRoleFromJWT(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	var claims struct {
		Role string `json:"role"`
	}

	_ = json.Unmarshal(payload, &claims)
	return claims.Role
}
func login() {
	username := input("Admin username: ")
	password := input("Password: ")

	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})

	resp, err := http.Post(API+"/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var res map[string]string
	json.NewDecoder(resp.Body).Decode(&res)

	accessToken = res["access_token"]
	refreshToken = res["refresh_token"]

	if accessToken == "" {
		fmt.Println("Login failed")
		os.Exit(1)
	}

	role := getRoleFromJWT(accessToken)
	if role != "admin" {
		fmt.Println("Access denied: admin only")
		os.Exit(1)
	}

	fmt.Println("Admin login successful")
}
func refreshAccessToken() error {
	body, _ := json.Marshal(map[string]string{
		"refresh_token": refreshToken,
	})

	resp, err := http.Post(API+"/auth/refresh", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("refresh failed")
	}

	var res map[string]string
	json.NewDecoder(resp.Body).Decode(&res)

	newToken := res["access_token"]
	if newToken == "" {
		return fmt.Errorf("no access token returned")
	}

	accessToken = newToken
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

	// try refresh
	if err := refreshAccessToken(); err != nil {
		return nil, err
	}

	// retry with new token
	authHeader(req)
	return http.DefaultClient.Do(req)
}

func addManga() {
	payload := map[string]interface{}{
		"id":             input("ID: "),
		"title":          input("Title: "),
		"author":         input("Author: "),
		"genres":         strings.Split(input("Genres (comma): "), ","),
		"status":         input("Status: "),
		"total_chapters": mustInt(input("Total chapters: ")),
		"description":    input("Description: "),
	}

	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", API+"/admin/manga", bytes.NewBuffer(data))

	resp, err := doAuthRequest(req)
	if err != nil {
		fmt.Println("Request failed:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Add manga status:", resp.Status)
}

func deleteManga() {
	id := input("Manga ID to delete: ")

	req, _ := http.NewRequest("DELETE", API+"/admin/manga/"+id, nil)

	resp, err := doAuthRequest(req)
	if err != nil {
		fmt.Println("Request failed:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Delete status:", resp.Status)
}

func mustInt(s string) int {
	var n int
	fmt.Sscan(s, &n)
	return n
}

func main() {
	login()

	for {
		fmt.Println("\nADMIN MENU")
		fmt.Println("1) Add manga")
		fmt.Println("2) Delete manga")
		fmt.Println("3) Exit")

		switch input("> ") {
		case "1":
			addManga()
		case "2":
			deleteManga()
		case "3":
			return
		}
	}
}
