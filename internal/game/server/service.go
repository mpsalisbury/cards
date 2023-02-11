package server

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/mpsalisbury/cards/internal/cards"
	pb "github.com/mpsalisbury/cards/internal/game/proto"
	"golang.org/x/exp/slices"
)

func NewCardGameService() pb.CardGameServiceServer {
	return &cardGameService{
		playerSessions: make(map[string]*playerSession),
		game: &game{
			state:        Preparing,
			players:      make(map[string]*player),
			currentTrick: &trick{},
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

func (s *cardGameService) GetGameState(ctx context.Context, req *pb.GameStateRequest) (*pb.GameStateResponse, error) {
	sessionId := req.GetSessionId()
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
	s.game.state = Completed
	s.reportGameFinishedToAll()
}

// Reports message to all clients.
func (s *cardGameService) reportMessageToAll(msg string) {
	s.reportActivityToAll(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_Msg{Msg: msg},
		})
}
func (s *cardGameService) reportPlayerJoinedToAll(name string) {
	s.reportActivityToAll(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_PlayerJoined{
				PlayerJoined: &pb.PlayerJoined{Name: name},
			},
		})
}
func (s *cardGameService) reportGameStartedToAll() {
	s.reportActivityToAll(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_GameStarted{},
		})
}
func (s *cardGameService) reportGameFinishedToAll() {
	s.reportActivityToAll(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_GameFinished{},
		})
}
func (s *cardGameService) reportActivityToAll(activity activityReport) {
	for _, p := range s.playerSessions {
		if p.ch != nil {
			p.ch <- activity
		}
	}
}
func (s *cardGameService) reportYourTurn() {
	g := s.game
	if g.state != Playing {
		log.Print("Can't report your turn for game not in state Playing")
		return
	}
	pId := s.game.playerOrder[g.nextPlayerIndex]
	ch := s.playerSessions[pId].ch
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
