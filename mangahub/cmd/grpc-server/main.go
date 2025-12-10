package main

import (
	"flag"
	"log"
	"net"

	grpcinternal "mangahub/internal/grpc"
	"mangahub/pkg/database"
	pb "mangahub/proto/manga"

	"google.golang.org/grpc"
)

func main() {
	addr := flag.String("addr", ":9092", "gRPC listen address")
	flag.Parse()

	// init DB (reuse your package)
	db := database.InitDB("mangahub.db")
	defer db.Close()

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("failed to listen %v", err)
	}

	grpcServer := grpc.NewServer()
	svc := &grpcinternal.GRPCMangaServer{DB: db}

	pb.RegisterMangaServiceServer(grpcServer, svc)

	log.Printf("gRPC server listening on %s", *addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}
