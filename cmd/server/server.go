package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	"github.com/mpsalisbury/cards/pkg/discovery"
	pb "github.com/mpsalisbury/cards/pkg/proto"
	"github.com/mpsalisbury/cards/pkg/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}
	host := server.GetOutboundIP()
	hostport := fmt.Sprintf("%s:%s", host, port)
	log.Printf("Listening on %s", hostport)
	listener, err := net.Listen("tcp", hostport)
	if err != nil {
		log.Fatalf("net.Listen: %v", err)
	}

	ad, err := discovery.AdvertiseService(listener.Addr().String())
	if err != nil {
		log.Fatalf("AdvertiseService: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCardGameServiceServer(grpcServer, server.NewCardGameService())
	if err = grpcServer.Serve(listener); err != nil {
		log.Fatal(err)
	}
	ad.Close()
}
