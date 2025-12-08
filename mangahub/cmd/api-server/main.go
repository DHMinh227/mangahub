package main

import (
	"log"
	"mangahub/internal/auth"
	"mangahub/internal/manga"
	"mangahub/internal/user"
	"mangahub/pkg/database"

	"github.com/gin-gonic/gin"
)

func main() {
	db := database.InitDB("mangahub.db")
	defer db.Close()

	router := gin.Default()

	router.POST("/auth/register", auth.RegisterHandler(db))
	router.POST("/auth/login", auth.LoginHandler(db))
	router.POST("/auth/refresh", auth.RefreshHandler(db))
	router.POST("/auth/logout", auth.LogoutHandler(db))

	manga.RegisterRoutes(router, db)
	user.RegisterProgressRoutes(router, db)

	log.Println("🌐 Starting MangaHub server at http://localhost:8080")
	router.Run(":8080")
}
