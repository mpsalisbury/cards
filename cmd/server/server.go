package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	pb "github.com/mpsalisbury/cards/pkg/proto"
	"github.com/mpsalisbury/cards/pkg/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}
	log.Printf("Listening on port %s", port)
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
