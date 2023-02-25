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
	cgs := &cardGameService{
		players:           make(map[string]*playerSession),
		games:             make(map[string]game.Game),
		registryListeners: make(map[string]chan registryActivityReport),
	}
	cgs.startGarbageCollector()
	return cgs
}

type cardGameService struct {
	pb.UnimplementedCardGameServiceServer
	mu                sync.Mutex                             // Mutex for all data below
	players           map[string]*playerSession              // Keyed by playerId
	games             map[string]game.Game                   // Keyed by gameId
	registryListeners map[string]chan registryActivityReport // Keyed by registryListenerId
}

type gameActivityReport = pb.GameActivityResponse_Type

type playerSession struct {
	id     string
	name   string
	gameId string
	ch     chan gameActivityReport
}

func (s *cardGameService) startGarbageCollector() {
	// Collection frequency.
	ticker := time.NewTicker(time.Minute)
	go func() {
		for t := range ticker.C {
			s.mu.Lock()
			for _, g := range s.games {
				if t.Sub(g.GetLastActivityTime()) > time.Hour {
					log.Printf("Removing game %s due to inactivity", g.Id())
					s.scheduleRemoveGame(g.Id(), time.Second)
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
	log.Printf("Adding player %s\n", playerId)
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
	s.ReportGameCreated(gameId)
	return g
}

func (s *cardGameService) removePlayer(playerId string) error {
	log.Printf("Removing player %s\n", playerId)
	player, ok := s.players[playerId]
	if !ok {
		return fmt.Errorf("can't find player %s", playerId)
	}
	if g, found := s.games[player.gameId]; found {
		switch g.Phase() {
		case game.Preparing:
			err := g.RemovePlayer(playerId)
			if err != nil {
				// Can't remove player, abort game
				s.abortGame(g)
			} else {
				player.gameId = ""
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
	s.scheduleRemoveGame(g.Id(), time.Second)
}

func (s *cardGameService) scheduleRemoveGame(gameId string, when time.Duration) {
	timer := time.NewTimer(when)
	go func() {
		<-timer.C
		s.removeGame(gameId)
	}()
}
func (s *cardGameService) removeGame(gameId string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Printf("Deleting game %s\n", gameId)
	delete(s.games, gameId)
	s.ReportGameDeleted(gameId)
	//	var playersToDelete []string
	// Disconnect players from deleted game.
	for _, player := range s.players {
		if player.gameId == gameId {
			player.gameId = ""
			//playersToDelete = append(playersToDelete, playerId)
		}
	}
	/*
		for _, playerId := range playersToDelete {
			delete(s.players, playerId)
		}
	*/
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
		s.ReportPlayerJoined(g, player.name)
		if g.IsEnoughPlayersToStart() {
			go s.triggerStartWhenPlayersReady(g)
		}
		log.Printf("Player %s joined game %s", playerId, gameId)
	}
	if req.GetMode() == pb.JoinGameRequest_AsObserver {
		g.AddObserver(player.name, player.id)
		log.Printf("Player %s observing game %s", playerId, gameId)
	}
	return &pb.JoinGameResponse{GameId: gameId}, nil
}

func (s *cardGameService) triggerStartWhenPlayersReady(g game.Game) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	// Allow one minute for all players to be ready, or we'll abort this game.
	doneTimer := time.NewTimer(time.Minute)
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			// log.Printf("Checking if players are ready for game %s\n", g.Id())
			if len(g.UnconfirmedPlayerIds()) == 0 {
				log.Printf("Starting game %s\n", g.Id())
				g.StartGame()
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
			log.Printf("Game %s players not ready. Aborting.", g.Id())
			g.Abort()
			s.ReportGameAborted(g)
			s.mu.Unlock()
			return
		}
	}
}

func (s *cardGameService) GameAction(ctx context.Context, req *pb.GameActionRequest) (*pb.Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	playerId := req.GetPlayerId()
	var err error
	switch r := req.Type.(type) {
	case *pb.GameActionRequest_ReadyToStartGame:
		err = s.handleReadyToStartGame(playerId)
	case *pb.GameActionRequest_LeaveGame:
		err = s.handleLeaveGame(playerId)
	case *pb.GameActionRequest_PlayCard:
		card, _ := cards.ParseCard(r.PlayCard.GetCard())
		err = s.handlePlayCard(playerId, card)
	default:
		return nil, fmt.Errorf("GameActionRequest has unexpected type %T", r)
	}
	if err != nil {
		return nil, err
	}
	return &pb.Status{Code: 0}, nil
}

func (s *cardGameService) handleReadyToStartGame(playerId string) error {
	player, found := s.players[playerId]
	if !found {
		return fmt.Errorf("playerId %s not found", playerId)
	}
	g, found := s.games[player.gameId]
	if !found {
		return fmt.Errorf("no game %s found for playerId %s", player.gameId, playerId)
	}
	return g.ConfirmPlayerReadyToStart(playerId)
}

func (s *cardGameService) handleLeaveGame(playerId string) error {
	return s.removePlayer(playerId)
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
		s.ReportNextTurn(g)
	} else {
		log.Printf("Game %s complete\n", g.Id())
		s.ReportGameFinished(g)
		// Clean this game up after folks have had a chance to check final state.
		s.scheduleRemoveGame(g.Id(), 20*time.Second)
	}
	return nil
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

func (s *cardGameService) ListenForGameActivity(req *pb.GameActivityRequest, resp pb.CardGameService_ListenForGameActivityServer) error {
	s.mu.Lock()
	playerId := req.GetPlayerId()
	//log.Printf("ListenForGameActivity from %s - %s\n", playerId, s.players[playerId].name)
	ch := make(chan gameActivityReport, 4)
	s.players[playerId].ch = ch
	s.mu.Unlock()
	err := reportGameActivityToListener(ch, resp)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.players[playerId].ch = nil
	close(ch)
	// TODO: move this to when player listening activity channel is closed, not game.
	s.handleLeaveGame(playerId)
	return err
}

// Broadcasts message to all clients.
func (s *cardGameService) BroadcastMessage(g game.Game, msg string) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivityResponse_BroadcastMsg{BroadcastMsg: msg})
}
func (s *cardGameService) ReportPlayerJoined(g game.Game, name string) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivityResponse_PlayerJoined_{
			PlayerJoined: &pb.GameActivityResponse_PlayerJoined{Name: name, GameId: g.Id()},
		})
}
func (s *cardGameService) ReportPlayerLeft(g game.Game, name string) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivityResponse_PlayerLeft_{
			PlayerLeft: &pb.GameActivityResponse_PlayerLeft{Name: name, GameId: g.Id()},
		})
}
func (s *cardGameService) ReportGameReadyToStart(g game.Game) {
	activity := &pb.GameActivityResponse_GameReadyToStart_{}
	for _, pid := range g.UnconfirmedPlayerIds() {
		p, ok := s.players[pid]
		if ok && p.ch != nil {
			p.ch <- activity
		}
	}
}
func (s *cardGameService) ReportGameStarted(g game.Game) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivityResponse_GameStarted_{})
}
func (s *cardGameService) ReportCardPlayed(g game.Game) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivityResponse_CardPlayed_{})
}
func (s *cardGameService) ReportTrickCompleted(g game.Game, trick cards.Cards, trickWinnerId, trickWinnerName string) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivityResponse_TrickCompleted_{
			TrickCompleted: &pb.GameActivityResponse_TrickCompleted{
				Trick:           trick.Strings(),
				TrickWinnerId:   trickWinnerId,
				TrickWinnerName: trickWinnerName,
			},
		})
}
func (s *cardGameService) ReportGameCreated(gameId string) {
	s.reportRegistryActivityToAll(
		&pb.RegistryActivityResponse_GameCreated_{
			GameCreated: &pb.RegistryActivityResponse_GameCreated{
				GameId: gameId,
			},
		})
}
func (s *cardGameService) ReportGameDeleted(gameId string) {
	s.reportRegistryActivityToAll(
		&pb.RegistryActivityResponse_GameDeleted_{
			GameDeleted: &pb.RegistryActivityResponse_GameDeleted{
				GameId: gameId,
			},
		})
}
func (s *cardGameService) ReportGameFinished(g game.Game) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivityResponse_GameFinished_{})
}
func (s *cardGameService) ReportGameAborted(g game.Game) {
	s.reportGameActivityToAll(
		g,
		&pb.GameActivityResponse_GameAborted_{})
}
func (s *cardGameService) ReportNextTurn(g game.Game) {
	pId := g.NextPlayerId()
	sess, ok := s.players[pId]
	if !ok {
		log.Printf("No such playerId %s", pId)
		return
	}
	yourTurn := &pb.GameActivityResponse_YourTurn_{}
	sess.ch <- yourTurn
}
func (s *cardGameService) reportGameActivityToAll(g game.Game, activity gameActivityReport) {
	for _, pid := range g.ListenerIds() {
		p, ok := s.players[pid]
		if ok && p.ch != nil {
			p.ch <- activity
		}
	}
}
func (s *cardGameService) reportRegistryActivityToAll(activity registryActivityReport) {
	for _, ch := range s.registryListeners {
		ch <- activity
	}
}

