package main

import (
	"log"
	"mangahub/internal/auth"
	grpcserver "mangahub/internal/grpc"
	"mangahub/internal/manga"
	"mangahub/internal/udp"
	"mangahub/internal/user"
	"mangahub/pkg/database"
	pb "mangahub/proto/manga"
	"path/filepath"

	"net"
	"net/http"

	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func main() {

	dbPath, _ := filepath.Abs("mangahub.db")
	db := database.InitDB(dbPath)

	defer db.Close()

	udpServer := udp.NewNotificationServer(":9091")

	go func() {
		log.Println("🔔 UDP notification server starting on :9091")
		if err := udpServer.Start(); err != nil {
			log.Println("UDP server error:", err)
		}
	}()

	log.Println("HTTP API DB path:", dbPath)

	router := gin.Default()
	for _, r := range router.Routes() {
		log.Println(r.Method, r.Path)
	}

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

	grpcServer := grpc.NewServer()
	pb.RegisterMangaServiceServer(grpcServer, &grpcserver.GRPCMangaServer{DB: db})

	go func() {
		lis, _ := net.Listen("tcp", ":50051")
		grpcServer.Serve(lis)
	}()

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("🛑 Shutting down API server...")

		grpcServer.GracefulStop()
		db.Close()

		os.Exit(0)
	}()

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

	// ADMIN
	admin := router.Group("/admin")
	admin.Use(auth.AuthMiddleware()) // 1️⃣ parse JWT, set claims
	admin.Use(auth.AdminOnly())      // 2️⃣ check role
	manga.RegisterAdminRoutes(admin, db, udpServer)

	// Public manga routes
	manga.RegisterRoutes(router, db)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	log.Println("🌐 Starting MangaHub server at http://localhost:8080")

	router.Run(":8080")

}
