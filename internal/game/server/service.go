package server

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/mpsalisbury/cards/internal/cards"
	pb "github.com/mpsalisbury/cards/internal/game/proto"
	"golang.org/x/exp/slices"
)

func NewCardGameService() pb.CardGameServiceServer {
	return &cardGameService{
		sessions: make(map[string]*session),
		game: &game{
			phase:        Preparing,
			players:      make(map[string]*player),
			currentTrick: &trick{},
		},
	}
}

type cardGameService struct {
	pb.UnimplementedCardGameServiceServer
	sessions   map[string]*session // Keyed by sessionId
	game       *game
	pingTicker *time.Ticker
}

type activityReport = *pb.GameActivityResponse

type session struct {
	id   string
	name string
	ch   chan activityReport
}

func (cardGameService) Ping(ctx context.Context, request *pb.PingRequest) (*pb.PingResponse, error) {
	log.Printf("Got ping %s", request.GetMessage())
	return &pb.PingResponse{Message: "Pong"}, nil
}

func (s *cardGameService) newSessionId() string {
	for {
		id := fmt.Sprintf("s%08d", rand.Int31n(100000000))
		// Ensure no collision with existing session id.
		if _, found := s.sessions[id]; !found {
			return id
		}
	}
}
func (s *cardGameService) addSession(name string) string {
	sessionId := s.newSessionId()
	p := &session{
		id:   sessionId,
		name: name,
	}
	s.sessions[sessionId] = p
	return sessionId
}
func (s *cardGameService) removeSession(sessionId string) {
	log.Printf("Closing session %s\n", sessionId)
	delete(s.sessions, sessionId)
	err := s.game.removePlayer(sessionId)
	if err != nil {
		// Can't remove player, abort game
		s.game.phase = Aborted
		s.reportGameAborted()
		s.scheduleGameRemoved()
	}
}
func (s *cardGameService) scheduleGameRemoved() {
	// TODO: When we support multiple games, clean this game up after time (1 minute?)
}

func (s *cardGameService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	sessionId := s.addSession(req.GetName())
	return &pb.RegisterResponse{SessionId: sessionId}, nil
}

func (s *cardGameService) JoinGame(ctx context.Context, req *pb.JoinGameRequest) (*pb.JoinGameResponse, error) {
	sessionId := req.GetSessionId()
	session, ok := s.sessions[sessionId]
	if !ok {
		return nil, fmt.Errorf("SessionId %s not found", sessionId)
	}
	if req.GetMode() == pb.JoinGameRequest_AsPlayer {
		if !s.game.acceptingMorePlayers() {
			return nil, fmt.Errorf("game is full")
		}
		s.game.addPlayer(session)
		s.reportPlayerJoined(session.name)
		if s.game.startIfReady() {
			s.reportGameStarted()
			s.reportYourTurn()
		}
	}
	return &pb.JoinGameResponse{}, nil
}

func (s *cardGameService) GetGameState(ctx context.Context, req *pb.GameStateRequest) (*pb.GameState, error) {
	sessionId := req.GetSessionId()
	_, ok := s.sessions[sessionId]
	if !ok {
		return nil, fmt.Errorf("SessionId %s not found", sessionId)
	}
	return s.game.getGameState(sessionId)
}

func (s *cardGameService) PlayerAction(ctx context.Context, req *pb.PlayerActionRequest) (*pb.Status, error) {
	switch r := req.Type.(type) {
	case *pb.PlayerActionRequest_PlayCard:
		sessionId := req.GetSessionId()
		card, _ := cards.ParseCard(r.PlayCard.GetCard())
		err := s.handlePlayCard(sessionId, card)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("PlayerActionRequest has unexpected type %T", r)
	}
	return &pb.Status{Code: 0}, nil
}

