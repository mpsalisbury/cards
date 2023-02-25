package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"

	"github.com/mpsalisbury/cards/pkg/cards"
	"github.com/mpsalisbury/cards/pkg/client"
)

var (
	//	logger     = log.New(os.Stdout, "", 0)
	gameId  = flag.String("game", "", "Game to observe")
	all     = flag.Bool("all", false, "Observe all games")
	verbose = flag.Bool("verbose", false, "Print extra information during the session")
	name    = flag.String("name", "", "Your observer name")
)

func main() {
	flag.Parse()

	conn, err := client.Connect(client.LocalServer, *verbose)
	if err != nil {
		log.Fatalf("Couldn't connect to server: %v", err)
	}
	ctx := context.Background()
	wg := new(sync.WaitGroup)
	if *all {
		wg.Add(1)
		err = conn.ObserveRegistry(ctx, wg, registryCallbacks{})
		if err != nil {
			log.Fatalf("Couldn't observe registry: %v", err)
		}
		fmt.Printf("Observing registry\n")
	} else if *gameId == "" {
		showGames(conn)
		return
	} else {
		gameState, err := conn.GetGameState(ctx, *gameId)
		if err != nil {
			log.Fatalf("Couldn't get gamestate: %v", err)
		}
		if gameState.Phase == client.Completed || gameState.Phase == client.Aborted {
			// game is complete, just dump state.
			fmt.Printf("%v\n", gameState)
			return
		}
		session, err := conn.Register(ctx, *name, gameCallbacks{})
		if err != nil {
			log.Fatalf("Couldn't register with server: %v", err)
		}
		wg.Add(1)
		joinedGameId, err := session.JoinGameAsObserver(ctx, wg, *gameId)
		if err != nil {
			log.Fatalf("Couldn't observe game %s: %v", *gameId, err)
		}
		fmt.Printf("Observing game %s\n", joinedGameId)
	}
	wg.Wait()
}

func showGames(conn client.Connection) {
	ctx := context.Background()
	games, err := conn.ListGames(ctx)
	if err != nil {
		log.Fatalf("Couldn't list games: %v", err)
	}
	if len(games) == 0 {
		fmt.Printf("No available games to observe\n")
		return
	}
	fmt.Printf("Available games\n")
	for _, g := range games {
		fmt.Printf("%s - %s %s\n", g.Id, g.Phase, g.Names)
	}
}

// client.RegistryCallbacks
type registryCallbacks struct {
	client.UnimplementedRegistryCallbacks
}

func (registryCallbacks) HandleGameCreated(c client.Connection, gameId string) error {
	fmt.Printf("Game %s created\n", gameId)
	return nil
}
func (registryCallbacks) HandleGameDeleted(c client.Connection, gameId string) error {
	fmt.Printf("Game %s deleted\n", gameId)
	return nil
}
func (registryCallbacks) HandleFullGamesList(c client.Connection, gameIds []string) error {
	if len(gameIds) > 0 {
		fmt.Printf("Existing games:\n")
		for _, gid := range gameIds {
			fmt.Printf("  %s\n", gid)
		}
	}
	return nil
}
func (registryCallbacks) HandleConnectionError(c client.Connection, err error) {
	fmt.Printf("Connection error: %v\n", err)
}

// client.GameCallbacks
type gameCallbacks struct {
	client.UnimplementedGameCallbacks
}

func (gameCallbacks) HandlePlayerJoined(s client.Session, name string, gameId string) error {
	fmt.Printf("Player %s joined game %s\n", name, gameId)
	return nil
}
func (gameCallbacks) HandlePlayerLeft(s client.Session, name string, gameId string) error {
	fmt.Printf("Player %s left game %s\n", name, gameId)
	return nil
}
func (c gameCallbacks) HandleGameStarted(s client.Session) error {
	gameState, err := s.GetGameState(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
	return nil
}
func (c gameCallbacks) HandleGameFinished(s client.Session) {
	fmt.Printf("Game over\n")
	c.showGameState(s)
}
func (c gameCallbacks) HandleGameAborted(s client.Session) {
	fmt.Printf("Game aborted\n")
	c.showGameState(s)
}
func (c gameCallbacks) HandleConnectionError(s client.Session, err error) {
	fmt.Printf("Connection error: %v\n", err)
}
func (c gameCallbacks) showGameState(s client.Session) {
	gameState, err := s.GetGameState(context.Background())
	if err != nil {
		log.Fatalf("Couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
}

func (c gameCallbacks) HandleTrickCompleted(s client.Session, trick cards.Cards, trickWinnerId, trickWinnerName string) error {
	ctx := context.Background()
	gameState, err := s.GetGameState(ctx)
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
	return nil
}
