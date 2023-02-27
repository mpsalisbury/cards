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
	"golang.org/x/exp/maps"
)

func NewCardGameService() pb.CardGameServiceServer {
	cgs := &cardGameService{
		players: make(map[string]*playerSession),
		games:   make(map[string]*gameSession),
	}
	cgs.startGarbageCollector()
	return cgs
}

type cardGameService struct {
	pb.UnsafeCardGameServiceServer
	mu      sync.Mutex                // Mutex for all data below
	players map[string]*playerSession // Keyed by sessionId
	games   map[string]*gameSession   // Keyed by gameId
}

type gameActivityReport = pb.GameActivity_Type
type registryActivityReport = pb.RegistryActivity_Type

type gameSession struct {
	game      game.Game
	reportChs map[string]chan gameActivityReport // Keyed by sessionId
}

type playerSession struct {
	id       string
	name     string
	gameIds  map[string]struct{} // All gameIds this player is participating in
	reportCh chan registryActivityReport
}

func (s *cardGameService) startGarbageCollector() {
	// Collection frequency.
	ticker := time.NewTicker(time.Minute)
	go func() {
		for t := range ticker.C {
			s.mu.Lock()
			for _, g := range s.games {
				if t.Sub(g.game.GetLastActivityTime()) > time.Hour {
					log.Printf("Removing game %s due to inactivity", g.game.Id())
					s.scheduleDeleteGame(g.game.Id(), time.Second)
				}
			}
			s.mu.Unlock()
		}
	}()
}

func (*cardGameService) Ping(ctx context.Context, request *pb.PingRequest) (*pb.PingResponse, error) {
	log.Printf("Got ping %s", request.GetMessage())
	return &pb.PingResponse{Message: "Pong"}, nil
}

