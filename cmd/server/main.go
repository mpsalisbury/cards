package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	pb "github.com/mpsalisbury/cards/internal/game/proto"
	"github.com/mpsalisbury/cards/internal/game/server"
)

func main() {
	log.Printf("game: starting server...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
		log.Printf("Defaulting to port %s", port)
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("net.Listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCardGameServiceServer(grpcServer, server.NewCardGameService())
	if err = grpcServer.Serve(listener); err != nil {
		log.Fatal(err)
	}
}
