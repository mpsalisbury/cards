package player

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/mpsalisbury/cards/pkg/client"
)

// RandomPlayer plays a random card that is legal.

func NewRandomPlayer() client.GameCallbacks {
	return &randomPlayer{}
}

type randomPlayer struct {
	client.UnimplementedGameCallbacks
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (c randomPlayer) HandleYourTurn(s client.Session, gameId string) error {
	ctx := context.Background()
	gameState, err := s.GetGameState(ctx, gameId)
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	legalPlays := gameState.LegalPlays
	card := legalPlays[rand.Intn(len(legalPlays))]
	return s.PlayCard(ctx, gameId, card)
}
