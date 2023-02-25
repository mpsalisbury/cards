package hearts

import (
	"fmt"
	"log"
	"time"

	"github.com/mpsalisbury/cards/pkg/cards"
	"github.com/mpsalisbury/cards/pkg/game"
	pb "github.com/mpsalisbury/cards/pkg/proto"
	"golang.org/x/exp/slices"
)

func NewGame(gameId string) game.Game {
	return &heartsGame{
		id:           gameId,
		phase:        game.Preparing,
		players:      make(map[string]*player),
		currentTrick: &trick{},
	}
}

type heartsGame struct {
	id               string
	lastActivityTime time.Time
	phase            game.GamePhase
	players          map[string]*player // Keyed by playerId
	playerOrder      []string           // by playerId
	listenerIds      []string           // all players and observers
	numTricksPlayed  int
	currentTrick     *trick
	nextPlayerIndex  int // index into playerOrder
	heartsBroken     bool
}

func (g heartsGame) Id() string {
	return g.id
}
func (g heartsGame) Phase() game.GamePhase {
	return g.phase
}
func (g heartsGame) GetLastActivityTime() time.Time {
	return g.lastActivityTime
}

func (g *heartsGame) touch() {
	g.lastActivityTime = time.Now()
}

func (g *heartsGame) Abort() {
	g.touch()
	g.phase = game.Aborted
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
	leadSuit := cs[0].Suit
	highValue := cs[0].Value
	highIndex := 0
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
	g.touch()
	p := &player{id: id, name: name}
	g.players[id] = p
	g.playerOrder = append(g.playerOrder, id)
	g.listenerIds = append(g.listenerIds, id)
}
func (g *heartsGame) AddObserver(name string, id string) {
	g.touch()
	g.listenerIds = append(g.listenerIds, id)
}
func (g heartsGame) ListenerIds() []string {
	return g.listenerIds
}
func (g heartsGame) containsPlayer(playerId string) bool {
	_, ok := g.players[playerId]
	return ok
}