func (s *cardGameService) handlePlayCard(sessionId string, card cards.Card) error {
	//log.Printf("handlePlayCard %s %s", sessionId, card)
	p, ok := s.game.players[sessionId]
	if !ok {
		return fmt.Errorf("SessionId %s not found", sessionId)
	}
	if !slices.Contains(p.cards, card) {
		return fmt.Errorf("SessionId %s does not contain card %s", sessionId, card)
	}
	if !isValidCardForTrick(card, s.game.currentTrick.cards, p.cards, s.game.heartsBroken) {
		return fmt.Errorf("SessionId %s cannot play card %s", sessionId, card)
	}
	if card.Suit == cards.Hearts {
		s.game.heartsBroken = true
	}
	p.cards = p.cards.Remove(card)
	s.game.currentTrick.addCard(card, sessionId)
	s.reportCardPlayed()
	fmt.Printf("%s - %s\n", card, p.cards.HandString())

	if s.game.currentTrick.size() < 4 {
		s.game.nextPlayerIndex = (s.game.nextPlayerIndex + 1) % 4
		s.reportYourTurn()
		return nil
	}
	// Trick is over.
	winningCard, winnerId := s.game.currentTrick.chooseWinner()
	winner := s.game.players[winnerId]
	fmt.Printf("Trick: %s - winning card %s\n", s.game.currentTrick.cards, winningCard)
	winner.tricks = append(winner.tricks, s.game.currentTrick.cards)
	s.game.currentTrick = &trick{}
	s.game.nextPlayerIndex = slices.Index(s.game.playerOrder, winnerId)
	s.reportTrickCompleted()

	// If next player has more cards, keep playing.
	if len(s.game.players[s.game.playerOrder[s.game.nextPlayerIndex]].cards) > 0 {
		s.reportYourTurn()
		return nil
	}

	// Game is over
	s.wrapUpGame()
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
		// if all cards are hearts, it's okay
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

func (s *cardGameService) wrapUpGame() {
	s.game.phase = Completed
	s.reportGameFinished()
	s.scheduleGameRemoved()
}

// Broadcasts message to all clients.
func (s *cardGameService) broadcastMessage(msg string) {
	s.reportActivity(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_BroadcastMsg{BroadcastMsg: msg},
		})
}
func (s *cardGameService) reportPlayerJoined(name string) {
	s.reportActivity(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_PlayerJoined_{
				PlayerJoined: &pb.GameActivityResponse_PlayerJoined{Name: name},
			},
		})
}
func (s *cardGameService) reportGameStarted() {
	s.reportActivity(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_GameStarted_{},
		})
}
func (s *cardGameService) reportCardPlayed() {
	s.reportActivity(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_CardPlayed_{},
		})
}
func (s *cardGameService) reportTrickCompleted() {
	s.reportActivity(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_TrickCompleted_{},
		})
}
func (s *cardGameService) reportGameFinished() {
	s.reportActivity(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_GameFinished_{},
		})
}
func (s *cardGameService) reportGameAborted() {
	s.reportActivity(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_GameAborted_{},
		})
}
func (s *cardGameService) reportActivity(activity activityReport) {
	for _, p := range s.sessions {
		if p.ch != nil {
			p.ch <- activity
		}
	}
}
func (s *cardGameService) reportYourTurn() {
	g := s.game
	if g.phase != Playing {
		log.Print("Can't report your turn for game not in state Playing")
		return
	}
	pId := s.game.playerOrder[g.nextPlayerIndex]
	ch := s.sessions[pId].ch
	yourTurn := &pb.GameActivityResponse{
		Type: &pb.GameActivityResponse_YourTurn_{},
	}
	ch <- yourTurn
}

func (s *cardGameService) ListenForGameActivity(req *pb.GameActivityRequest, resp pb.CardGameService_ListenForGameActivityServer) error {
	sessionId := req.GetSessionId()
	log.Printf("ListenForGameActivity from %s - %s\n", sessionId, s.sessions[sessionId].name)
	ch := make(chan activityReport)
	s.sessions[sessionId].ch = ch
	err := reportActivity(ch, resp)
	close(ch)
	s.removeSession(sessionId)
	return err
}

func reportActivity(activityCh chan activityReport, server pb.CardGameService_ListenForGameActivityServer) error {
	for {
		select {
		case activity := <-activityCh:
			err := server.Send(activity)
			if err != nil {
				return err
			}
			if _, isFinished := activity.Type.(*pb.GameActivityResponse_GameFinished_); isFinished {
				// Game is over. Close this reporting request.
				return nil
			}
		case <-server.Context().Done():
			return server.Context().Err()
		}
	}
}
