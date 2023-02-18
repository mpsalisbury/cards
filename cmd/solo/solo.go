package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/mpsalisbury/cards/pkg/cards"
	"github.com/mpsalisbury/cards/pkg/client"
)

var (
	verbose = flag.Bool("verbose", false, "Print extra information during the session")
)

func init() {
	rand.Seed(time.Now().UnixNano())
}
func main() {
	flag.Parse()
	err := RunPlayer()
	if err != nil {
		log.Fatal(err)
	}
}
func RunPlayer() error {
	conn, err := client.Connect(client.LocalServer, *verbose)
	if err != nil {
		return fmt.Errorf("couldn't connect to server: %v", err)
	}
	gameId := ""
	wg := new(sync.WaitGroup)
	for i := 0; i < 3; i++ {
		gameId, err = startAutoPlayer(conn, wg, gameId)
		if err != nil {
			return err
		}
	}
	err = startCmdlinePlayer(conn, wg, gameId)
	if err != nil {
		return err
	}
	wg.Wait()
	gameState, err := conn.GetGameState(context.Background(), gameId)
	if err != nil {
		return err
	}
	fmt.Print(gameState)
	return nil
}

func startAutoPlayer(conn client.Connection, wg *sync.WaitGroup, gameId string) (string, error) {
	return startPlayer(conn, wg, gameId, autoplayCallbacks{})
}
func startCmdlinePlayer(conn client.Connection, wg *sync.WaitGroup, gameId string) error {
	_, err := startPlayer(conn, wg, gameId, cmdlineCallbacks{})
	return err
}
func startPlayer(conn client.Connection, wg *sync.WaitGroup, gameId string, callbacks client.GameCallbacks) (string, error) {
	ctx := context.Background()
	name := fmt.Sprintf("Henry%04d", rand.Intn(10000))
	session, err := conn.Register(ctx, name, callbacks)
	if err != nil {
		return "", fmt.Errorf("couldn't register with server: %v", err)
	}
	gameId, err = session.JoinGameAsPlayer(ctx, wg, gameId)
	if err != nil {
		return "", fmt.Errorf("couldn't join game: %v", err)
	}
	wg.Add(1)
	return gameId, nil
}

// client.GameCallbacks
type autoplayCallbacks struct {
	client.UnimplementedGameCallbacks
}

func (c autoplayCallbacks) HandleYourTurn(s client.Session) error {
	ctx := context.Background()
	gameState, err := s.GetGameState(ctx)
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	for _, card := range gameState.Players[0].Cards {
		err = s.PlayCard(ctx, card)
		if err == nil {
			break
		}
	}
	return nil
}

// client.GameCallbacks
type cmdlineCallbacks struct {
	client.UnimplementedGameCallbacks
}

func (c cmdlineCallbacks) HandleTrickCompleted(s client.Session, trick cards.Cards, trickWinnerId, trickWinnerName string) error {
	fmt.Printf("Trick: %s won by %s\n", trick, trickWinnerName)
	return nil
}

func (c cmdlineCallbacks) HandleYourTurn(s client.Session) error {
	ctx := context.Background()
	gameState, err := s.GetGameState(ctx)
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	for {
		card := c.chooseCard(gameState)
		if err := s.PlayCard(ctx, card); err == nil {
			return nil
		}
		fmt.Printf("Can't play card %s. Try again\n", card)
	}
}

func (c cmdlineCallbacks) chooseCard(gameState client.GameState) cards.Card {
	for {
		fmt.Println(showGame(gameState))
		fmt.Print("Enter card to play: ")
		var cs string
		fmt.Scanln(&cs)
		card, err := cards.ParseCard(cs)
		if err == nil {
			return card
		}
		fmt.Printf("Invalid card %s, try again\n", cs)
	}
}

func showGame(gs client.GameState) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Hand: %s\n", gs.Players[0].Cards.HandString()))
	sb.WriteString(fmt.Sprintf("Trick: %s\n", gs.CurrentTrick))
	return sb.String()
}
