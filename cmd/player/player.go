package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"

	"github.com/mpsalisbury/cards/pkg/client"
	hearts "github.com/mpsalisbury/cards/pkg/game/hearts/player"
)

var (
	gameId     = flag.String("game", "", "Game to join")
	joinAny    = flag.Bool("joinany", false, "Join any available game")
	verbose    = flag.Bool("verbose", false, "Print extra information during the session")
	name       = flag.String("name", "", "Your player name")
	hints      = flag.Bool("hints", false, "Provide gameplay hints")
	playerType = "basic"
	serverType = "lan"
)

func init() {
	hearts.AddPlayerFlag(&playerType, "player")
	client.AddServerFlag(&serverType, "server")
}

func main() {
	flag.Parse()
	err := runPlayer()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}
func runPlayer() error {
	stype, err := client.ServerTypeFromFlag(serverType)
	if err != nil {
		return err
	}
	conn, err := client.Connect(stype, *verbose)
	if err != nil {
		return err
	}
	if *gameId == "" {
		if *joinAny {
			*gameId, err = chooseGame(conn)
			if err != nil {
				return err
			}
		} else {
			return showGames(conn)
		}
	}
	ctx := context.Background()
	player, err := hearts.NewPlayerFromFlag(playerType, *hints)
	if err != nil {
		return err
	}
	session, err := conn.Register(ctx, *name, callbacks{player})
	if err != nil {
		return err
	}
	wg := new(sync.WaitGroup)
	err = session.JoinGame(ctx, wg, *gameId)
	if err != nil {
		return err
	}
	fmt.Printf("Joined game %s\n", *gameId)
	wg.Wait()
	fmt.Printf("Game %s finished\n", *gameId)
	return nil
}

func showGames(conn client.Connection) error {
	ctx := context.Background()
	games, err := conn.ListGames(ctx, client.Preparing)
	if err != nil {
		return fmt.Errorf("couldn't list games: %v", err)
	}
	fmt.Printf("Available games\n")
	for _, g := range games {
		fmt.Printf("%s - %s %s\n", g.Id, g.Phase, g.Names)
	}
	return nil
}
func chooseGame(conn client.Connection) (string, error) {
	ctx := context.Background()
	games, err := conn.ListGames(ctx, client.Preparing)
	if err != nil {
		return "", fmt.Errorf("couldn't list games: %v", err)
	}
	// Choose first game that's available.
	if len(games) > 0 {
		return games[0].Id, nil
	}
	// Else create a new game.
	return conn.CreateGame(ctx)
}

// client.GameCallbacks
type callbacks struct {
	client.GameCallbacks
}

func (c callbacks) HandlePlayerJoined(s client.Session, name string, gameId string) error {
	fmt.Printf("Player %s joined game %s\n", name, gameId)
	return c.GameCallbacks.HandlePlayerJoined(s, name, gameId)
}
func (c callbacks) HandlePlayerLeft(s client.Session, name string, gameId string) error {
	fmt.Printf("Player %s left game %s\n", name, gameId)
	return c.GameCallbacks.HandlePlayerLeft(s, name, gameId)
}

func (c callbacks) HandleGameStarted(s client.Session, gameId string) error {
	fmt.Printf("Game %s starting\n", gameId)
	return c.GameCallbacks.HandleGameStarted(s, gameId)
}
func (c callbacks) HandleGameFinished(s client.Session, gameId string) {
	fmt.Printf("Game %s over\n", gameId)
	showGameState(s, gameId)
}
func (c callbacks) HandleGameAborted(s client.Session, gameId string) {
	fmt.Printf("Game %s aborted\n", gameId)
	showGameState(s, gameId)
}
func (c callbacks) HandleConnectionError(s client.Session, err error) {
	fmt.Printf("%v\n", err)
}
func showGameState(s client.Session, gameId string) {
	gameState, err := s.GetGameState(context.Background(), gameId)
	if err != nil {
		log.Fatalf("Couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
}
