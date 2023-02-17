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
	gameId  = flag.String("game", "", "Game to join")
	joinAny = flag.Bool("joinany", false, "Join any available game")
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
	name := fmt.Sprintf("Henry%04d", rand.Intn(10000))
	err = conn.Register(ctx, name, callbacks{client: conn})
	if err != nil {
		return fmt.Errorf("couldn't register with server: %v", err)
	}
	wg := new(sync.WaitGroup)
	wg.Add(1)
	joinedGameId, err := conn.JoinGameAsPlayer(ctx, wg, *gameId)
	if err != nil {
		return fmt.Errorf("couldn't join game: %v", err)
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

func (c callbacks) HandleYourTurn() error {
	ctx := context.Background()
	log.Println("Performing turn")
	gameState, err := c.client.GetGameState(ctx)
	if err != nil {
		return fmt.Errorf("couldn't get game state: %v", err)
	}
	for _, card := range gameState.Players[0].Cards {
		log.Printf("Trying card %s", card)
		err = c.client.PlayCard(ctx, card)
		if err == nil {
			log.Printf("  success")
			// Successful play, we're done.
			break
		}
	}
	return nil
}
