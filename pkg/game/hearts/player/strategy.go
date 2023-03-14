package player

import (
	"context"
	"fmt"
	"log"

	"github.com/mpsalisbury/cards/pkg/cards"
	"github.com/mpsalisbury/cards/pkg/client"
)

type PlayerStrategy interface {
	ChooseCardToPlay(client.GameState) cards.Card
}

func newStrategyPlayer(strategy PlayerStrategy) client.GameCallbacks {
	return &strategyPlayer{strategy: strategy}
}

type strategyPlayer struct {
	client.UnimplementedGameCallbacks
	strategy PlayerStrategy
}

func (p strategyPlayer) HandleYourTurn(s client.Session, gameId string) error {
	ctx := context.Background()
	gameState, err := s.GetGameState(ctx, gameId)
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	card := p.strategy.ChooseCardToPlay(gameState)
	err = s.PlayCard(ctx, gameId, card)
	if err != nil {
		log.Fatalf("Player chose invalid card %s\nerror: %v\nGamestate: %v", card, err, gameState)
	}
	return nil
}
