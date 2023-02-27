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
)

func init() {
	hearts.AddPlayerFlag(&playerType, "type")
}

func main() {
	flag.Parse()
	err := runPlayers()
	if err != nil {
		log.Fatal(err)
	}
}
func runPlayers() error {
	conn, err := client.Connect(client.InProcessServer, *verbose)
	if err != nil {
		return fmt.Errorf("couldn't connect to server: %v", err)
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
	wg.Wait() // wait if you want to join with other player threads.
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
		return fmt.Errorf("couldn't create player: %v", err)
	}
	session, err := conn.Register(ctx, "", player)
	if err != nil {
		return fmt.Errorf("couldn't register with server: %v", err)
	}
	wg.Add(1)
	err = session.JoinGame(ctx, wg, gameId)
	if err != nil {
		return fmt.Errorf("couldn't join game: %v", err)
	}
	return nil
}
