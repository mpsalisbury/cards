package hearts

import (
	"context"
	"fmt"
	"log"

	"github.com/mpsalisbury/cards/pkg/cards"
	"github.com/mpsalisbury/cards/pkg/client"
	"golang.org/x/exp/maps"
)

// BasicPlayer implements simple basic strategy.

func NewBasicPlayer() client.GameCallbacks {
	return &basicPlayer{}
}

type basicPlayer struct {
	client.UnimplementedGameCallbacks
}

func numTricksOfSuit(gs client.GameState, suit cards.Suit) int {
	count := 0
	for _, p := range gs.Players {
		for _, t := range p.Tricks {
			if t[0].Suit == suit {
				count++
			}
		}
	}
	return count
}
func anyPlayedCard(gs client.GameState, cond func(cards.Card) bool) bool {
	for _, p := range gs.Players {
		for _, t := range p.Tricks {
			for _, c := range t {
				if cond(c) {
					return true
				}
			}
		}
	}
	return false
}
func qsWasPlayed(gs client.GameState) bool {
	return anyPlayedCard(gs, func(c cards.Card) bool { return c == cards.Cqs })
}

func (c basicPlayer) HandleYourTurn(s client.Session) error {
	ctx := context.Background()
	gameState, err := s.GetGameState(ctx)
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	card := ChooseBasicStrategyCard(gameState)
	err = s.PlayCard(ctx, card)
	if err != nil {
		log.Fatalf("BasicPlayer chose invalid card %s\nGamestate: %v", card, gameState)
	}
	return nil
}

func ChooseBasicStrategyCard(gs client.GameState) cards.Card {
	fullHand := gs.Players[0].Cards
	hand := gs.LegalPlays
	trick := gs.CurrentTrick

	// Play only valid card. This includes leading 2c.
	if len(hand) == 1 {
		return hand[0]
	}

	// If we have the lead
	haveLead := len(trick) == 0
	if haveLead {
		if !qsWasPlayed(gs) {
			//   If we have qs, ks, or as
			if hand.ContainsAny(cards.Cqs, cards.Cks, cards.Cas) {
				// Lead lowest card in non-spade suit,
				nonSpades := hand.FilterBySuit(cards.Hearts, cards.Diamonds, cards.Clubs)
				if len(nonSpades) > 0 {
					return nonSpades.Lowest()
				}
				// or lead lowest nonQueen spade, (remaining cards are all spades)
				nonQueenSpades := hand.Filter(func(c cards.Card) bool { return c != cards.Cqs })
				if len(nonQueenSpades) > 0 {
					return nonQueenSpades.Lowest()
				}
				// or lead Qs
				return cards.Cqs
			}
			// If we have a spade (but not qka), lead highest spade
			if hand.ContainsSuit(cards.Spades) {
				return hand.FilterBySuit(cards.Spades).Highest()
			}
		}
		// Lead lowest card
		return hand.Lowest()
	} else {
		// We're following
		leadSuit := trick[0].Suit
		leadingCard := trick.LeadingCardOfTrick()

		// If spades is led and qs is still available and we have spades
		if leadSuit == cards.Spades && !qsWasPlayed(gs) && hand.ContainsSuit(cards.Spades) {
			// if we have qs ...
			if hand.ContainsCard(cards.Cqs) {
				// if as or ks is already played, play qs
				if trick.ContainsAny(cards.Cks, cards.Cas) {
					return cards.Cqs
				}
				// play high spade not queen
				nonQueenSpades := hand.Filter(func(c cards.Card) bool { return c != cards.Cqs })
				if len(nonQueenSpades) > 0 {
					return nonQueenSpades.Highest()
				}
				// else play queen
				return cards.Cqs
			}
			// so we don't have qs,
			// if we're the last card in the trick, play high spade
			if len(trick) == 3 {
				return hand.Highest()
			}
			// else play high spade under qs
			spadesUnderQueen := hand.FilterLE(cards.Jack)
			if len(spadesUnderQueen) > 0 {
				return spadesUnderQueen.Highest()
			}
			// else play high spade
			return hand.Highest()
		}
		// If we have the lead suit, (then that's all we can play)
		if hand.ContainsSuit(leadSuit) {
			// Best card if we don't want to take the trick.
			best := hand.HighestUnderValueOrLowest(leadingCard.Value)
			// If this is the last card
			if len(trick) == 3 {
				// If there are just 0 or 1 hearts, play highest card.
				if trick.CountSuit(cards.Hearts) <= 1 && !trick.ContainsCard(cards.Cqs) {
					return hand.Highest()
				}
				// If we have to take it anyway, go high.
				if best.Value > leadingCard.Value {
					return hand.Highest()
				}
				return best
			}
			// If hearts are led, try not to take it.
			if leadSuit == cards.Hearts {
				return best
			}
			// If this is the second-to-last card and the queen might be dumped last and we'd have to
			// take it anyway, might as well go high.
			if len(trick) == 2 && leadSuit != cards.Spades &&
				!qsWasPlayed(gs) && !trick.ContainsCard(cards.Cqs) &&
				best.Value > leadingCard.Value {
				return hand.Highest()
			}
			// if qs is available
			if !qsWasPlayed(gs) {
				// If first trick of suit, play high
				if numTricksOfSuit(gs, leadSuit) == 0 {
					return hand.Highest()
				} else {
					// else play highest under winning card or lowest card
					return best
				}
			} else { // we have the led suit and qs is not available
				// If first two tricks of suit, play high
				if numTricksOfSuit(gs, leadSuit) <= 1 {
					return hand.Highest()
				} else {
					// else play highest under winning card or lowest card
					return best
				}
			}
		}
		// We don't have lead suit
		// If qs hasn't been played, play a high spade
		if !qsWasPlayed(gs) {
			if hand.ContainsCard(cards.Cqs) {
				return cards.Cqs
			}
			if hand.ContainsCard(cards.Cas) {
				return cards.Cas
			}
			if hand.ContainsCard(cards.Cks) {
				return cards.Cks
			}
		}
		// If we have hearts over 7, play highest
		highHearts := hand.FilterBySuit(cards.Hearts).FilterGE(cards.Eight)
		if len(highHearts) > 0 {
			return highHearts.Highest()
		}
		// Play highest card of suit with highest low card
		// but don't dump a spade if we have the queen.
		if fullHand.ContainsCard(cards.Cqs) {
			hand = hand.FilterBySuit(cards.Clubs, cards.Hearts, cards.Diamonds)
		}
		handBySuit := maps.Values(hand.SplitBySuit())
		suitWithHighestLowCard := cards.GetExtremeHand(handBySuit, func(c1, c2 cards.Cards) bool {
			return c1.Lowest().Value > c2.Lowest().Value
		})
		return suitWithHighestLowCard.Highest()
	}
}
