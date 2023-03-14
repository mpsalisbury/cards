package player

import (
	"math/rand"
	"time"

	"github.com/mpsalisbury/cards/pkg/cards"
	"github.com/mpsalisbury/cards/pkg/client"
)

// Plays a random card that is legal.

func NewRandomStrategy() PlayerStrategy {
	return &randomStrategy{}
}

type randomStrategy struct{}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (s randomStrategy) ChooseCardToPlay(gs client.GameState) cards.Card {
	legalPlays := gs.LegalPlays
	return legalPlays[rand.Intn(len(legalPlays))]
}
