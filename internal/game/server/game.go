package server

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/mpsalisbury/cards/internal/cards"
	pb "github.com/mpsalisbury/cards/internal/game/proto"
)

type game struct {
	id              string
	phase           gamePhase
	players         map[string]*player // Keyed by sessionId
	playerOrder     []string           // by sessionId
	currentTrick    *trick
	nextPlayerIndex int // index into playerOrder
	heartsBroken    bool
}

type gamePhase int8

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

func (g *game) addPlayer(session *session) {
	p := &player{name: session.name, sessionId: session.id}
	g.players[session.id] = p
	g.playerOrder = append(g.playerOrder, session.id)
}
func (g *game) containsPlayer(sessionId string) bool {
	_, ok := g.players[sessionId]
	return ok
}

// Remove player if present
func (g *game) removePlayer(sessionId string) error {
	if !g.containsPlayer(sessionId) {
		return nil
	}
	if g.phase != Preparing {
		return fmt.Errorf("can't remove player from game in Preparing phase.")
	}
	delete(g.players, sessionId)
	for i, s := range g.playerOrder {
		if s == sessionId {
			l := len(g.playerOrder)
			copy(g.playerOrder[i:], g.playerOrder[i+1:])
			g.playerOrder = g.playerOrder[:l-1]
			break
		}
	}
	return nil
}

// Return all players, starting with sessionId and following in order.
// If this session isn't a player, observer starts with the first player.
func (g *game) allPlayersInOrder(sessionId string) []*player {
	var matchingId = 0
	for i, sid := range g.playerOrder {
		if sid == sessionId {
			matchingId = i
			break
		}
	}
	players := []*player{}
	for i := 0; i <= 3; i++ {
		opIndex := (matchingId + i) % 4
		op := g.players[g.playerOrder[opIndex]]
		players = append(players, op)
	}
	return players
}

// Returns true if started.
func (g *game) startIfReady() bool {
	if g.phase != Preparing {
		return false
	}
	log.Printf("Game %s - numPlayers %d", g.id, len(g.players))
	if len(g.players) != 4 {
		return false
	}
	log.Printf("Enough players, initializing game")
	for i, h := range cards.Deal(4) {
		sessionId := g.playerOrder[i]
		g.players[sessionId].cards = h
	}
	g.nextPlayerIndex = rand.Intn(4)
	g.phase = Playing
	return true
}

func phaseToProto(phase gamePhase) pb.GameState_Phase {
	switch phase {
	case Preparing:
		return pb.GameState_Preparing
	case Playing:
		return pb.GameState_Playing
	case Completed:
		return pb.GameState_Completed
	case Aborted:
		return pb.GameState_Aborted
	default:
		return pb.GameState_Unknown
	}
}

func (g game) getGameState(sessionId string) (*pb.GameState, error) {
	_, requesterIsPlayer := g.players[sessionId]
	phase := phaseToProto(g.phase)
	if g.phase != Playing && g.phase != Completed {
		return &pb.GameState{Phase: phase}, nil
	}
	players := []*pb.GameState_Player{}
	for _, p := range g.allPlayersInOrder(sessionId) {
		hideOtherPlayerState := requesterIsPlayer && (p.sessionId != sessionId)
		players = append(players, g.playerState(p, hideOtherPlayerState))
	}
	currentTrick := toCardsProto(g.currentTrick.cards)

	gs := &pb.GameState{
		Phase:        phase,
		Players:      players,
		CurrentTrick: currentTrick,
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

func toCardsProto(cards cards.Cards) *pb.GameState_Cards {
	return &pb.GameState_Cards{
		Cards: cards.Strings(),
	}
}

func (g game) playerState(p *player, hideOther bool) *pb.GameState_Player {
	ps := &pb.GameState_Player{
		Name:         p.name,
		NumCards:     int32(len(p.cards)),
		NumTricks:    int32(len(p.tricks)),
		TrickScore:   int32(p.trickScore),
		IsNextPlayer: p.sessionId == g.playerOrder[g.nextPlayerIndex],
	}
	if !hideOther {
		ps.Cards = toCardsProto(p.cards)
		ts := []*pb.GameState_Cards{}
		for _, t := range p.tricks {
			ts = append(ts, toCardsProto(t))
		}
		ps.Tricks = ts
	}
	return ps
}
