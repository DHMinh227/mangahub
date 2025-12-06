package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// The struct that matches your manga.json
type Manga struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	Genres        []string `json:"genres"`
	Status        string   `json:"status"`
	TotalChapters int      `json:"total_chapters"`
	Description   string   `json:"description"`
}

func main() {
	// 1. Open Database
	db, err := sql.Open("sqlite3", "../../mangahub.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 2. Load JSON
	data, err := os.ReadFile("manga.json")
	if err != nil {
		panic(err)
	}

	// 3. Parse JSON → []Manga
	var items []Manga
	if err := json.Unmarshal(data, &items); err != nil {
		panic(err)
	}

	// 4. Prepare SQL insert
	stmt, err := db.Prepare(`
        INSERT INTO manga (id, title, author, genres, status, total_chapters, description)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	// 5. Insert each manga
	for _, item := range items {

		// convert []string → JSON string
		genresJson, _ := json.Marshal(item.Genres)

		_, err := stmt.Exec(
			item.ID,
			item.Title,
			item.Author,
			string(genresJson), // <-- store JSON text
			item.Status,
			item.TotalChapters,
			item.Description,
		)
		if err != nil {
			fmt.Println("Error inserting", item.ID, ":", err)
			continue
		}
	}

	fmt.Println("Import complete.")
}
