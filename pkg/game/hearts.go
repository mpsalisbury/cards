package game

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/mpsalisbury/cards/pkg/cards"
	pb "github.com/mpsalisbury/cards/pkg/proto"
	"golang.org/x/exp/slices"
)

func NewHeartsGame(gameId string) Game {
	return &heartsGame{
		id:           gameId,
		phase:        Preparing,
		players:      make(map[string]*player),
		currentTrick: &trick{},
	}
}

type heartsGame struct {
	id              string
	phase           GamePhase
	players         map[string]*player // Keyed by playerId
	playerOrder     []string           // by playerId
	currentTrick    *trick
	nextPlayerIndex int // index into playerOrder
	heartsBroken    bool
}

func (g heartsGame) Id() string {
	return g.id
}
func (g heartsGame) Phase() GamePhase {
	return g.phase
}
func (g *heartsGame) Abort() {
	g.phase = Aborted
}

func (g heartsGame) PlayerNames() []string {
	names := []string{}
	for _, p := range g.players {
		names = append(names, p.name)
	}
	return names
}

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
	// Trick winner logic
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

func (g heartsGame) AcceptingMorePlayers() bool {
	return len(g.players) < 4
}

func (g *heartsGame) AddPlayer(name string, id string) {
	p := &player{name: name, playerId: id}
	g.players[id] = p
	g.playerOrder = append(g.playerOrder, id)
}
func (g *heartsGame) containsPlayer(playerId string) bool {
	_, ok := g.players[playerId]
	return ok
}

// Remove player if present
func (g *heartsGame) RemovePlayer(playerId string) error {
	if !g.containsPlayer(playerId) {
		return nil
	}
	if g.phase != Preparing {
		return fmt.Errorf("can't remove player from game in Preparing phase.")
	}
	delete(g.players, playerId)
	for i, s := range g.playerOrder {
		if s == playerId {
			l := len(g.playerOrder)
			copy(g.playerOrder[i:], g.playerOrder[i+1:])
			g.playerOrder = g.playerOrder[:l-1]
			break
		}
	}
	return nil
}

// Return all players, starting with playerId and following in order.
// If this playerId isn't a player, observer starts with the first player.
func (g *heartsGame) allPlayersInOrder(playerId string) []*player {
	var matchingId = 0
	for i, pid := range g.playerOrder {
		if pid == playerId {
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
func (g *heartsGame) StartIfReady() bool {
	if g.phase != Preparing {
		return false
	}
	log.Printf("Game %s - numPlayers %d", g.id, len(g.players))
	if len(g.players) != 4 {
		return false
	}
	log.Printf("Enough players, initializing game")
	for i, h := range cards.Deal(4) {
		playerId := g.playerOrder[i]
		g.players[playerId].cards = h
	}
	g.nextPlayerIndex = rand.Intn(4)
	g.phase = Playing
	return true
}

func (g heartsGame) NextPlayerId() string {
	return g.playerOrder[g.nextPlayerIndex]
}

func (g heartsGame) GetGameState(playerId string) (*pb.GameState, error) {
	_, requesterIsPlayer := g.players[playerId]
	if g.phase != Playing && g.phase != Completed {
		return &pb.GameState{Phase: g.phase.ToProto()}, nil
	}
	players := []*pb.GameState_Player{}
	for _, p := range g.allPlayersInOrder(playerId) {
		hideOtherPlayerState := requesterIsPlayer && (p.playerId != playerId)
		players = append(players, g.playerState(p, hideOtherPlayerState))
	}
	currentTrick := toCardsProto(g.currentTrick.cards)

	gs := &pb.GameState{
		Id:           g.id,
		Phase:        g.phase.ToProto(),
		Players:      players,
		CurrentTrick: currentTrick,
	}
	return gs, nil
}

type player struct {
	name       string
	playerId   string
	cards      cards.Cards
	tricks     []cards.Cards
	trickScore int
}

func toCardsProtos(tricks []cards.Cards) []*pb.GameState_Cards {
	ts := []*pb.GameState_Cards{}
	for _, t := range tricks {
		ts = append(ts, toCardsProto(t))
	}
	return ts
}
func toCardsProto(cards cards.Cards) *pb.GameState_Cards {
	return &pb.GameState_Cards{
		Cards: cards.Strings(),
	}
}

func (g heartsGame) playerState(p *player, hideOther bool) *pb.GameState_Player {
	ps := &pb.GameState_Player{
		Name:         p.name,
		NumCards:     int32(len(p.cards)),
		Tricks:       toCardsProtos(p.tricks),
		NumTricks:    int32(len(p.tricks)),
		TrickScore:   int32(p.trickScore),
		IsNextPlayer: p.playerId == g.playerOrder[g.nextPlayerIndex],
	}
	if !hideOther {
		ps.Cards = toCardsProto(p.cards)
	}
	return ps
}

func (g *heartsGame) HandlePlayCard(playerId string, card cards.Card, r Reporter) error {
	p, ok := g.players[playerId]
	if !ok {
		return fmt.Errorf("player not found for game %s in playerId %s", g.Id, playerId)
	}
	if !slices.Contains(p.cards, card) {
		return fmt.Errorf("player %s does not have card %s", p.playerId, card)
	}
	if !isValidCardForTrick(card, g.currentTrick.cards, p.cards, g.heartsBroken) {
		return fmt.Errorf("player %s cannot play card %s", p.playerId, card)
	}
	if card.Suit == cards.Hearts {
		g.heartsBroken = true
	}
	p.cards = p.cards.Remove(card)
	g.currentTrick.addCard(card, p.playerId)
	r.ReportCardPlayed()
	fmt.Printf("%s - %s\n", card, p.cards.HandString())

	if g.currentTrick.size() < 4 {
		g.nextPlayerIndex = (g.nextPlayerIndex + 1) % 4
		return nil
	}
	// Trick is over.
	winningCard, winnerId := g.currentTrick.chooseWinner()
	winner := g.players[winnerId]
	fmt.Printf("Trick: %s - winning card %s\n", g.currentTrick.cards, winningCard)
	winner.tricks = append(winner.tricks, g.currentTrick.cards)
	g.currentTrick = &trick{}
	g.nextPlayerIndex = slices.Index(g.playerOrder, winnerId)
	r.ReportTrickCompleted()

	// If next player has no more cards, we're done.
	if len(g.players[g.playerOrder[g.nextPlayerIndex]].cards) == 0 {
		g.phase = Completed
	}
	return nil
}

func isValidCardForTrick(card cards.Card, trick cards.Cards, hand cards.Cards, heartsBroken bool) bool {
	// Can play any lead card unless hearts haven't been broken.
	if len(trick) == 0 {
		if card.Suit != cards.Hearts {
			return true
		}
		if heartsBroken {
			return true
		}
		// if all cards are Hearts, it's okay
		for _, c := range hand {
			if c.Suit != cards.Hearts {
				return false
			}
		}
		return true
	}
	leadSuit := trick[0].Suit
	// If this card matches suit of lead card, we're good.
	if card.Suit == leadSuit {
		return true
	}
	// Else player must not have any of the lead suit in hand.
	for _, c := range hand {
		if c.Suit == leadSuit {
			return false
		}
	}
	return true
}
