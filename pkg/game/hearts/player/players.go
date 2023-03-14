package player

import (
	"context"
	"fmt"
	"log"

	"github.com/mpsalisbury/cards/pkg/cards"
	"github.com/mpsalisbury/cards/pkg/client"
)

// Creates a flag for specifying the player type to use.
func AddPlayerFlag(target *string, name string) {
	client.EnumFlag(target, name, []string{"basic", "term", "random"}, "Type of player logic to use")
}

// Constructs a player from a player flag value.
func NewPlayerFromFlag(playerType string, hints bool) (client.GameCallbacks, error) {
	switch playerType {
	case "", "basic":
		return newPlayer(NewBasicStrategy()), nil
	case "term":
		return NewTerminalPlayer(hints), nil
	case "random":
		return newPlayer(NewRandomStrategy()), nil
	default:
		return nil, fmt.Errorf("invalid player type %s", playerType)
	}
}

type PlayerStrategy interface {
	ChooseCardToPlay(client.GameState) cards.Card
}

func newPlayer(strategy PlayerStrategy) client.GameCallbacks {
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
