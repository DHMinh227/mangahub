package main

import (
	"log"
	"mangahub/internal/auth"
	"mangahub/internal/manga"
	"mangahub/internal/user"
	"mangahub/pkg/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	db := database.InitDB("mangahub.db")
	defer db.Close()

	router := gin.Default()

	// --- CORS ---
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// --- Public Auth ---
	router.POST("/auth/register", auth.RegisterHandler(db))
	router.POST("/auth/login", auth.LoginHandler(db))
	router.POST("/auth/logout", auth.LogoutHandler(db))
	router.POST("/auth/refresh", auth.RefreshHandler(db))

	// --- Protected Routes ---
	authRequired := router.Group("/")
	authRequired.Use(auth.AuthMiddleware())

	// Protected: update / get progress
	user.RegisterProgressRoutes(authRequired, db)

	// Public manga routes
	manga.RegisterRoutes(router, db)

	log.Println("🌐 Starting MangaHub server at http://localhost:8080")
	router.Run(":8080")
}
