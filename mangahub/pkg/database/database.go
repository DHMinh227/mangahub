package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// InitDB connects to SQLite and creates tables if they don't exist.
func InitDB(path string) *sql.DB {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal("❌ Cannot open database:", err)
	}

	createTables := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		password_hash TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS manga (
		id TEXT PRIMARY KEY,
		title TEXT,
		author TEXT,
		genres TEXT,
		status TEXT,
		total_chapters INTEGER,
		description TEXT
	);
	`

	_, err = db.Exec(createTables)
	if err != nil {
		log.Fatal("❌ Failed to create tables:", err)
	}

	log.Println(" Database initialized successfully")
	return db
}
