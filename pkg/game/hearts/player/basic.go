package player

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
func qsNotYetPlayed(gs client.GameState) bool {
	return !anyPlayedCard(gs, func(c cards.Card) bool { return c == cards.Cqs })
}

func (c basicPlayer) HandleYourTurn(s client.Session, gameId string) error {
	ctx := context.Background()
	gameState, err := s.GetGameState(ctx, gameId)
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	card := chooseCardToPlay(gameState)
	err = s.PlayCard(ctx, gameId, card)
	if err != nil {
		log.Fatalf("BasicPlayer chose invalid card %s\nGamestate: %v", card, gameState)
	}
	return nil
}

// Publicly expose basic strategy.
func ChooseBasicStrategyCard(gs client.GameState) cards.Card {
	return chooseCardToPlay(gs)
}

func chooseCardToPlay(gs client.GameState) cards.Card {
	legalPlays := gs.LegalPlays
	trick := gs.CurrentTrick

	// Play only valid card. This includes leading 2c.
	if len(legalPlays) == 1 {
		return legalPlays[0]
	}

	haveLead := len(trick) == 0
	if haveLead {
		return chooseLeadCard(gs)
	}
	// else, we're following
	leadSuit := trick[0].Suit

	// If we have the lead suit,
	if legalPlays.ContainsSuit(leadSuit) {
		// If spades is led and qs is still available
		if leadSuit == cards.Spades && qsNotYetPlayed(gs) {
			return followSpadesWhenQueenOutstanding(gs)
		}
		return followSuit(gs)
	}
	// We can't follow lead suit.
	return chooseDumpCard(gs)
}

func chooseLeadCard(gs client.GameState) cards.Card {
	legalPlays := gs.LegalPlays
	if qsNotYetPlayed(gs) {
		//   If we have qs, ks, or as, ...
		if legalPlays.ContainsAny(cards.Cqs, cards.Cks, cards.Cas) {
			// ... lead lowest card in non-spade suit,
			nonSpades := legalPlays.FilterBySuit(cards.Hearts, cards.Diamonds, cards.Clubs)
			if len(nonSpades) > 0 {
				return nonSpades.Lowest()
			}
			// or lead lowest nonQueen spade, (remaining cards are all spades)
			nonQueenSpades := legalPlays.Filter(func(c cards.Card) bool { return c != cards.Cqs })
			if len(nonQueenSpades) > 0 {
				return nonQueenSpades.Lowest()
			}
			// or lead Qs
			return cards.Cqs
		}
		// If we have a spade (but not qka), lead highest spade
		if legalPlays.ContainsSuit(cards.Spades) {
			return legalPlays.FilterBySuit(cards.Spades).Highest()
		}
	}
	// Lead lowest card
	return legalPlays.Lowest()
}

func followSpadesWhenQueenOutstanding(gs client.GameState) cards.Card {
	legalPlays := gs.LegalPlays
	trick := gs.CurrentTrick

	// if we have qs ...
	if legalPlays.ContainsCard(cards.Cqs) {
		// if as or ks is already in the trick, play qs
		if trick.ContainsAny(cards.Cks, cards.Cas) {
			return cards.Cqs
		}
		// play high spade not queen
		nonQueenSpades := legalPlays.Filter(func(c cards.Card) bool { return c != cards.Cqs })
		if len(nonQueenSpades) > 0 {
			return nonQueenSpades.Highest()
		}
		// else play queen
		return cards.Cqs
	}
	// so we don't have qs,
	// if we're the last card in the trick, play high spade
	if len(trick) == 3 {
		return legalPlays.Highest()
	}
	// else play high spade under qs
	spadesUnderQueen := legalPlays.FilterLE(cards.Jack)
	if len(spadesUnderQueen) > 0 {
		return spadesUnderQueen.Highest()
	}
	// else play high spade
	return legalPlays.Highest()
}

func followSuit(gs client.GameState) cards.Card {
	legalPlays := gs.LegalPlays
	trick := gs.CurrentTrick
	leadSuit := trick[0].Suit
	leadingCard := trick.LeadingCardOfTrick()

	// Consider the best card while trying not to take the trick.
	best := legalPlays.HighestUnderValueOrLowest(leadingCard.Value)
	// If this is the last card in the trick ...
	if len(trick) == 3 {
		// If there are just 0 or 1 hearts in the trick, just take it.
		if trick.CountSuit(cards.Hearts) <= 1 && !trick.ContainsCard(cards.Cqs) {
			return legalPlays.Highest()
		}
		// If we have to take it anyway, go high.
		if best.Value > leadingCard.Value {
			return legalPlays.Highest()
		}
		// Otherwise, dump our highest card without taking the trick.
		return best
	}
	// If hearts are led, try not to take it.
	if leadSuit == cards.Hearts {
		return best
	}
	// If this is the second-to-last card and the queen might be dumped last and we'd have to
	// take it anyway, might as well go high.
	if len(trick) == 2 && leadSuit != cards.Spades &&
		qsNotYetPlayed(gs) && !trick.ContainsCard(cards.Cqs) &&
		best.Value > leadingCard.Value {
		return legalPlays.Highest()
	}
	// If qs is available ...
	if qsNotYetPlayed(gs) {
		// ... and if this is the first trick of the suit, play high
		if numTricksOfSuit(gs, leadSuit) == 0 {
			return legalPlays.Highest()
		}
		// else try not to take trick.
		return best
	}
	// qs is not available
	// If this is one of the first two tricks of suit, play high
	if numTricksOfSuit(gs, leadSuit) <= 1 {
		return legalPlays.Highest()
	}
	// else try not to take the trick.
	return best
}

func chooseDumpCard(gs client.GameState) cards.Card {
	legalPlays := gs.LegalPlays

	// We don't have lead suit
	// If qs hasn't been played, play a high spade
	if qsNotYetPlayed(gs) {
		if legalPlays.ContainsCard(cards.Cqs) {
			return cards.Cqs
		}
		// if we don't have enough low spades, dump a high one.
		spades := legalPlays.FilterBySuit(cards.Spades)
		if len(spades.FilterLE(cards.Jack)) <= 3 &&
			len(spades.FilterGE(cards.King)) > 0 {
			return spades.Highest()
		}
	}
	// If we have hearts over 7, play highest
	highHearts := legalPlays.FilterBySuit(cards.Hearts).FilterGE(cards.Eight)
	if len(highHearts) > 0 {
		return highHearts.Highest()
	}
	// Don't dump a spade if we have the queen (unless we have to).
	hand := gs.Players[0].Cards // We might have the queen even if we can't play it.
	if hand.ContainsCard(cards.Cqs) {
		nonSpades := legalPlays.FilterBySuit(cards.Clubs, cards.Hearts, cards.Diamonds)
		if len(nonSpades) > 0 {
			legalPlays = nonSpades
		}
	}
	// Play highest card of suit with highest low card (bad suit for us).
	playsBySuit := maps.Values(legalPlays.SplitBySuit())
	suitWithHighestLowCard := cards.GetExtremeCards(playsBySuit, func(c1, c2 cards.Cards) bool {
		return c1.Lowest().Value > c2.Lowest().Value
	})
	return suitWithHighestLowCard.Highest()
}
