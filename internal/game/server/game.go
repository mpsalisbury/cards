package server

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/mpsalisbury/cards/internal/cards"
	pb "github.com/mpsalisbury/cards/internal/game/proto"
)

type game struct {
	state           gamePhase
	players         map[string]*player // Keyed by sessionId
	playerOrder     []string           // by sessionId
	currentTrick    *trick
	nextPlayerIndex int // index into playerOrder
	heartsBroken bool
}

type gamePhase = int8

const (
	Preparing gamePhase = iota
	Playing
	Completed
	Aborted
)

type trick struct {
	cards     cards.Cards
	playerIds []string
}

func (t *trick) size() int {
	return len(t.cards)
}
func (t *trick) addCard(card cards.Card, playerId string) {
	t.cards = append(t.cards, card)
	t.playerIds = append(t.playerIds, playerId)
}
func (t *trick) chooseWinner() (cards.Card, string) {
	// Hearts trick winner logic
	cs := t.cards
	highIndex := 0
	leadSuit := cs[highIndex].Suit
	highValue := cs[highIndex].Value
	for i, c := range cs {
		if c.Suit == leadSuit && c.Value > highValue {
			highValue = c.Value
			highIndex = i
		}
	}
	return cs[highIndex], t.playerIds[highIndex]
}

func (g game) acceptingMorePlayers() bool {
	return len(g.players) < 4
}

func (g *game) addPlayer(playerSession *playerSession) {
	sessionId := playerSession.sessionId
	p := &player{name: playerSession.name, sessionId: playerSession.sessionId}
	g.players[sessionId] = p
	g.playerOrder = append(g.playerOrder, sessionId)
}

// Return other players, starting with the player after sessionId.
func (g *game) otherPlayers(sessionId string) []*player {
	var matchingId = -1
	for i, sid := range g.playerOrder {
		if sid == sessionId {
			matchingId = i
			break
		}
	}
	if matchingId == -1 {
		return []*player{}
	}
	otherPlayers := []*player{}
	for i := 1; i <= 3; i++ {
		opIndex := (matchingId + i) % 4
		op := g.players[g.playerOrder[opIndex]]
		otherPlayers = append(otherPlayers, op)
	}
	return otherPlayers
}

// Returns true if started.
func (g *game) startIfReady() bool {
	if g.state != Preparing {
		return false
	}
	log.Printf("numPlayers: %d", len(g.players))
	if len(g.players) != 4 {
		return false
	}
	log.Printf("Enough players, initializing game")
	for i, h := range cards.Deal(4) {
		sessionId := g.playerOrder[i]
		g.players[sessionId].cards = h
	}
	g.nextPlayerIndex = rand.Intn(4)
	g.state = Playing
	return true
}

func (g game) getGameState(sessionId string) (*pb.GameStateResponse, error) {
	p, ok := g.players[sessionId]
	if !ok {
		return nil, fmt.Errorf("SessionId %s not found", sessionId)
	}
	var state pb.GameStateResponse_GameState
	switch g.state {
	case Preparing:
		state = pb.GameStateResponse_Preparing
	case Playing:
		state = pb.GameStateResponse_Playing
	case Completed:
		state = pb.GameStateResponse_Completed
	case Aborted:
		state = pb.GameStateResponse_Aborted
	default:
		state = pb.GameStateResponse_Unknown
	}
	if g.state != Playing && g.state != Completed {
		return &pb.GameStateResponse{State: state}, nil
	}
	player := g.yourPlayerState(p)
	otherPlayers := []*pb.GameStateResponse_OtherPlayerState{}
	for _, op := range g.otherPlayers(sessionId) {
		otherPlayers = append(otherPlayers, g.otherPlayerState(op))
	}
	currentTrick := g.currentTrick.cards.Strings()

	gs := &pb.GameStateResponse{
		State:             state,
		Player:            player,
		OtherPlayers:      otherPlayers,
		CurrentTrickCards: currentTrick,
	}
	return gs, nil
}

type player struct {
	name       string
	sessionId  string
	cards      cards.Cards
	tricks     []cards.Cards
	trickScore int
}

func (g game) yourPlayerState(p *player) *pb.GameStateResponse_YourPlayerState {
	return &pb.GameStateResponse_YourPlayerState{
		Name:           p.name,
		Cards:          p.cards.Strings(),
		NumTricksTaken: int32(len(p.tricks)),
		TrickScore:     int32(p.trickScore),
		IsNextPlayer:   p.sessionId == g.playerOrder[g.nextPlayerIndex],
	}
}

func (g game) otherPlayerState(p *player) *pb.GameStateResponse_OtherPlayerState {
	return &pb.GameStateResponse_OtherPlayerState{
		Name:              p.name,
		NumCardsRemaining: int32(len(p.cards)),
		NumTricksTaken:    int32(len(p.tricks)),
		IsNextPlayer:      p.sessionId == g.playerOrder[g.nextPlayerIndex],
	}
}
