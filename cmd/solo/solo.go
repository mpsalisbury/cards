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
	hearts "github.com/mpsalisbury/cards/pkg/game/hearts/player"
)

var (
	verbose    = flag.Bool("verbose", false, "Print extra information during the session")
	name       = flag.String("name", "", "Your player name")
	playerType = "basic"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	hearts.AddPlayerFlag(&playerType, "type")
}
func main() {
	flag.Parse()
	err := runPlayer()
	if err != nil {
		log.Fatal(err)
	}
}
func runPlayer() error {
	conn, err := client.Connect(client.LocalServer, *verbose)
	if err != nil {
		return fmt.Errorf("couldn't connect to server: %v", err)
	}
	gameId, err := conn.CreateGame(context.Background())
	if err != nil {
		return err
	}
	wg := new(sync.WaitGroup)
	for i := 0; i < 3; i++ {
		err = startAutoPlayer(conn, wg, gameId)
		if err != nil {
			return err
		}
	}
	err = startCmdlinePlayer(conn, wg, gameId)
	if err != nil {
		return err
	}
	wg.Wait()
	gameState, err := conn.GetGameState(context.Background(), gameId)
	if err != nil {
		return err
	}
	fmt.Print(gameState)
	return nil
}

func startAutoPlayer(conn client.Connection, wg *sync.WaitGroup, gameId string) error {
	player, err := hearts.NewPlayerFromFlag(playerType)
	if err != nil {
		return fmt.Errorf("couldn't create player: %v", err)
	}
	return startPlayer(conn, wg, "", gameId, player)
}
func startCmdlinePlayer(conn client.Connection, wg *sync.WaitGroup, gameId string) error {
	return startPlayer(conn, wg, *name, gameId, hearts.NewTerminalPlayer())
}
func startPlayer(conn client.Connection, wg *sync.WaitGroup, name string, gameId string, callbacks client.GameCallbacks) error {
	ctx := context.Background()
	session, err := conn.Register(ctx, name, callbacks)
	if err != nil {
		return fmt.Errorf("couldn't register with server: %v", err)
	}
	err = session.JoinGame(ctx, wg, gameId)
	if err != nil {
		return fmt.Errorf("couldn't join game: %v", err)
	}
	wg.Add(1)
	return nil
}
