package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/mpsalisbury/cards/internal/cards"
	"github.com/mpsalisbury/cards/internal/game/client"
)

var (
	//	logger     = log.New(os.Stdout, "", 0)
	//	serverAddr = flag.String("server", "api.cards.salisburyclan.com:443", "Server address (host:port)")
	// Raw server: "cards-api-5g5wrbokbq-uw.a.run.app:443"
	//	insecure  = flag.Bool("insecure", false, "Use insecure connection to server")
	//	local     = flag.Bool("local", false, "Override serverAddr and insecure connection for local server")
	playCards = flag.Bool("playcards", false, "Play all cards automatically")
)

func init() {
	rand.Seed(time.Now().UnixNano())
}
func main() {
	flag.Parse()

	ctx := context.Background()
	client, err := client.Connect(client.LocalServer)
	if err != nil {
		log.Fatalf("Couldn't connect to server: %v", err)
	}
	name := fmt.Sprintf("Henry%04d", rand.Intn(10000))
	err = client.Register(ctx, name, callbacks{client})
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
	if *playCards {
		deck := cards.MakeDeck()[:13]
		for _, c := range deck {
			err := client.PlayCard(ctx, c.String())
			if err != nil {
				log.Printf("Couldn't play card: %v", err)
			}
		}
	} else {
		time.Sleep(time.Second * 100)
	}
}

// client.GameCallbacks
type callbacks struct {
	client client.Connection
}

func (c callbacks) HandlePlayerJoined(name string) {
	fmt.Printf("Player joined: %s\n", name)
}
func (c callbacks) HandleGameStarted() {
	gameState, err := c.client.GetGameState(context.Background())
	if err != nil {
		log.Fatalf("Couldn't get game state: %v", err)
	}
	fmt.Printf("%v\n", gameState)
}

func (callbacks) HandleYourTurn() {
	log.Println("Performing turn")
	// implement
}
