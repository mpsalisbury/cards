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
		games:    make(map[string]*game),
	}
}

type cardGameService struct {
	pb.UnimplementedCardGameServiceServer
	sessions   map[string]*session // Keyed by sessionId
	games      map[string]*game    // Keyed by gameId
	pingTicker *time.Ticker
}

type activityReport = *pb.GameActivityResponse

type session struct {
	id     string
	name   string
	gameId string
	ch     chan activityReport
}

func (cardGameService) Ping(ctx context.Context, request *pb.PingRequest) (*pb.PingResponse, error) {
	log.Printf("Got ping %s", request.GetMessage())
	return &pb.PingResponse{Message: "Pong"}, nil
}

func (s *cardGameService) newSessionId() string {
	for {
		id := fmt.Sprintf("s%04d", rand.Int31n(10000))
		// Ensure no collision with existing session id.
		if _, found := s.sessions[id]; !found {
			return id
		}
	}
}
func (s *cardGameService) addSession(name string) string {
	sessionId := s.newSessionId()
	sess := &session{
		id:   sessionId,
		name: name,
	}
	s.sessions[sessionId] = sess
	return sessionId
}

func (s *cardGameService) newGameId() string {
	for {
		id := fmt.Sprintf("g%04d", rand.Int31n(10000))
		// Ensure no collision with existing game id.
		if _, found := s.games[id]; !found {
			return id
		}
	}
}
func (s *cardGameService) addGame() *game {
	gameId := s.newGameId()
	g := &game{
		id:           gameId,
		phase:        Preparing,
		players:      make(map[string]*player),
		currentTrick: &trick{},
	}
	s.games[gameId] = g
	return g
}

func (s *cardGameService) removeSession(sessionId string) error {
	log.Printf("Closing session %s\n", sessionId)
	session, ok := s.sessions[sessionId]
	if !ok {
		return fmt.Errorf("can't find session %s", sessionId)
	}
	delete(s.sessions, sessionId)
	if game, found := s.games[session.gameId]; found {
		err := game.removePlayer(sessionId)
		if err != nil {
			// Can't remove player, abort game
			game.phase = Aborted
			s.reportGameAborted()
			s.scheduleGameRemoved()
		} else {
			s.reportPlayerLeft(sessionId, game.id)
		}
	}
	return nil
}
func (s *cardGameService) scheduleGameRemoved() {
	// TODO: When we support multiple games, clean this game up after time (1 minute?)
}

func (s *cardGameService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	sessionId := s.addSession(req.GetName())
	return &pb.RegisterResponse{SessionId: sessionId}, nil
}
func (s *cardGameService) ListGames(ctx context.Context, req *pb.ListGamesRequest) (*pb.ListGamesResponse, error) {
	filter := makeGameFilter(req.GetPhase())
	var games []*pb.ListGamesResponse_GameSummary
	for _, g := range s.games {
		if filter(g) {
			names := []string{}
			for _, p := range g.players {
				names = append(names, p.name)
			}
			games = append(games, &pb.ListGamesResponse_GameSummary{
				Id:          g.id,
				Phase:       phaseToProto(g.phase),
				PlayerNames: names,
			})
		}
	}
	return &pb.ListGamesResponse{
		Games: games,
	}, nil
}

// Builds filter that accepts only games with one of the given phases (or any phase if no phases listed).
func makeGameFilter(phases []pb.GameState_Phase) func(*game) bool {
	return func(g *game) bool {
		if len(phases) == 0 {
			return true
		}
		for _, ph := range phases {
			if phaseToProto(g.phase) == ph {
				return true
			}
		}
		return false
	}
}

