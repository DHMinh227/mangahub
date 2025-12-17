package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	grpcinternal "mangahub/internal/grpc"
	"mangahub/pkg/database"
	pb "mangahub/proto/manga"

	"google.golang.org/grpc"

	health "google.golang.org/grpc/health"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {

	addr := flag.String("addr", ":9092", "gRPC listen address")
	flag.Parse()

	// init DB (reuse your package)
	dbPath, _ := filepath.Abs("mangahub.db")
	db := database.InitDB(dbPath)

	defer db.Close()

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("failed to listen %v", err)
	}

	log.Println("grpc DB path:", dbPath)
	grpcServer := grpc.NewServer()
	svc := &grpcinternal.GRPCMangaServer{DB: db}
	pb.RegisterMangaServiceServer(grpcServer, svc)

	// Health check service
	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, hs)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("🛑 Shutting down gRPC server...")
		grpcServer.GracefulStop()
		db.Close()
		os.Exit(0)
	}()

	log.Printf("gRPC server listening on %s", *addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}

}
