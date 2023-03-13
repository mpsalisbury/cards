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
	gameId     = flag.String("game", "", "Game to observe")
	all        = flag.Bool("all", false, "Observe all games")
	verbose    = flag.Bool("verbose", false, "Print extra information during the session")
	name       = flag.String("name", "", "Your observer name")
	serverType = "lan"
)

func init() {
	client.AddServerFlag(&serverType, "server")
}

func main() {
	flag.Parse()
	err := observe()
	if err != nil {
		log.Print(err)
	}
}
func observe() error {
	stype, err := client.ServerTypeFromFlag(serverType)
	if err != nil {
		return err
	}
	conn, err := client.Connect(stype, *verbose)
	if err != nil {
		return fmt.Errorf("Couldn't connect to server: %w", err)
	}
	ctx := context.Background()
	wg := new(sync.WaitGroup)
	if *all {
		rc := &registryCallbacks{wg: wg}
		_, err := conn.RegisterObserver(ctx, wg, *name, rc, gameCallbacks{})
		if err != nil {
			return fmt.Errorf("Couldn't observe registry: %w", err)
		}
		fmt.Printf("Observing all games\n")
	} else if *gameId == "" {
		showGames(conn)
		return nil
	} else {
		// Observe one game.
		gameState, err := conn.GetGameState(ctx, *gameId)
		if err != nil {
			return fmt.Errorf("Couldn't get gamestate: %w", err)
		}
		if gameState.Phase == client.Completed || gameState.Phase == client.Aborted {
			// game is complete, just dump state.
			fmt.Printf("%v\n", gameState)
			return nil
		}
		session, err := conn.Register(ctx, *name, gameCallbacks{})
		if err != nil {
			return fmt.Errorf("Couldn't register with server: %w", err)
		}
		err = session.ObserveGame(ctx, wg, *gameId)
		if err != nil {
			return fmt.Errorf("Couldn't observe game %s: %w", *gameId, err)
		}
		fmt.Printf("Observing game %s\n", *gameId)
	}
	wg.Wait()
	return nil
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
	session client.Session
	wg      *sync.WaitGroup // WaitGroup used by the main process to wait for subthreads.
}

func (rc *registryCallbacks) observeGame(gameId string) error {
	if rc.session == nil {
		fmt.Printf("Can't observe game %s - no session yet\n", gameId)
		return nil
	}
	err := rc.session.ObserveGame(context.Background(), rc.wg, gameId)
	if err != nil {
		log.Fatalf("Couldn't observe game %s: %v", gameId, err)
	}
	fmt.Printf("Observing new game %s\n", gameId)
	return nil
}

func (rc *registryCallbacks) InstallSession(session client.Session) {
	rc.session = session
}

func (rc *registryCallbacks) HandleGameCreated(c client.Connection, gameId string) error {
	return rc.observeGame(gameId)
}

func (registryCallbacks) HandleGameDeleted(c client.Connection, gameId string) error {
	fmt.Printf("Game %s deleted\n", gameId)
	return nil
}

func (rc *registryCallbacks) HandleFullGamesList(c client.Connection, gameIds []string) error {
	if len(gameIds) > 0 {
		for _, gid := range gameIds {
			if err := rc.observeGame(gid); err != nil {
				return err
			}
		}
	}
	return nil
}
func (registryCallbacks) HandleConnectionError(c client.Connection, err error) {
	fmt.Printf("%v\n", err)
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
func (c gameCallbacks) HandleGameStarted(s client.Session, gameId string) error {
	fmt.Printf("Game %s started\n", gameId)
	return nil
}
func (c gameCallbacks) HandleGameFinished(s client.Session, gameId string) {
	fmt.Printf("Game %s over\n", gameId)
	c.showGameState(s, gameId)
}
func (c gameCallbacks) HandleGameAborted(s client.Session, gameId string) {
	fmt.Printf("Game %s aborted\n", gameId)
	c.showGameState(s, gameId)
}
func (c gameCallbacks) HandleConnectionError(s client.Session, err error) {
	fmt.Printf("%v\n", err)
}
func (c gameCallbacks) showGameState(s client.Session, gameId string) {
	gameState, err := s.GetGameState(context.Background(), gameId)
	if err != nil {
		log.Fatalf("Couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
}

func (c gameCallbacks) HandleTrickCompleted(s client.Session, gameId string,
	trick cards.Cards, winningCard cards.Card, trickWinnerId, trickWinnerName string) error {
	fmt.Printf("%s trick: %v - winner %s\n", gameId, trick, winningCard)
	return nil
}
