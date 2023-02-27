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
	verbose    = flag.Bool("verbose", false, "Print extra information during the session")
	playerType = "basic"
	serverType = "inprocess"
)

func init() {
	hearts.AddPlayerFlag(&playerType, "player")
	client.AddServerFlag(&serverType, "server")
}

func main() {
	flag.Parse()
	err := runPlayers()
	if err != nil {
		log.Print(err)
	}
}
func runPlayers() error {
	stype, err := client.ServerTypeFromFlag(serverType)
	if err != nil {
		return err
	}
	conn, err := client.Connect(stype, *verbose)
	if err != nil {
		return fmt.Errorf("couldn't connect to server: %w", err)
	}
	gameId, err := conn.CreateGame(context.Background())
	if err != nil {
		return err
	}
	wg := new(sync.WaitGroup)
	for i := 0; i < 4; i++ {
		err = startAutoPlayer(conn, wg, gameId)
		if err != nil {
			return err
		}
	}
	wg.Wait() // join with player threads.
	gameState, err := conn.GetGameState(context.Background(), gameId)
	if err != nil {
		return err
	}
	fmt.Print(gameState)
	return nil
}

func startAutoPlayer(conn client.Connection, wg *sync.WaitGroup, gameId string) error {
	ctx := context.Background()
	player, err := hearts.NewPlayerFromFlag(playerType)
	if err != nil {
		return fmt.Errorf("couldn't create player: %w", err)
	}
	session, err := conn.Register(ctx, "", player)
	if err != nil {
		return fmt.Errorf("couldn't register with server: %w", err)
	}
	err = session.JoinGame(ctx, wg, gameId)
	if err != nil {
		return fmt.Errorf("couldn't join game: %w", err)
	}
	return nil
}