func (s *cardGameService) JoinGame(ctx context.Context, req *pb.JoinGameRequest) (*pb.JoinGameResponse, error) {
	sessionId := req.GetSessionId()
	gameId := req.GetGameId()
	session, ok := s.sessions[sessionId]
	if !ok {
		return nil, fmt.Errorf("SessionId %s not found", sessionId)
	}
	var game *game
	if gameId == "" {
		game = s.addGame()
		gameId = game.id
	} else {
		game, ok = s.games[gameId]
		if !ok {
			return nil, fmt.Errorf("game %s not found", gameId)
		}
	}
	session.gameId = gameId
	if req.GetMode() == pb.JoinGameRequest_AsPlayer {
		if !game.acceptingMorePlayers() {
			return nil, fmt.Errorf("game %s is full", gameId)
		}
		game.addPlayer(session)
		s.reportPlayerJoined(session.name, gameId)
		if game.startIfReady() {
			s.reportGameStarted()
			s.reportYourTurn(game)
		}
	}
	return &pb.JoinGameResponse{}, nil
}

func (s *cardGameService) GetGameState(ctx context.Context, req *pb.GameStateRequest) (*pb.GameState, error) {
	sessionId := req.GetSessionId()
	session, found := s.sessions[sessionId]
	if !found {
		return nil, fmt.Errorf("no session found for SessionId %s", sessionId)
	}
	game, found := s.games[session.gameId]
	if !found {
		return nil, fmt.Errorf("no game found for SessionId %s : %s", sessionId, session.gameId)
	}
	return game.getGameState(sessionId)
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
	session, found := s.sessions[sessionId]
	if !found {
		return fmt.Errorf("SessionId %s not found", sessionId)
	}
	game, found := s.games[session.gameId]
	if !found {
		return fmt.Errorf("no game %s found for SessionId %s", session.gameId, sessionId)
	}
	p, ok := game.players[sessionId]
	if !ok {
		return fmt.Errorf("player not found for game %s in SessionId %s", session.gameId, sessionId)
	}
	if !slices.Contains(p.cards, card) {
		return fmt.Errorf("SessionId %s does not contain card %s", sessionId, card)
	}
	if !isValidCardForTrick(card, game.currentTrick.cards, p.cards, game.heartsBroken) {
		return fmt.Errorf("SessionId %s cannot play card %s", sessionId, card)
	}
	if card.Suit == cards.Hearts {
		game.heartsBroken = true
	}
	p.cards = p.cards.Remove(card)
	game.currentTrick.addCard(card, sessionId)
	s.reportCardPlayed()
	fmt.Printf("%s - %s\n", card, p.cards.HandString())

	if game.currentTrick.size() < 4 {
		game.nextPlayerIndex = (game.nextPlayerIndex + 1) % 4
		s.reportYourTurn(game)
		return nil
	}
	// Trick is over.
	winningCard, winnerId := game.currentTrick.chooseWinner()
	winner := game.players[winnerId]
	fmt.Printf("Trick: %s - winning card %s\n", game.currentTrick.cards, winningCard)
	winner.tricks = append(winner.tricks, game.currentTrick.cards)
	game.currentTrick = &trick{}
	game.nextPlayerIndex = slices.Index(game.playerOrder, winnerId)
	s.reportTrickCompleted()

	// If next player has more cards, keep playing.
	if len(game.players[game.playerOrder[game.nextPlayerIndex]].cards) > 0 {
		s.reportYourTurn(game)
		return nil
	}

	// Game is over
	s.wrapUpGame(game)
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

func (s *cardGameService) wrapUpGame(game *game) {
	game.phase = Completed
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
func (s *cardGameService) reportPlayerJoined(name string, gameId string) {
	s.reportActivity(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_PlayerJoined_{
				PlayerJoined: &pb.GameActivityResponse_PlayerJoined{Name: name, GameId: gameId},
			},
		})
}
func (s *cardGameService) reportPlayerLeft(name string, gameId string) {
	s.reportActivity(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_PlayerLeft_{
				PlayerLeft: &pb.GameActivityResponse_PlayerLeft{Name: name, GameId: gameId},
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
func (s *cardGameService) reportYourTurn(game *game) {
	if game.phase != Playing {
		log.Print("Can't report your turn for game not in state Playing")
		return
	}
	pId := game.playerOrder[game.nextPlayerIndex]
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
