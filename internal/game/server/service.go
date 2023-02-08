package server

import (
	"context"
	"log"

	pb "github.com/mpsalisbury/cards/internal/game/proto"
)

func NewCardGameService() pb.CardGameServiceServer {
	return &cardGameService{}
}

type cardGameService struct {
	pb.UnimplementedCardGameServiceServer
}

func (cardGameService) Ping(ctx context.Context, request *pb.PingRequest) (*pb.PingResponse, error) {
	log.Printf("Got ping %s", request.GetMessage())
	return &pb.PingResponse{Message: "Pong"}, nil
}
