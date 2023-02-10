package server

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/mpsalisbury/cards/internal/cards"
	pb "github.com/mpsalisbury/cards/internal/game/proto"
)

func NewCardGameService() pb.CardGameServiceServer {
	return &cardGameService{
		playerSessions: make(map[string]*playerSession),
		game: &game{
			state:   Preparing,
			players: make(map[string]*player),
		},
	}
}

type cardGameService struct {
	pb.UnimplementedCardGameServiceServer
	playerSessions map[string]*playerSession // Keyed by sessionId
	game           *game
}

type activityReport = *pb.GameActivityResponse

type playerSession struct {
	sessionId string
	name      string
	ch        chan activityReport
}

type gamePhase = int8

const (
	Preparing gamePhase = iota
	Playing
	Completed
	Aborted
)

type game struct {
	state             gamePhase
	players           map[string]*player // Keyed by sessionId
	playerOrder       []string           // by sessionId
	currentTrickCards []string
	nextPlayerId      string // sessionId
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
	g.nextPlayerId = g.playerOrder[rand.Intn(4)]
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
	currentTrickCards := g.currentTrickCards

	gs := &pb.GameStateResponse{
		State:             state,
		Player:            player,
		OtherPlayers:      otherPlayers,
		CurrentTrickCards: currentTrickCards,
	}
	return gs, nil
}

type player struct {
	name           string
	sessionId      string
	cards          cards.Cards
	numTricksTaken int
	trickScore     int
}

func (g game) yourPlayerState(p *player) *pb.GameStateResponse_YourPlayerState {
	return &pb.GameStateResponse_YourPlayerState{
		Name:           p.name,
		Cards:          p.cards.Strings(),
		NumTricksTaken: int32(p.numTricksTaken),
		TrickScore:     int32(p.trickScore),
		IsNextPlayer:   p.sessionId == g.nextPlayerId,
	}
}

func (g game) otherPlayerState(p *player) *pb.GameStateResponse_OtherPlayerState {
	return &pb.GameStateResponse_OtherPlayerState{
		Name:              p.name,
		NumCardsRemaining: int32(len(p.cards)),
		NumTricksTaken:    int32(p.numTricksTaken),
		IsNextPlayer:      p.sessionId == g.nextPlayerId,
	}
}

func (cardGameService) Ping(ctx context.Context, request *pb.PingRequest) (*pb.PingResponse, error) {
	log.Printf("Got ping %s", request.GetMessage())
	return &pb.PingResponse{Message: "Pong"}, nil
}

func (s *cardGameService) newSessionId() string {
	for {
		id := fmt.Sprintf("s%08d", rand.Int31n(100000000))
		// Ensure no collision with existing session id.
		if _, found := s.playerSessions[id]; !found {
			return id
		}
	}
}

func (s *cardGameService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	name := req.GetName()
	sessionId := s.newSessionId()
	p := &playerSession{
		sessionId: sessionId,
		name:      name,
	}
	s.playerSessions[sessionId] = p
	return &pb.RegisterResponse{SessionId: sessionId}, nil
}

func (s *cardGameService) JoinGame(ctx context.Context, req *pb.JoinGameRequest) (*pb.JoinGameResponse, error) {
	sessionId := req.GetSessionId()
	playerSession, ok := s.playerSessions[sessionId]
	if !ok {
		return nil, fmt.Errorf("SessionId %s not found", sessionId)
	}
	if !s.game.acceptingMorePlayers() {
		return nil, fmt.Errorf("game is full")
	}
	s.game.addPlayer(playerSession)
	s.reportPlayerJoinedToAll(playerSession.name)
	if s.game.startIfReady() {
		s.reportGameStartedToAll()
		s.reportYourTurn()
	}
	return &pb.JoinGameResponse{}, nil
}

func (s cardGameService) GetGameState(ctx context.Context, req *pb.GameStateRequest) (*pb.GameStateResponse, error) {
	sessionId := req.GetSessionId()
	return s.game.getGameState(sessionId)
}

func (s cardGameService) PlayerAction(ctx context.Context, req *pb.PlayerActionRequest) (*pb.Status, error) {
	switch r := req.Type.(type) {
	case *pb.PlayerActionRequest_PlayCard:
		card := r.PlayCard.Card
		s.reportMessageToAll(fmt.Sprintf("Playing %s", card))
	default:
		return nil, fmt.Errorf("PlayerActionRequest has unexpected type %T", r)
	}
	return &pb.Status{Code: 0}, nil
}

// Reports message to all clients.
func (s cardGameService) reportMessageToAll(msg string) {
	s.reportActivityToAll(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_Msg{Msg: msg},
		})
}
func (s cardGameService) reportPlayerJoinedToAll(name string) {
	s.reportActivityToAll(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_PlayerJoined{
				PlayerJoined: &pb.PlayerJoined{Name: name},
			},
		})
}
func (s cardGameService) reportGameStartedToAll() {
	s.reportActivityToAll(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_GameStarted{},
		})
}
func (s cardGameService) reportActivityToAll(activity activityReport) {
	for _, p := range s.playerSessions {
		if p.ch != nil {
			p.ch <- activity
		}
	}
}
func (s cardGameService) reportYourTurn() {
	g := s.game
	if g.state != Playing {
		log.Print("Can't report your turn for game not in state Playing")
		return
	}
	ch := s.playerSessions[g.nextPlayerId].ch
	yourTurn := &pb.GameActivityResponse{
		Type: &pb.GameActivityResponse_YourTurn{},
	}
	ch <- yourTurn
}

func (s *cardGameService) ListenForGameActivity(request *pb.GameActivityRequest, server pb.CardGameService_ListenForGameActivityServer) error {
	sessionId := request.GetSessionId()
	log.Printf("ListenForGameActivity from %s - %s\n", sessionId, s.playerSessions[sessionId].name)
	ch := make(chan activityReport)
	s.playerSessions[sessionId].ch = ch
	reportActivity(ch, server)
	close(ch)
	s.playerSessions[sessionId].ch = nil
	log.Printf("Closing connection from %s\n", sessionId)
	return nil
}

func reportActivity(c chan activityReport, server pb.CardGameService_ListenForGameActivityServer) {
	for activity := range c {
		err := server.Send(activity)
		if err != nil {
			log.Printf("Error sending message: %v", err)
			break
		}
	}
}
