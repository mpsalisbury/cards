package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

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
	for i := 0; i < 4; i++ {
		wg.Add(1)
		gameId, err = startAutoPlayer(conn, wg, gameId)
		if err != nil {
			return err
		}
	}
	wg.Wait() // wait if you want to join with other player threads.
	gameState, err := conn.GetGameState(context.Background(), gameId)
	if err != nil {
		return err
	}
	fmt.Print(gameState)
	return nil
}

func startAutoPlayer(conn client.Connection, wg *sync.WaitGroup, gameId string) (string, error) {
	ctx := context.Background()
	name := fmt.Sprintf("Henry%04d", rand.Intn(10000))
	session, err := conn.Register(ctx, name, autoplayCallbacks{})
	if err != nil {
		return "", fmt.Errorf("couldn't register with server: %v", err)
	}
	gameId, err = session.JoinGameAsPlayer(ctx, wg, gameId)
	if err != nil {
		return "", fmt.Errorf("couldn't join game: %v", err)
	}
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
