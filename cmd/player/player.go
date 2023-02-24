package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"

	"github.com/mpsalisbury/cards/pkg/client"
	"github.com/mpsalisbury/cards/pkg/game/hearts"
)

var (
	gameId     = flag.String("game", "", "Game to join")
	joinAny    = flag.Bool("joinany", false, "Join any available game")
	verbose    = flag.Bool("verbose", false, "Print extra information during the session")
	name       = flag.String("name", "", "Your player name")
	playerType = "basic"
)

func init() {
	hearts.AddPlayerFlag(&playerType, "type")
}

func main() {
	flag.Parse()
	err := RunPlayer()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}
func RunPlayer() error {
	conn, err := client.Connect(client.LocalServer, *verbose)
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
	player, err := hearts.NewPlayerFromFlag(playerType)
	if err != nil {
		return err
	}
	session, err := conn.Register(ctx, *name, callbacks{player})
	if err != nil {
		return err
	}
	wg := new(sync.WaitGroup)
	wg.Add(1)
	joinedGameId, err := session.JoinGameAsPlayer(ctx, wg, *gameId)
	if err != nil {
		return err
	}
	fmt.Printf("Joined game %s\n", joinedGameId)
	wg.Wait()
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
	// Empty gameId will create a new game.
	return "", nil
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

func (c callbacks) HandleGameStarted(s client.Session) error {
	fmt.Printf("Game starting\n")
	return c.GameCallbacks.HandleGameStarted(s)
}
func (c callbacks) HandleGameFinished(s client.Session) {
	fmt.Printf("Game over\n")
	showGameState(s)
}
func (c callbacks) HandleGameAborted(s client.Session) {
	fmt.Printf("Game aborted\n")
	showGameState(s)
}
func (c callbacks) HandleConnectionError(s client.Session, err error) {
	fmt.Printf("%v\n", err)
}
func showGameState(s client.Session) {
	gameState, err := s.GetGameState(context.Background())
	if err != nil {
		log.Fatalf("Couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
}
