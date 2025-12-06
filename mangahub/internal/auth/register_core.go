package auth

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

func CoreRegister(db *sql.DB, username, password string) error {
	// check if username exists
	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&exists)
	if err != nil {
		return err
	}
	if exists > 0 {
		return errors.New("username already exists")
	}

	// hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// insert into database
	_, err = db.Exec("INSERT INTO users(username, password_hash) VALUES (?, ?)", username, string(hash))
	if err != nil {
		return err
	}

	return nil // success
}
