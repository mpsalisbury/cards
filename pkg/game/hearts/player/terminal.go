package player

import (
	"context"
	"fmt"
	"strings"

	"github.com/mpsalisbury/cards/pkg/cards"
	"github.com/mpsalisbury/cards/pkg/client"
)

// TerminalPlayer has user enter plays via terminal.

func NewTerminalPlayer() client.GameCallbacks {
	return &terminalCallbacks{}
}

type terminalCallbacks struct {
	client.UnimplementedGameCallbacks
}

func (c terminalCallbacks) HandleGameStarted(s client.Session, gameId string) error {
	ctx := context.Background()
	gameState, err := s.GetGameState(ctx, gameId)
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	myName, otherNames := c.playerNames(gameState, s.GetSessionId())
	fmt.Printf("Welcome %s. Other players are %s.\n", myName, strings.Join(otherNames, ", "))
	return nil
}

func (terminalCallbacks) playerNames(gameState client.GameState, pid string) (playerName string, otherNames []string) {
	for _, ps := range gameState.Players {
		if ps.Id == pid {
			playerName = ps.Name
		} else {
			otherNames = append(otherNames, ps.Name)
		}
	}
	return
}

func (c terminalCallbacks) HandleTrickCompleted(s client.Session, gameId string, trick cards.Cards, trickWinnerId, trickWinnerName string) error {
	fmt.Printf("Trick: %s won by %s\n\n", trick, trickWinnerName)
	return nil
}

func (c terminalCallbacks) HandleYourTurn(s client.Session, gameId string) error {
	ctx := context.Background()
	gameState, err := s.GetGameState(ctx, gameId)
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	for {
		card := c.chooseCard(gameState)
		if err := s.PlayCard(ctx, gameId, card); err == nil {
			return nil
		}
		fmt.Printf("Can't play card %s. Try again\n", card)
	}
}

func (c terminalCallbacks) chooseCard(gs client.GameState) cards.Card {
	for {
		recommended := ChooseBasicStrategyCard(gs)
		fmt.Println(showGame(gs))
		fmt.Printf("Enter card to play [%s]: ", recommended)
		var cs string
		fmt.Scanln(&cs)
		if cs == "" {
			return recommended
		}
		card, err := cards.ParseCard(cs)
		if err == nil {
			return card
		}
		fmt.Printf("Invalid card %s, try again\n", cs)
	}
}

func showGame(gs client.GameState) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Your hand: %s\n", gs.Players[0].Cards.HandString()))
	sb.WriteString(fmt.Sprintf("Trick so far: %s", gs.CurrentTrick))
	return sb.String()
}
