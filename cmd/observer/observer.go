package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/mpsalisbury/cards/internal/game/client"
)

var (
	//	logger     = log.New(os.Stdout, "", 0)
	gameId  = flag.String("game", "", "Game to observe")
	verbose = flag.Bool("verbose", false, "Print extra information during the session")
)

func init() {
	rand.Seed(time.Now().UnixNano())
}
func main() {
	flag.Parse()

	conn, err := client.Connect(client.LocalServer, *verbose)
	if err != nil {
		log.Fatalf("Couldn't connect to server: %v", err)
	}
	if *gameId == "" {
		showGames(conn)
		return
	}
	ctx := context.Background()
	gameState, err := conn.GetGameStateForGameId(ctx, *gameId)
	if gameState.Phase == client.Completed || gameState.Phase == client.Aborted {
		// game is complete, just dump state.
		fmt.Printf("%v\n", gameState)
		return
	}

	name := fmt.Sprintf("Observer%04d", rand.Intn(10000))
	err = conn.Register(ctx, name, callbacks{client: conn})
	if err != nil {
		log.Fatalf("Couldn't register with server: %v", err)
	}
	wg := new(sync.WaitGroup)
	wg.Add(1)
	joinedGameId, err := conn.JoinGameAsObserver(ctx, wg, *gameId)
	if err != nil {
		log.Fatalf("Couldn't observe game %s: %v", *gameId, err)
	}
	fmt.Printf("Observing game %s\n", joinedGameId)
	wg.Wait()
}

func showGames(conn client.Connection) {
	ctx := context.Background()
	games, err := conn.ListGames(ctx)
	if err != nil {
		log.Fatalf("Couldn't list games: %v", err)
	}
	fmt.Printf("Available games\n")
	for _, g := range games {
		fmt.Printf("%s - %s %s\n", g.Id, g.Phase, g.Names)
	}
}

// client.GameCallbacks
type callbacks struct {
	client.UnimplementedGameCallbacks
	client client.Connection
}

func (callbacks) HandlePlayerJoined(name string, gameId string) error {
	fmt.Printf("Player %s joined game %s\n", name, gameId)
	return nil
}
func (callbacks) HandlePlayerLeft(name string, gameId string) error {
	fmt.Printf("Player %s left game %s\n", name, gameId)
	return nil
}
func (c callbacks) HandleGameStarted() error {
	gameState, err := c.client.GetGameState(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
	return nil
}
func (c callbacks) HandleGameFinished() error {
	fmt.Printf("Game over\n")
	c.showGameState()
	return nil
}
func (c callbacks) HandleGameAborted() error {
	fmt.Printf("Game aborted\n")
	c.showGameState()
	return nil
}
func (c callbacks) HandleConnectionError(err error) {
	fmt.Printf("Connection error: %v\n", err)
}
func (c callbacks) showGameState() {
	gameState, err := c.client.GetGameState(context.Background())
	if err != nil {
		log.Fatalf("Couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
}

func (c callbacks) HandleTrickCompleted() error {
	ctx := context.Background()
	gameState, err := c.client.GetGameState(ctx)
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
	return nil
}
