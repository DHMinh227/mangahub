package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func startDBHealthCheck(db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := db.Ping(); err != nil {
				log.Printf("Database health check failed: %v", err)
			}
		}
	}()
}
func InitDB(path string) *sql.DB {
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=1")
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(25) // adjust as needed
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	// Verify connection
	if err := db.Ping(); err != nil {
		log.Fatalf("db ping failed: %v", err)
	}

	fmt.Println("Database initialized successfully")

	// Create required tables if missing
	createTables(db)

	return db
}

func createTables(db *sql.DB) {
	// users, manga assumed to already exist — add user_progress and refresh_tokens
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS user_progress (
			user_id TEXT,
			manga_id TEXT,
			current_chapter INTEGER,
			status TEXT,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, manga_id)
		);`,
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			token TEXT PRIMARY KEY,
			user_id TEXT,
			expires_at TIMESTAMP
		);`,
	}

	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			log.Fatalf("failed to create table: %v", err)
		}
	}
}
