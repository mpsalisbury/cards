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
//	logger     = log.New(os.Stdout, "", 0)
)

func init() {
	rand.Seed(time.Now().UnixNano())
}
func main() {
	flag.Parse()

	wg := new(sync.WaitGroup)
	wg.Add(1)
	ctx := context.Background()
	client, err := client.Connect(client.LocalServer)
	if err != nil {
		log.Fatalf("Couldn't connect to server: %v", err)
	}
	name := fmt.Sprintf("Observer%04d", rand.Intn(10000))
	err = client.Register(ctx, name, callbacks{client: client, wg: wg})
	if err != nil {
		log.Fatalf("Couldn't register with server: %v", err)
	}
	err = client.JoinGameAsObserver(ctx)
	if err != nil {
		log.Fatalf("Couldn't join game: %v", err)
	}
	wg.Wait()
}

// client.GameCallbacks
type callbacks struct {
	client.UnimplementedGameCallbacks
	client client.Connection
	wg     *sync.WaitGroup
}

func (callbacks) HandlePlayerJoined(name string) {
	fmt.Printf("Player joined: %s\n", name)
}
func (c callbacks) HandleGameStarted() {
	gameState, err := c.client.GetGameState(context.Background())
	if err != nil {
		log.Fatalf("Couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
}
func (c callbacks) HandleGameFinished() {
	gameState, err := c.client.GetGameState(context.Background())
	if err != nil {
		log.Fatalf("Couldn't get game state: %v", err)
	}
	fmt.Printf("Game over\n")
	fmt.Printf("%v\n", gameState)
	c.wg.Done()
}

func (c callbacks) HandleTrickCompleted() {
	ctx := context.Background()
	gameState, err := c.client.GetGameState(ctx)
	if err != nil {
		log.Fatalf("Couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
}
