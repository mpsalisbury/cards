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
	verbose    = flag.Bool("verbose", false, "Print extra information during the session")
	playerType = "basic"
)

func init() {
	hearts.AddPlayerFlag(&playerType, "type")
}

func main() {
	flag.Parse()
	err := RunPlayers()
	if err != nil {
		log.Fatal(err)
	}
}
func RunPlayers() error {
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
	player, err := hearts.NewPlayerFromFlag(playerType)
	if err != nil {
		return "", fmt.Errorf("couldn't create player: %v", err)
	}
	session, err := conn.Register(ctx, "", player)
	if err != nil {
		return "", fmt.Errorf("couldn't register with server: %v", err)
	}
	gameId, err = session.JoinGameAsPlayer(ctx, wg, gameId)
	if err != nil {
		return "", fmt.Errorf("couldn't join game: %v", err)
	}
	return gameId, nil
}
