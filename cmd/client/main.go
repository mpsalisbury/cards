package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/mpsalisbury/cards/internal/game/proto"
)

var (
	logger     = log.New(os.Stdout, "", 0)
	serverAddr = flag.String("server", "api.cards.salisburyclan.com:443", "Server address (host:port)")
	// Raw server: "cards-api-5g5wrbokbq-uw.a.run.app:443"
	insecure = flag.Bool("insecure", false, "Use insecure connection to server")
	local    = flag.Bool("local", false, "Override serverAddr and insecure connection for local server")
)

func main() {
	flag.Parse()

	var opts []grpc.DialOption
	if *local {
		*serverAddr = "localhost:50051"
		*insecure = true
	}
	if *insecure {
		opts = append(opts, grpc.WithInsecure())
	} else {
		cred := credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: false,
		})
		opts = append(opts, grpc.WithTransportCredentials(cred))
	}

	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		logger.Printf("Failed to dial: %v", err)
		return
	}
	defer conn.Close()
	client := pb.NewCardGameServiceClient(conn)
	ping(client)
}

func ping(client pb.CardGameServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Ping(ctx, &pb.PingRequest{Message: "howdy"})
	if err != nil {
		logger.Fatalf("Error while executing ping: %v", err)
	}

	logger.Printf("Got ping result message %s\n", resp.GetMessage())
}
