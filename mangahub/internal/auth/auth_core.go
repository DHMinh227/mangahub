package auth

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// CoreLogin is a plain Go login function for TCP, gRPC, CLI, etc.
func CoreLogin(db *sql.DB, username, password string) error {
	var id int
	var passwordHash string

	err := db.QueryRow(
		"SELECT id, password_hash FROM users WHERE username = ?",
		username,
	).Scan(&id, &passwordHash)

	if err == sql.ErrNoRows {
		return errors.New("invalid username or password")
	}
	if err != nil {
		return err
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		return errors.New("invalid username or password")
	}

	return nil // success
}
