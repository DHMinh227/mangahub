package main

import (
	"log"
	"mangahub/internal/auth"
	"mangahub/pkg/database"

	"github.com/gin-gonic/gin"
)

func main() {
	db := database.InitDB("mangahub.db")
	defer db.Close()

	router := gin.Default()

	// Simple route
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// User registration
	router.POST("/auth/register", auth.RegisterHandler(db))

	log.Println("Starting MangaHub server at http://localhost:8080")
	router.Run(":8080")
}
