package hearts

import (
	"context"
	"fmt"

	"github.com/mpsalisbury/cards/pkg/client"
)

// RandomPlayer plays any random card that is legal (by trying all cards until one works).

func NewRandomPlayer() client.GameCallbacks {
	return &randomPlayer{}
}

type randomPlayer struct {
	client.UnimplementedGameCallbacks
}

func (c randomPlayer) HandleYourTurn(s client.Session) error {
	ctx := context.Background()
	gameState, err := s.GetGameState(ctx)
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	for _, card := range gameState.Players[0].Cards {
		err = s.PlayCard(ctx, card)
		if err == nil {
			break
		}
	}
	return nil
}