func reportGameActivityToListener(ch chan gameActivityReport, listener pb.CardGameService_ListenForGameActivityServer) error {
	for {
		select {
		case gameActivityType := <-ch:
			activity := &pb.GameActivityResponse{Type: gameActivityType}
			err := listener.Send(activity)
			if err != nil {
				return err
			}
			shouldStopListening := func() bool {
				switch gameActivityType.(type) {
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

type registryActivityReport = pb.RegistryActivityResponse_Type

func (s *cardGameService) ListenForRegistryActivity(req *pb.RegistryActivityRequest, resp pb.CardGameService_ListenForRegistryActivityServer) error {
	s.mu.Lock()
	listenerKey := guid()
	ch := make(chan registryActivityReport, 4)
	s.registryListeners[listenerKey] = ch
	log.Printf("Adding registry listener %s", listenerKey)
	s.mu.Unlock()
	go s.reportFullGameList(ch)
	err := reportRegistryActivityToListener(ch, resp)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.registryListeners, listenerKey)
	log.Printf("Dropping registry listener %s", listenerKey)
	close(ch)
	return err
}
func (s *cardGameService) reportFullGameList(ch chan registryActivityReport) {
	var gameIds []string
	for gid, _ := range s.games {
		gameIds = append(gameIds, gid)
	}
	activity := &pb.RegistryActivityResponse_FullGamesList_{
		FullGamesList: &pb.RegistryActivityResponse_FullGamesList{
			GameIds: gameIds,
		},
	}
	ch <- activity
}

func reportRegistryActivityToListener(ch chan registryActivityReport, listener pb.CardGameService_ListenForRegistryActivityServer) error {
	for {
		select {
		case registryActivityType := <-ch:
			activity := &pb.RegistryActivityResponse{Type: registryActivityType}
			err := listener.Send(activity)
			if err != nil {
				return err
			}
		case <-listener.Context().Done():
			return listener.Context().Err()
		}
	}
}

func guid() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("%x", b)
}
