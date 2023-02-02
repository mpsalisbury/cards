package cards

import (
	"testing"
)

func absDiff(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

func TestDeal(t *testing.T) {
	fullDeck := MakeDeck()
	for numHands := 2; numHands <= 6; numHands++ {
		hands := Deal(numHands)
		if len(hands) != numHands {
			t.Errorf("Deal(%d)=%d hands, want %d", numHands, len(hands), numHands)
		}
		// Make sure all hands have number of cards within 1 of each other.
		cardsPerHand := len(hands[0])
		for _, h := range hands {
			numCards := len(h)
			if absDiff(numCards, cardsPerHand) > 1 {
				t.Errorf("Deal(%d): Expected each hand to have similar count. Found numCards = %d vs %d",
					numHands, numCards, cardsPerHand)
			}
		}
		// Make sure all cards were dealt.
		allCards := Combine(hands...)
		if allCards.String() != fullDeck.String() {
			t.Errorf("Deal(%d)='%s', expected full deck", numHands, allCards)
		}
	}
}
