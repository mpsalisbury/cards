package server

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/mpsalisbury/cards/pkg/cards"
	"github.com/mpsalisbury/cards/pkg/game"
	"github.com/mpsalisbury/cards/pkg/game/hearts"
	pb "github.com/mpsalisbury/cards/pkg/proto"
)

func NewCardGameService() pb.CardGameServiceServer {
	return &cardGameService{
		players: make(map[string]*playerSession),
		games:   make(map[string]game.Game),
	}
}

// ListPlayers
// ClosePlayerSession
//	delete(s.players, playerId)

type cardGameService struct {
	pb.UnimplementedCardGameServiceServer
	mu      sync.Mutex                // Mutex for all data below
	players map[string]*playerSession // Keyed by playerId
	games   map[string]game.Game      // Keyed by gameId
}

type activityReport = *pb.GameActivityResponse

type playerSession struct {
	id     string
	name   string
	gameId string
	ch     chan activityReport
	// lastActivityTimestamp - for tracking dead sessions.
}

func (*cardGameService) Ping(ctx context.Context, request *pb.PingRequest) (*pb.PingResponse, error) {
	log.Printf("Got ping %s", request.GetMessage())
	return &pb.PingResponse{Message: "Pong"}, nil
}

func (s *cardGameService) newPlayerId() string {
	for {
		id := fmt.Sprintf("p%04d", rand.Int31n(10000))
		// Ensure no collision with existing player id.
		if _, found := s.players[id]; !found {
			return id
		}
	}
}
func (s *cardGameService) addPlayer(name string) string {
	playerId := s.newPlayerId()
	sess := &playerSession{
		id:   playerId,
		name: name,
	}
	s.players[playerId] = sess
	return playerId
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
func (s *cardGameService) addGame() game.Game {
	gameId := s.newGameId()
	g := hearts.NewGame(gameId)
	s.games[gameId] = g
	return g
}

func (s *cardGameService) removePlayer(playerId string) error {
	log.Printf("Removing player %s\n", playerId)
	player, ok := s.players[playerId]
	if !ok {
		return fmt.Errorf("can't find player %s", playerId)
	}
	if game, found := s.games[player.gameId]; found {
		err := game.RemovePlayer(playerId)
		if err != nil {
			// Can't remove player, abort game
			game.Abort()
			s.ReportGameAborted()
			s.scheduleRemoveGame(game.Id())
		} else {
			player.gameId = ""
			s.ReportPlayerLeft(playerId, game.Id())
			// How to stop listener here.
		}
	}
	return nil
}
func (s *cardGameService) scheduleRemoveGame(gameId string) {
	// Clean this game up after folks have had a chance to check final state.
	timer := time.NewTimer(20 * time.Second)
	go func() {
		<-timer.C
		s.removeGame(gameId)
	}()
}
func (s *cardGameService) removeGame(gameId string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Printf("Deleting game %s\n", gameId)
	delete(s.games, gameId)
	var playersToDelete []string
	for playerId, player := range s.players {
		if player.gameId == gameId {
			playersToDelete = append(playersToDelete, playerId)
		}
	}
	for _, playerId := range playersToDelete {
		delete(s.players, playerId)
	}
}

func (s *cardGameService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	playerId := s.addPlayer(req.GetName())
	return &pb.RegisterResponse{PlayerId: playerId}, nil
}

func (s *cardGameService) ListGames(ctx context.Context, req *pb.ListGamesRequest) (*pb.ListGamesResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	filter := makeGameFilter(req.GetPhase())
	var games []*pb.ListGamesResponse_GameSummary
	for _, g := range s.games {
		if filter(g) {
			games = append(games, &pb.ListGamesResponse_GameSummary{
				Id:          g.Id(),
				Phase:       g.Phase().ToProto(),
				PlayerNames: g.PlayerNames(),
			})
		}
	}
	return &pb.ListGamesResponse{
		Games: games,
	}, nil
}

// Builds filter that accepts only games with one of the given phases (or any phase if no phases listed).
func makeGameFilter(phases []pb.GameState_Phase) func(game.Game) bool {
	return func(g game.Game) bool {
		if len(phases) == 0 {
			return true
		}
		for _, ph := range phases {
			if g.Phase().ToProto() == ph {
				return true
			}
		}
		return false
	}
}

func (s *cardGameService) JoinGame(ctx context.Context, req *pb.JoinGameRequest) (*pb.JoinGameResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	playerId := req.GetPlayerId()
	gameId := req.GetGameId()
	player, ok := s.players[playerId]
	if !ok {
		return nil, fmt.Errorf("playerId %s not found", playerId)
	}
	var g game.Game
	if gameId == "" {
		g = s.addGame()
		gameId = g.Id()
	} else {
		g, ok = s.games[gameId]
		if !ok {
			return nil, fmt.Errorf("game %s not found", gameId)
		}
	}
	player.gameId = gameId
	if req.GetMode() == pb.JoinGameRequest_AsPlayer {
		if !g.AcceptingMorePlayers() {
			return nil, fmt.Errorf("game %s is full", gameId)
		}
		g.AddPlayer(player.name, player.id)
		s.ReportPlayerJoined(player.name, gameId)
		if g.StartIfReady() {
			s.ReportGameStarted()
			s.ReportYourTurn(g.NextPlayerId())
		}
	}
	return &pb.JoinGameResponse{GameId: gameId}, nil
}

func (s *cardGameService) LeaveGame(ctx context.Context, req *pb.LeaveGameRequest) (*pb.LeaveGameResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	playerId := req.GetPlayerId()
	err := s.removePlayer(playerId)
	if err != nil {
		return nil, err
	}
	return &pb.LeaveGameResponse{}, nil
}

func (s *cardGameService) GetGameState(ctx context.Context, req *pb.GameStateRequest) (*pb.GameState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	gameId, playerId, err := func() (string, string, error) {
		switch r := req.Type.(type) {
		case *pb.GameStateRequest_GameId:
			return r.GameId, "", nil
		case *pb.GameStateRequest_PlayerId:
			playerId := r.PlayerId
			player, found := s.players[playerId]
			if !found {
				return "", "", fmt.Errorf("no player found for playerId %s", playerId)
			}
			return player.gameId, playerId, nil
		default:
			return "", "", fmt.Errorf("no value found for GameStateRequest.Type")
		}
	}()
	if err != nil {
		return nil, err
	}
	g, found := s.games[gameId]
	if !found {
		return nil, fmt.Errorf("no game found for playerId %s : %s", playerId, gameId)
	}
	return g.GetGameState(playerId)
}

func (s *cardGameService) PlayerAction(ctx context.Context, req *pb.PlayerActionRequest) (*pb.Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch r := req.Type.(type) {
	case *pb.PlayerActionRequest_PlayCard:
		playerId := req.GetPlayerId()
		card, _ := cards.ParseCard(r.PlayCard.GetCard())
		err := s.handlePlayCard(playerId, card)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("PlayerActionRequest has unexpected type %T", r)
	}
	return &pb.Status{Code: 0}, nil
}

func (s *cardGameService) handlePlayCard(playerId string, card cards.Card) error {
	player, found := s.players[playerId]
	if !found {
		return fmt.Errorf("playerId %s not found", playerId)
	}
	g, found := s.games[player.gameId]
	if !found {
		return fmt.Errorf("no game %s found for playerId %s", player.gameId, playerId)
	}
	err := g.HandlePlayCard(playerId, card, s)
	if err != nil {
		return err
	}
	if g.Phase() != game.Completed {
		s.ReportYourTurn(g.NextPlayerId())
	} else {
		s.ReportGameFinished()
		s.scheduleRemoveGame(g.Id())
	}
	return nil
}

func (s *cardGameService) ListenForGameActivity(req *pb.GameActivityRequest, resp pb.CardGameService_ListenForGameActivityServer) error {
	s.mu.Lock()
	playerId := req.GetPlayerId()
	log.Printf("ListenForGameActivity from %s - %s\n", playerId, s.players[playerId].name)
	ch := make(chan activityReport, 4)
	s.players[playerId].ch = ch
	s.mu.Unlock()
	err := reportActivityToListener(ch, resp)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.players[playerId].ch = nil
	close(ch)
	return err
}

// Broadcasts message to all clients.
func (s *cardGameService) BroadcastMessage(msg string) {
	s.reportActivityToAll(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_BroadcastMsg{BroadcastMsg: msg},
		})
}
func (s *cardGameService) ReportPlayerJoined(name string, gameId string) {
	s.reportActivityToAll(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_PlayerJoined_{
				PlayerJoined: &pb.GameActivityResponse_PlayerJoined{Name: name, GameId: gameId},
			},
		})
}
func (s *cardGameService) ReportPlayerLeft(name string, gameId string) {
	s.reportActivityToAll(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_PlayerLeft_{
				PlayerLeft: &pb.GameActivityResponse_PlayerLeft{Name: name, GameId: gameId},
			},
		})
}
func (s *cardGameService) ReportGameStarted() {
	s.reportActivityToAll(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_GameStarted_{},
		})
}
func (s *cardGameService) ReportCardPlayed() {
	s.reportActivityToAll(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_CardPlayed_{},
		})
}
func (s *cardGameService) ReportTrickCompleted(trick cards.Cards, trickWinnerId, trickWinnerName string) {
	s.reportActivityToAll(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_TrickCompleted_{
				TrickCompleted: &pb.GameActivityResponse_TrickCompleted{
					Trick:           trick.Strings(),
					TrickWinnerId:   trickWinnerId,
					TrickWinnerName: trickWinnerName,
				},
			},
		})
}
func (s *cardGameService) ReportGameFinished() {
	s.reportActivityToAll(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_GameFinished_{},
		})
}
func (s *cardGameService) ReportGameAborted() {
	s.reportActivityToAll(
		&pb.GameActivityResponse{
			Type: &pb.GameActivityResponse_GameAborted_{},
		})
}
func (s *cardGameService) reportActivityToAll(activity activityReport) {
	for _, p := range s.players {
		if p.ch != nil {
			p.ch <- activity
		}
	}
}
func (s *cardGameService) ReportYourTurn(pId string) {
	sess, ok := s.players[pId]
	if !ok {
		log.Printf("No such playerId %s", pId)
		return
	}
	yourTurn := &pb.GameActivityResponse{
		Type: &pb.GameActivityResponse_YourTurn_{},
	}
	sess.ch <- yourTurn
}

func reportActivityToListener(activityCh chan activityReport, listener pb.CardGameService_ListenForGameActivityServer) error {
	for {
		select {
		case activity := <-activityCh:
			err := listener.Send(activity)
			if err != nil {
				return err
			}
			shouldStopListening := func() bool {
				switch activity.Type.(type) {
				case *pb.GameActivityResponse_GameFinished_,
					*pb.GameActivityResponse_GameAborted_:
					return true
				default:
					return false
				}
			}()
			if shouldStopListening {
				// Game is over. Close this reporting request.
				return nil
			}
		case <-listener.Context().Done():
			return listener.Context().Err()
		}
	}
}
