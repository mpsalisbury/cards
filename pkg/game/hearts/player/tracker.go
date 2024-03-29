package player

import (
	"context"
	"fmt"

	"github.com/mpsalisbury/cards/pkg/client"
)

// TrackerPlayer tracks other players' past behavior.

func NewTrackerPlayer() client.GameCallbacks {
	return &trackerPlayer{}
}

type trackerPlayer struct {
	client.UnimplementedGameCallbacks
}

func (c trackerPlayer) HandleYourTurn(s client.Session, gameId string) error {
	// TODO: Implement
	ctx := context.Background()
	gameState, err := s.GetGameState(ctx, gameId)
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	for _, card := range gameState.Players[0].Cards {
		err = s.PlayCard(ctx, gameId, card)
		if err == nil {
			break
		}
	}
	return nil
}