func (s *cardGameService) newSessionId() string {
	for {
		id := fmt.Sprintf("p%04d", rand.Int31n(10000))
		// Ensure no collision with existing player id.
		if _, found := s.players[id]; !found {
			return id
		}
	}
}
func (s *cardGameService) addPlayer(name string) *playerSession {
	sessionId := s.newSessionId()
	sess := &playerSession{
		id:       sessionId,
		name:     name,
		gameIds:  make(map[string]struct{}),
		reportCh: make(chan registryActivityReport, 4),
	}
	s.players[sessionId] = sess
	log.Printf("Added player %s\n", sessionId)
	return sess
}
func (s *cardGameService) deletePlayer(playerId string) error {
	p, ok := s.players[playerId]
	if !ok {
		return fmt.Errorf("can't find player %s", playerId)
	}
	delete(s.players, playerId)
	close(p.reportCh)
	for gameId := range p.gameIds {
		s.removePlayerFromGame(playerId, gameId)
	}
	log.Printf("Deleted player %s\n", playerId)
	return nil
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
func (s *cardGameService) addGame() *gameSession {
	gameId := s.newGameId()
	g := hearts.NewGame(gameId)
	gs := &gameSession{
		game:      g,
		reportChs: make(map[string]chan gameActivityReport),
	}
	s.games[gameId] = gs
	s.reportGameCreated(gameId)
	return gs
}

func (s *cardGameService) removePlayerFromGame(playerId, gameId string) error {
	//	log.Printf("Removing player %s from game %s\n", playerId, gameId)
	player, ok := s.players[playerId]
	if !ok {
		return fmt.Errorf("can't find player %s", playerId)
	}
	if gs, found := s.games[gameId]; found {
		g := gs.game
		switch g.Phase() {
		case game.Preparing:
			err := g.RemovePlayer(playerId)
			if err != nil {
				// Can't remove player, abort game
				s.abortGame(g)
			} else {
				delete(player.gameIds, gameId)
				delete(gs.reportChs, playerId)
				s.ReportPlayerLeft(g, player.name)
				// How to stop listener here.
			}
		case game.Playing:
			s.abortGame(g)
		case game.Completed, game.Aborted:
			// Don't bother to remove player from completed or aborted game.
		}
	}
	return nil
}
func (s *cardGameService) abortGame(g game.Game) {
	g.Abort()
	s.ReportGameAborted(g)
	s.scheduleDeleteGame(g.Id(), time.Second)
}

func (s *cardGameService) scheduleDeleteGame(gameId string, when time.Duration) {
	timer := time.NewTimer(when)
	go func() {
		<-timer.C
		s.deleteGame(gameId)
	}()
}
func (s *cardGameService) deleteGame(gameId string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, found := s.games[gameId]
	if !found {
		log.Printf("deleteGame: no such gameId %s", gameId)
		return
	}
	// Disconnect players from deleted game.
	for playerId, _ := range g.reportChs {
		if player, ok := s.players[playerId]; ok {
			delete(player.gameIds, gameId)
		}
	}
	log.Printf("Deleted game %s\n", gameId)
	delete(s.games, gameId)
	s.reportGameDeleted(gameId)
}

func (s *cardGameService) Register(req *pb.RegisterRequest, resp pb.CardGameService_RegisterServer) error {
	s.mu.Lock()
	p := s.addPlayer(req.GetName())
	ch := p.reportCh
	ch <- makeReportSessionCreatedActivity(p.id)
	ch <- makeFullGamesListActivity(maps.Keys(s.games))
	s.mu.Unlock()
	err := reportRegistryActivityToListener(p.reportCh, resp)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deletePlayer(p.id)
	return err
}

func makeReportSessionCreatedActivity(sessionId string) registryActivityReport {
	return &pb.RegistryActivity_SessionCreated_{
		SessionCreated: &pb.RegistryActivity_SessionCreated{
			SessionId: sessionId,
		},
	}
}
func makeFullGamesListActivity(gameIds []string) registryActivityReport {
	return &pb.RegistryActivity_FullGamesList_{
		FullGamesList: &pb.RegistryActivity_FullGamesList{
			GameIds: gameIds,
		},
	}
}

func (s *cardGameService) ListGames(ctx context.Context, req *pb.ListGamesRequest) (*pb.ListGamesResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	filter := makeGameFilter(req.GetPhase())
	var games []*pb.ListGamesResponse_GameSummary
	for _, gs := range s.games {
		if filter(gs.game) {
			games = append(games, &pb.ListGamesResponse_GameSummary{
				Id:          gs.game.Id(),
				Phase:       gs.game.Phase().ToProto(),
				PlayerNames: gs.game.PlayerNames(),
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

func (s *cardGameService) CreateGame(ctx context.Context, req *pb.CreateGameRequest) (*pb.CreateGameResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	gs := s.addGame()
	return &pb.CreateGameResponse{
		GameId: gs.game.Id(),
	}, nil
}
func (s *cardGameService) JoinGame(req *pb.JoinGameRequest, resp pb.CardGameService_JoinGameServer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessionId := req.GetSessionId()
	gameId := req.GetGameId()
	player, ok := s.players[sessionId]
	if !ok {
		return fmt.Errorf("sessionId %s not found", sessionId)
	}
	gs, ok := s.games[gameId]
	if !ok {
		return fmt.Errorf("game %s not found", gameId)
	}
	g := gs.game
	player.gameIds[gameId] = struct{}{}
	if !g.AcceptingMorePlayers() {
		return fmt.Errorf("game %s is full", gameId)
	}
	g.AddPlayer(player.name, player.id)
	s.ReportPlayerJoined(g, player.name)
	if g.IsEnoughPlayersToStart() {
		go s.triggerStartWhenPlayersReady(gs)
	}
	log.Printf("Player %s joined game %s", sessionId, gameId)

	ch := make(chan gameActivityReport, 4)
	gs.reportChs[sessionId] = ch
	s.mu.Unlock()
	reportGameActivityToListener(gameId, ch, resp)
	s.mu.Lock()
	delete(gs.reportChs, sessionId)
	close(ch)
	// TODO: move this to when player listening activity channel is closed, not game.
	s.handleLeaveGame(sessionId, gameId)
	return nil
}

func (s *cardGameService) ObserveGame(req *pb.ObserveGameRequest, resp pb.CardGameService_ObserveGameServer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessionId := req.GetSessionId()
	gameId := req.GetGameId()
	player, ok := s.players[sessionId]
	if !ok {
		return fmt.Errorf("sessionId %s not found", sessionId)
	}
	var gs *gameSession
	gs, ok = s.games[gameId]
	if !ok {
		return fmt.Errorf("game %s not found", gameId)
	}
	player.gameIds[gameId] = struct{}{}
	log.Printf("Player %s observing game %s", sessionId, gameId)

	// TODO: deduplicate this
	ch := make(chan gameActivityReport, 4)
	gs.reportChs[sessionId] = ch
	s.mu.Unlock()
	reportGameActivityToListener(gameId, ch, resp)
	s.mu.Lock()
	delete(gs.reportChs, sessionId)
	close(ch)
	// TODO: move this to when player listening activity channel is closed, not game.
	s.handleLeaveGame(sessionId, gameId)
	return nil
}

func (s *cardGameService) triggerStartWhenPlayersReady(gs *gameSession) {
	g := gs.game
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	// Allow one minute for all players to be ready, or we'll abort this game.
	doneTimer := time.NewTimer(time.Minute)
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			// log.Printf("Checking if players are ready for game %s\n", g.Id())
			if len(gs.game.UnconfirmedPlayerIds()) == 0 {
				log.Printf("Starting game %s\n", g.Id())
				gs.game.StartGame()
				s.ReportGameStarted(g)
				s.ReportNextTurn(g)
				s.mu.Unlock()
				return
			}
			// Tell everyone again in case they weren't listening before.
			s.ReportGameReadyToStart(g)
			s.mu.Unlock()
		case <-doneTimer.C:
			s.mu.Lock()
			log.Printf("Game %s players not ready. Aborting.", gs.game.Id())
			gs.game.Abort()
			s.ReportGameAborted(g)
			s.mu.Unlock()
			return
		}
	}
}

func (s *cardGameService) GameAction(ctx context.Context, req *pb.GameActionRequest) (*pb.Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessionId := req.GetSessionId()
	gameId := req.GetGameId()
	var err error
	switch r := req.Type.(type) {
	case *pb.GameActionRequest_ReadyToStartGame:
		err = s.handleReadyToStartGame(sessionId, gameId)
	case *pb.GameActionRequest_LeaveGame:
		err = s.handleLeaveGame(sessionId, gameId)
	case *pb.GameActionRequest_PlayCard:
		card, _ := cards.ParseCard(r.PlayCard.GetCard())
		err = s.handlePlayCard(sessionId, gameId, card)
	default:
		return nil, fmt.Errorf("GameActionRequest has unexpected type %T", r)
	}
	if err != nil {
		return nil, err
	}
	return &pb.Status{Code: 0}, nil
}

func (s *cardGameService) handleReadyToStartGame(sessionId, gameId string) error {
	g, found := s.games[gameId]
	if !found {
		return fmt.Errorf("no game %s found", gameId)
	}
	return g.game.ConfirmPlayerReadyToStart(sessionId)
}

func (s *cardGameService) handleLeaveGame(sessionId, gameId string) error {
	return s.removePlayerFromGame(sessionId, gameId)
}

func (s *cardGameService) handlePlayCard(sessionId, gameId string, card cards.Card) error {
	gs, found := s.games[gameId]
	if !found {
		return fmt.Errorf("no game %s found", gameId)
	}
	g := gs.game
	err := g.HandlePlayCard(sessionId, card, s)
	if err != nil {
		return err
	}
	if g.Phase() != game.Completed {
		s.ReportNextTurn(g)
	} else {
		log.Printf("Game %s complete\n", g.Id())
		s.ReportGameFinished(g)
		// Clean this game up after folks have had a chance to check final state.
		s.scheduleDeleteGame(g.Id(), 20*time.Second)
	}
	return nil
}

func (s *cardGameService) GetGameState(ctx context.Context, req *pb.GameStateRequest) (*pb.GameState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessionId := req.GetSessionId()
	gameId := req.GetGameId()
	gs, found := s.games[gameId]
	if !found {
		return nil, fmt.Errorf("no game %s found", gameId)
	}
	return gs.game.GetGameState(sessionId)
}

// Broadcasts message to all clients.
func (s *cardGameService) BroadcastMessage(g game.Game, msg string) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivity_BroadcastMsg{BroadcastMsg: msg})
}
func (s *cardGameService) ReportPlayerJoined(g game.Game, name string) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivity_PlayerJoined_{
			PlayerJoined: &pb.GameActivity_PlayerJoined{Name: name},
		})
}
func (s *cardGameService) ReportPlayerLeft(g game.Game, name string) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivity_PlayerLeft_{
			PlayerLeft: &pb.GameActivity_PlayerLeft{Name: name},
		})
}
func (s *cardGameService) ReportGameReadyToStart(g game.Game) {
	gs, ok := s.games[g.Id()]
	if !ok {
		log.Printf("ReportGameReadyToStart: no such gameId %s", g.Id())
		return
	}
	activity := &pb.GameActivity_GameReadyToStart_{}
	for _, pid := range g.UnconfirmedPlayerIds() {
		if ch, ok := gs.reportChs[pid]; ok {
			ch <- activity
		}
	}
}
func (s *cardGameService) ReportGameStarted(g game.Game) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivity_GameStarted_{})
}
func (s *cardGameService) ReportCardPlayed(g game.Game) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivity_CardPlayed_{})
}
func (s *cardGameService) ReportTrickCompleted(g game.Game, trick cards.Cards, winningCard cards.Card, winnerId, winnerName string) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivity_TrickCompleted_{
			TrickCompleted: &pb.GameActivity_TrickCompleted{
				Trick:       trick.Strings(),
				WinningCard: winningCard.String(),
				WinnerId:    winnerId,
				WinnerName:  winnerName,
			},
		})
}
func (s *cardGameService) reportGameCreated(gameId string) {
	s.reportRegistryActivityToAll(
		&pb.RegistryActivity_GameCreated_{
			GameCreated: &pb.RegistryActivity_GameCreated{
				GameId: gameId,
			},
		})
}
func (s *cardGameService) reportGameDeleted(gameId string) {
	s.reportRegistryActivityToAll(
		&pb.RegistryActivity_GameDeleted_{
			GameDeleted: &pb.RegistryActivity_GameDeleted{
				GameId: gameId,
			},
		})
}
func (s *cardGameService) ReportGameFinished(g game.Game) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivity_GameFinished_{})
}
func (s *cardGameService) ReportGameAborted(g game.Game) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivity_GameAborted_{})
}
func (s *cardGameService) ReportNextTurn(g game.Game) {
	gs, ok := s.games[g.Id()]
	if !ok {
		log.Printf("ReportNextTurn: no such gameId %s", g.Id())
		return
	}
	pId := g.NextPlayerId()
	ch, ok := gs.reportChs[pId]
	if !ok {
		log.Printf("No such playerId %s", pId)
		return
	}
	yourTurn := &pb.GameActivity_YourTurn_{}
	ch <- yourTurn
}
func (s *cardGameService) reportGameActivityToAll(g game.Game, activity gameActivityReport) {
	if gs, ok := s.games[g.Id()]; ok {
		for _, ch := range gs.reportChs {
			ch <- activity
		}
	}
}
func (s *cardGameService) reportRegistryActivityToAll(activity registryActivityReport) {
	for _, p := range s.players {
		p.reportCh <- activity
	}
}

// Handles both JoinGame and ObserveGame streams.
func reportGameActivityToListener(gameId string, ch chan gameActivityReport, listener pb.CardGameService_ObserveGameServer) error {
	for {
		select {
		case gameActivityType := <-ch:
			activity := &pb.GameActivity{GameId: gameId, Type: gameActivityType}
			err := listener.Send(activity)
			if err != nil {
				return err
			}
			shouldStopListening := func() bool {
				switch gameActivityType.(type) {
				case *pb.GameActivity_GameFinished_,
					*pb.GameActivity_GameAborted_:
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

func reportRegistryActivityToListener(ch chan registryActivityReport, listener pb.CardGameService_RegisterServer) error {
	for {
		select {
		case registryActivityType := <-ch:
			activity := &pb.RegistryActivity{Type: registryActivityType}
			err := listener.Send(activity)
			if err != nil {
				return err
			}
		case <-listener.Context().Done():
			return listener.Context().Err()
		}
	}
}
