package hearts

import (
	"context"
	"fmt"
	"log"

	"github.com/mpsalisbury/cards/pkg/cards"
	"github.com/mpsalisbury/cards/pkg/client"
)

// BasicPlayer implements simple basic strategy.

func NewBasicPlayer() client.GameCallbacks {
	return &basicPlayer{}
}

type basicPlayer struct {
	client.UnimplementedGameCallbacks
}

func (c basicPlayer) HandleYourTurn(s client.Session) error {
	ctx := context.Background()
	gameState, err := s.GetGameState(ctx)
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	card := c.chooseCard(gameState)
	err = s.PlayCard(ctx, card)
	if err != nil {
		log.Fatalf("BasicPlayer chose invalid card %s\nGamestate: %v", card, gameState)
	}
	return nil
}

// Card capabilities needed.
//   Literals: 2c, qs, ks, as
//   for Cards
//     filter by suit(s)
//     filter by lower-than-value
//     filter by higher-than-value
//     lowest card
//     highest card
//     highest under value or lowest
//     containsAny(cards)
//     containsSuit
//   for Game
//     numTricksOfSuit
//     qs was played
//     currentWinningCard

func (c basicPlayer) chooseCard(gs client.GameState) cards.Card {
	return cards.ParseCardOrDie("2c")
	//	hand := gs.Players[0].Cards
	//	trick := gs.CurrentTrick

	// If we have 2c, play 2c.
	// If we have the lead
	//   If we have qs, ks, or as
	//     Play lowest card in non-spade suit, or lowest spade
	//   If we have a spade, play highest spade
	//   Play lowest card
	// If we're following
	//   If spades is led and qs is still available and we have spades
	//     if we have qs and as or ks is played, play qs
	//     if we have qs, play high spade not queen
	//       else play queen
	//     if we don't have qs,
	//       if we're the last card in the trick, play high spade
	//       else play high spade under qs
	//       else play high spade
	//   If we have the led suit and qs is available
	//     If first trick of suit, play high
	//     else play highest under winning card or lowest card
	//   If we have the led suit and qs is not available
	//     If first two tricks of suit, play high
	//     else play highest under winning card or lowest card
	//   If we don't have led suit
	//     If we have qs, play qs
	//     If we have ks or as, play those
	//     If we have hearts over 8, play highest
	//     Play highest card of suit with highest low card
}
