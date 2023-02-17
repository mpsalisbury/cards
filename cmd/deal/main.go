package main

import (
	"fmt"

	"github.com/mpsalisbury/cards/pkg/cards"
	"golang.org/x/exp/slices"
)

//func (g *Game) PlayTrick() Trick {

//}

type Player struct {
	p  cards.Player
	cs cards.Cards
}

type Game struct {
	players []Player
}

func NewGame(playerNames ...string) *Game {
	hs := cards.Deal(len(playerNames))
	ps := []Player{}
	for i, h := range hs {
		name := playerNames[i]
		ps = append(ps, Player{cards.Player{Id: i, Name: name}, h})
	}
	return &Game{ps}
}

func main() {
	g := NewGame("Joe", "Mary", "Bob", "Jill")
	trick := cards.NewTrick()
	for _, p := range g.players {
		fmt.Printf("%5s: %s\n", p.p.Name, p.cs.HandString())
		for {
			fmt.Printf("Card to play: ")
			var cardToPlay string
			fmt.Scanln(&cardToPlay)
			card, err := cards.ParseCard(cardToPlay)
			if err != nil {
				fmt.Println("Invalid card")
				continue
			}
			if !slices.Contains(p.cs, card) {
				fmt.Printf("Hand does not contain %s\n", card)
				continue
			}
			trick.Add(p.p, card)
			break
		}
	}
	fmt.Printf("Trick: %s\n", trick)
}
