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
//	serverAddr = flag.String("server", "api.cards.salisburyclan.com:443", "Server address (host:port)")
// Raw server: "cards-api-5g5wrbokbq-uw.a.run.app:443"
//	insecure  = flag.Bool("insecure", false, "Use insecure connection to server")
//	local     = flag.Bool("local", false, "Override serverAddr and insecure connection for local server")
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
	name := fmt.Sprintf("Henry%04d", rand.Intn(10000))
	err = client.Register(ctx, name, callbacks{client, wg})
	if err != nil {
		log.Fatalf("Couldn't register with server: %v", err)
	}
	err = client.JoinGame(ctx)
	if err != nil {
		log.Fatalf("Couldn't join game: %v", err)
	}
	gameState, err := client.GetGameState(ctx)
	if err != nil {
		log.Fatalf("Couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
	wg.Wait()
}

// client.GameCallbacks
type callbacks struct {
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

func (c callbacks) HandleYourTurn() {
	ctx := context.Background()
	log.Println("Performing turn")
	gameState, err := c.client.GetGameState(ctx)
	if err != nil {
		log.Fatalf("Couldn't get game state: %v", err)
	}
	card := gameState.Players[0].Cards[0]
	c.client.PlayCard(ctx, card)
}
