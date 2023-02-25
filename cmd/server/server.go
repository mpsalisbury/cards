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
	hostport := "localhost:" + port
	log.Printf("Listening on %s", hostport)
	listener, err := net.Listen("tcp", hostport)
	if err != nil {
		log.Fatalf("net.Listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCardGameServiceServer(grpcServer, server.NewCardGameService())
	if err = grpcServer.Serve(listener); err != nil {
		log.Fatal(err)
	}
}
