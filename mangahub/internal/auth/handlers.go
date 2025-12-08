package auth

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func RegisterHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "invalid input"})
			return
		}

		hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		res, err := db.Exec(`INSERT INTO users (username, password_hash) VALUES (?, ?)`,
			req.Username, hash)

		if err != nil {
			c.JSON(409, gin.H{"error": "username exists"})
			return
		}

		id, _ := res.LastInsertId()
		userID := fmt.Sprintf("%d", id)

		access, _ := CreateAccessToken(userID, req.Username)
		refresh, _ := CreateRefreshToken(db, userID)

		c.JSON(201, gin.H{
			"access_token":  access,
			"refresh_token": refresh,
		})
	}
}

func LoginHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "invalid input"})
			return
		}

		var id int
		var hash string

		err := db.QueryRow(`SELECT id, password_hash FROM users WHERE username = ?`,
			req.Username).Scan(&id, &hash)

		if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
			c.JSON(401, gin.H{"error": "invalid credentials"})
			return
		}

		userID := fmt.Sprintf("%d", id)

		access, _ := CreateAccessToken(userID, req.Username)
		refresh, _ := CreateRefreshToken(db, userID)

		c.JSON(200, gin.H{
			"access_token":  access,
			"refresh_token": refresh,
		})
	}
}

func RefreshHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Token string `json:"refresh_token"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "invalid"})
			return
		}

		userID, err := ValidateRefreshToken(db, req.Token)
		if err != nil {
			c.JSON(401, gin.H{"error": "invalid refresh token"})
			return
		}

		var username string
		_ = db.QueryRow(`SELECT username FROM users WHERE id = ?`, userID).Scan(&username)

		access, _ := CreateAccessToken(userID, username)

		c.JSON(200, gin.H{"access_token": access})
	}
}

func LogoutHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Token string `json:"refresh_token"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "invalid"})
			return
		}

		_ = RevokeRefreshToken(db, req.Token)

		c.JSON(200, gin.H{"message": "logged out"})
	}
}
