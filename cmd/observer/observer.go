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
	gameState, err := conn.GetGameState(ctx, *gameId)
	if err != nil {
		log.Fatalf("Couldn't get gamestate: %v", err)
	}
	if gameState.Phase == client.Completed || gameState.Phase == client.Aborted {
		// game is complete, just dump state.
		fmt.Printf("%v\n", gameState)
		return
	}

	name := fmt.Sprintf("Observer%04d", rand.Intn(10000))
	session, err := conn.Register(ctx, name, callbacks{})
	if err != nil {
		log.Fatalf("Couldn't register with server: %v", err)
	}
	wg := new(sync.WaitGroup)
	wg.Add(1)
	joinedGameId, err := session.JoinGameAsObserver(ctx, wg, *gameId)
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
}

func (callbacks) HandlePlayerJoined(s client.Session, name string, gameId string) error {
	fmt.Printf("Player %s joined game %s\n", name, gameId)
	return nil
}
func (callbacks) HandlePlayerLeft(s client.Session, name string, gameId string) error {
	fmt.Printf("Player %s left game %s\n", name, gameId)
	return nil
}
func (c callbacks) HandleGameStarted(s client.Session) error {
	gameState, err := s.GetGameState(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
	return nil
}
func (c callbacks) HandleGameFinished(s client.Session) error {
	fmt.Printf("Game over\n")
	c.showGameState(s)
	return nil
}
func (c callbacks) HandleGameAborted(s client.Session) error {
	fmt.Printf("Game aborted\n")
	c.showGameState(s)
	return nil
}
func (c callbacks) HandleConnectionError(s client.Session, err error) {
	fmt.Printf("Connection error: %v\n", err)
}
func (c callbacks) showGameState(s client.Session) {
	gameState, err := s.GetGameState(context.Background())
	if err != nil {
		log.Fatalf("Couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
}

func (c callbacks) HandleTrickCompleted(s client.Session) error {
	ctx := context.Background()
	gameState, err := s.GetGameState(ctx)
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
	return nil
}