// Remove player if present
func (g *heartsGame) RemovePlayer(playerId string) error {
	g.touch()
	if !g.containsPlayer(playerId) {
		return nil
	}
	if g.phase != game.Preparing {
		return fmt.Errorf("can't remove player from game not in Preparing phase.")
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
func (g heartsGame) allPlayersInOrder(playerId string) []*player {
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
func (g heartsGame) IsEnoughPlayersToStart() bool {
	if g.phase != game.Preparing {
		return false
	}
	//log.Printf("Game %s - numPlayers %d", g.id, len(g.players))
	if len(g.players) != 4 {
		return false
	}
	//log.Printf("Enough players, ready to start %s", g.id)
	return true
}

func (g *heartsGame) ConfirmPlayerReadyToStart(playerId string) error {
	g.touch()
	p, ok := g.players[playerId]
	if !ok {
		return fmt.Errorf("no player %s found", playerId)
	}
	//fmt.Printf("Player %s ready to start\n", playerId)
	p.isReadyToStart = true
	return nil
}

func (g heartsGame) UnconfirmedPlayerIds() []string {
	var ids []string
	for _, p := range g.players {
		if !p.isReadyToStart {
			ids = append(ids, p.id)
		}
	}
	return ids
}

func (g *heartsGame) StartGame() {
	g.touch()
	for i, h := range cards.Deal(4) {
		playerId := g.playerOrder[i]
		g.players[playerId].cards = h
	}
	g.nextPlayerIndex = g.findPlayerIndexWithCard(cards.C2c)
	g.phase = game.Playing
}
func (g heartsGame) findPlayerIndexWithCard(fc cards.Card) int {
	for i, pid := range g.playerOrder {
		p := g.players[pid]
		for _, c := range p.cards {
			if c == fc {
				return i
			}
		}
	}
	log.Fatalf("Unable to find player with card %s", fc)
	return 0
}

func (g heartsGame) nextPlayer() *player {
	return g.players[g.NextPlayerId()]
}
func (g heartsGame) NextPlayerId() string {
	return g.playerOrder[g.nextPlayerIndex]
}

func (g heartsGame) GetGameState(playerId string) (*pb.GameState, error) {
	_, requesterIsPlayer := g.players[playerId]
	if g.phase != game.Playing && g.phase != game.Completed {
		return &pb.GameState{Phase: g.phase.ToProto()}, nil
	}
	players := []*pb.GameState_Player{}
	for _, p := range g.allPlayersInOrder(playerId) {
		hideOtherPlayerState := requesterIsPlayer && (p.id != playerId)
		players = append(players, g.playerState(p, hideOtherPlayerState))
	}
	currentTrick := g.currentTrick.cards.ToProto()
	legalPlays := g.legalPlays()

	gs := &pb.GameState{
		Id:           g.id,
		Phase:        g.phase.ToProto(),
		Players:      players,
		CurrentTrick: currentTrick,
		LegalPlays:   legalPlays.ToProto(),
	}
	return gs, nil
}

type player struct {
	id             string
	name           string
	isReadyToStart bool
	cards          cards.Cards
	tricks         []cards.Cards
	trickScore     int // sum of all trick's scores
	handScore      int // when game is completed.
}

func (g heartsGame) playerState(p *player, hideOther bool) *pb.GameState_Player {
	ps := &pb.GameState_Player{
		Id:           p.id,
		Name:         p.name,
		NumCards:     int32(len(p.cards)),
		Tricks:       cards.ToProtos(p.tricks),
		NumTricks:    int32(len(p.tricks)),
		TrickScore:   int32(p.trickScore),
		HandScore:    int32(p.handScore),
		IsNextPlayer: p.id == g.playerOrder[g.nextPlayerIndex],
	}
	if !hideOther {
		ps.Cards = p.cards.ToProto()
	}
	return ps
}

func cardScore(c cards.Card) int {
	if c.Suit == cards.Hearts {
		return 1
	}
	if c.Value == cards.Queen && c.Suit == cards.Spades {
		return 13
	}
	return 0
}
func trickScore(cs cards.Cards) int {
	s := 0
	for _, c := range cs {
		s += cardScore(c)
	}
	return s
}
func (g heartsGame) legalPlays() cards.Cards {
	playerId := g.NextPlayerId()
	p, ok := g.players[playerId]
	if !ok {
		log.Fatalf("player %s not found for game %s", playerId, g.id)
		return cards.Cards{}
	}
	isValid := func(c cards.Card) bool {
		return isValidCardForTrick(c, g.currentTrick.cards, p.cards, g.numTricksPlayed == 0, g.heartsBroken)
	}
	var cs cards.Cards
	for _, c := range p.cards {
		if isValid(c) {
			cs = append(cs, c)
		}
	}
	return cs
}

func (g *heartsGame) HandlePlayCard(playerId string, card cards.Card, r game.Reporter) error {
	g.touch()
	if playerId != g.NextPlayerId() {
		return fmt.Errorf("it is not player %s's turn", playerId)
	}
	p, ok := g.players[playerId]
	if !ok {
		return fmt.Errorf("player %s not found for game %s", playerId, g.id)
	}
	if !slices.Contains(p.cards, card) {
		return fmt.Errorf("player %s does not have card %s", playerId, card)
	}
	if !isValidCardForTrick(card, g.currentTrick.cards, p.cards, g.numTricksPlayed == 0, g.heartsBroken) {
		return fmt.Errorf("player %s cannot play card %s", p.id, card)
	}
	if card.Suit == cards.Hearts {
		g.heartsBroken = true
	}
	p.cards = p.cards.Remove(card)
	g.currentTrick.addCard(card, p.id)
	r.ReportCardPlayed(g)

	if g.currentTrick.size() < 4 {
		g.nextPlayerIndex = (g.nextPlayerIndex + 1) % 4
		return nil
	}
	// Trick is over.
	winningTrick := g.currentTrick
	winningCard, winnerId := winningTrick.chooseWinner()
	winner := g.players[winnerId]
	log.Printf("%s trick: %s - winning card %s\n", g.id, winningTrick.cards, winningCard)
	winner.tricks = append(winner.tricks, g.currentTrick.cards)
	winner.trickScore += trickScore(winningTrick.cards)
	g.currentTrick = &trick{}
	g.numTricksPlayed++
	g.nextPlayerIndex = slices.Index(g.playerOrder, winnerId)
	r.ReportTrickCompleted(g, winningTrick.cards, winnerId, winner.name)

	// If next player has no more cards, we're done.
	if len(g.nextPlayer().cards) == 0 {
		g.phase = game.Completed
		if didSomeoneShootTheMoon(g.players) {
			for _, p := range g.players {
				p.handScore = 26 - p.trickScore
			}
		} else {
			for _, p := range g.players {
				p.handScore = p.trickScore
			}
		}
	}
	return nil
}

func didSomeoneShootTheMoon(players map[string]*player) bool {
	for _, p := range players {
		if p.trickScore == 26 {
			return true
		}
	}
	return false
}

func isValidCardForTrick(card cards.Card, trick cards.Cards, hand cards.Cards, isFirstTrick, heartsBroken bool) bool {
	// For first trick, must lead 2c.
	if isFirstTrick && len(trick) == 0 {
		return card == cards.C2c
	}
	// Can't break hearts or qs on first trick.
	if isFirstTrick {
		if card == cards.Cqs {
			return false
		}
		if card.Suit == cards.Hearts && len(hand.FilterBySuit(cards.Spades, cards.Clubs, cards.Diamonds)) > 0 {
			return false
		}
	}

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
