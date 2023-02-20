package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/mpsalisbury/cards/pkg/cards"
	pb "github.com/mpsalisbury/cards/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const hostedServer = "api.cards.salisburyclan.com:443"
const localServer = "localhost:50051"

//const rawHostedServer = "cards-api-5g5wrbokbq-uw.a.run.app:443"

type ServerType uint8

const (
	LocalServer ServerType = iota
	HostedServer

//	InProcessServer
)

var configs = map[ServerType]struct {
	serverAddr string
	secure     bool
}{
	LocalServer:  {localServer, false},
	HostedServer: {hostedServer, true},
}

func Connect(stype ServerType, verbose bool) (Connection, error) {
	conn, client, err := createClient(stype)
	if err != nil {
		return nil, err
	}
	return &connection{conn: conn, client: client, verbose: verbose}, nil
}

func createClient(stype ServerType) (*grpc.ClientConn, pb.CardGameServiceClient, error) {
	switch stype {
	case LocalServer, HostedServer:
		return createExternalServer(stype)
		//	case InProcessServer:
		//		return createInProcessServer()
	}
	return nil, nil, fmt.Errorf("server type %v not supported", stype)
}

func createExternalServer(stype ServerType) (*grpc.ClientConn, pb.CardGameServiceClient, error) {
	config := configs[stype]

	cred := func() credentials.TransportCredentials {
		if config.secure {
			return credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: false,
			})
		}
		return insecure.NewCredentials()
	}()
	conn, err := grpc.Dial(config.serverAddr, grpc.WithTransportCredentials(cred))
	if err != nil {
		return nil, nil, err
	}
	client := pb.NewCardGameServiceClient(conn)
	return conn, client, nil
}

// func createInProcessServer() (*grpc.ClientConn, pb.CardGameServiceClient, error) {
// 	return nil, newInProcessServer(), nil
// }

type Connection interface {
	Close()
	Register(ctx context.Context, name string, callbacks GameCallbacks) (Session, error)
	ListGames(ctx context.Context, phase ...GamePhase) ([]GameSummary, error)
	GetGameState(ctx context.Context, gameId string) (GameState, error)
}
type Session interface {
	GetPlayerId() string
	JoinGameAsPlayer(ctx context.Context, wg *sync.WaitGroup, gameId string) (string, error)
	JoinGameAsObserver(ctx context.Context, wg *sync.WaitGroup, gameId string) (string, error)
	ReadyToStartGame(ctx context.Context) error
	LeaveGame(ctx context.Context) error
	PlayCard(ctx context.Context, card cards.Card) error
	GetGameState(ctx context.Context) (GameState, error)
}

type GameCallbacks interface {
	HandlePlayerJoined(s Session, name string, gameId string) error
	HandlePlayerLeft(s Session, name string, gameId string) error
	HandleGameReadyToStart(Session) error
	HandleGameStarted(Session) error
	HandleCardPlayed(Session /*currentTrick cards.Cards*/) error
	HandleYourTurn(Session) error
	HandleTrickCompleted(s Session, trick cards.Cards, trickWinnerId, trickWinnerName string) error
	HandleGameFinished(Session) error
	HandleGameAborted(Session) error
	HandleConnectionError(s Session, err error)
}
type UnimplementedGameCallbacks struct{}

func (UnimplementedGameCallbacks) HandlePlayerJoined(s Session, name string, gameId string) error {
	return nil
}
func (UnimplementedGameCallbacks) HandlePlayerLeft(s Session, name string, gameId string) error {
	return nil
}
func (UnimplementedGameCallbacks) HandleGameReadyToStart(s Session) error {
	// By default we reply that this player is ready to start playing, since we got the notification.
	return s.ReadyToStartGame(context.Background())
}
func (UnimplementedGameCallbacks) HandleGameStarted(Session) error { return nil }
func (UnimplementedGameCallbacks) HandleCardPlayed(Session) error  { return nil }
func (UnimplementedGameCallbacks) HandleYourTurn(Session) error    { return nil }
func (UnimplementedGameCallbacks) HandleTrickCompleted(
	s Session, trick cards.Cards, trickWinnerId, trickWinnerName string) error {
	return nil
}
func (UnimplementedGameCallbacks) HandleGameFinished(Session) error     { return nil }
func (UnimplementedGameCallbacks) HandleGameAborted(Session) error      { return nil }
func (UnimplementedGameCallbacks) HandleConnectionError(Session, error) {}

type GameState struct {
	Id           string
	Phase        GamePhase
	Players      []PlayerState
	CurrentTrick cards.Cards
	LegalPlays   cards.Cards
}

type GamePhase int8

const (
	Preparing GamePhase = iota
	Playing
	Completed
	Aborted
)

func (gp GamePhase) String() string {
	switch gp {
	case Preparing:
		return "Preparing"
	case Playing:
		return "Playing"
	case Completed:
		return "Completed"
	case Aborted:
		return "Aborted"
	}
	return "unknown"
}

func (gp GamePhase) toProto() pb.GameState_Phase {
	switch gp {
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

func protoToPhase(phase pb.GameState_Phase) GamePhase {
	switch phase {
	case pb.GameState_Preparing:
		return Preparing
	case pb.GameState_Playing:
		return Playing
	case pb.GameState_Completed:
		return Completed
	case pb.GameState_Aborted:
		return Aborted
	default:
		panic("Unknown phase")
	}
}

func (gs GameState) GetPlayerState(id string) (PlayerState, error) {
	for _, ps := range gs.Players {
		if id == ps.Id {
			return ps, nil
		}
	}
	return PlayerState{}, fmt.Errorf("no such player id %s", id)
}

type PlayerState struct {
	Id         string
	Name       string
	Cards      cards.Cards
	NumCards   int
	Tricks     []cards.Cards
	NumTricks  int
	TrickScore int
	HandScore  int
}

func (g GameState) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Game Phase: %s\n", g.Phase))
	if g.Phase != Preparing {
		for _, p := range g.Players {
			sb.WriteString(p.String(g.Phase == Completed))
		}
		if len(g.CurrentTrick) > 0 {
			sb.WriteString(fmt.Sprintf("Current Trick: %s\n", g.CurrentTrick))
		}
		if len(g.LegalPlays) > 0 {
			sb.WriteString(fmt.Sprintf("Legal Plays: %s", g.LegalPlays))
		}
	}
	return sb.String()
}

func (p PlayerState) String(isCompleted bool) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Name: %s\n", p.Name))
	if len(p.Cards) > 0 {
		sb.WriteString(fmt.Sprintf("Cards: %s\n", p.Cards.HandString()))
	} else if p.NumCards > 0 {
		sb.WriteString(fmt.Sprintf("Num Cards: %d\n", p.NumCards))
	}
	if len(p.Tricks) > 0 {
		sb.WriteString("Tricks:\n")
		for _, t := range p.Tricks {
			sb.WriteString(fmt.Sprintf("  %s\n", t.String()))
		}
	} else {
		sb.WriteString(fmt.Sprintf("Num Tricks Taken: %d\n", p.NumTricks))
	}
	if isCompleted {
		sb.WriteString(fmt.Sprintf("Hand Score: %d\n", p.HandScore))
	} else {
		sb.WriteString(fmt.Sprintf("Trick Score: %d\n", p.TrickScore))
	}
	return sb.String()
}

type connection struct {
	conn    *grpc.ClientConn
	client  pb.CardGameServiceClient
	verbose bool
}

func (c *connection) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

func (c *connection) Register(ctx context.Context, name string, callbacks GameCallbacks) (Session, error) {
	if name == "" {
		name = chooseRandomName()
	}
	req := &pb.RegisterRequest{
		Name: name,
	}
	resp, err := c.client.Register(ctx, req)
	if err != nil {
		return nil, err
	}
	return &session{client: c.client, playerId: resp.GetPlayerId(), callbacks: callbacks, verbose: c.verbose}, nil
}

type session struct {
	client    pb.CardGameServiceClient
	playerId  string
	callbacks GameCallbacks
	verbose   bool
}

type GameSummary struct {
	Id    string
	Phase GamePhase
	Names []string
}

func (c *connection) ListGames(ctx context.Context, phase ...GamePhase) ([]GameSummary, error) {
	var phases []pb.GameState_Phase
	for _, p := range phase {
		phases = append(phases, p.toProto())
	}
	req := &pb.ListGamesRequest{
		Phase: phases,
	}
	resp, err := c.client.ListGames(ctx, req)
	if err != nil {
		return []GameSummary{}, err
	}
	games := []GameSummary{}
	for _, g := range resp.GetGames() {
		games = append(games,
			GameSummary{
				Id:    g.GetId(),
				Phase: protoToPhase(g.GetPhase()),
				Names: g.GetPlayerNames(),
			})
	}
	return games, nil
}

func (s *session) GetPlayerId() string {
	return s.playerId
}
func (s *session) JoinGameAsPlayer(ctx context.Context, wg *sync.WaitGroup, gameId string) (string, error) {
	return s.JoinGame(ctx, wg, gameId, pb.JoinGameRequest_AsPlayer)
}
func (s *session) JoinGameAsObserver(ctx context.Context, wg *sync.WaitGroup, gameId string) (string, error) {
	return s.JoinGame(ctx, wg, gameId, pb.JoinGameRequest_AsObserver)
}
func (s *session) JoinGame(ctx context.Context, wg *sync.WaitGroup, gameId string, mode pb.JoinGameRequest_Mode) (string, error) {
	joinReq := &pb.JoinGameRequest{
		PlayerId: s.playerId,
		GameId:   gameId,
		Mode:     mode,
	}
	joinResp, err := s.client.JoinGame(ctx, joinReq)
	if err != nil {
		return "", err
	}
	activityReq := &pb.GameActivityRequest{
		PlayerId: s.playerId,
	}
	activityStream, err := s.client.ListenForGameActivity(ctx, activityReq)
	if err != nil {
		return "", err
	}
	go s.processActivity(wg, activityStream)
	return joinResp.GetGameId(), nil
}

// possible conn closed errors.
const possibleConnResetMsg = "connection reset by peer"
const possibleEOFMsg = "error reading from server: EOF"

// isConnClosedErr checks the error msg for possible conn closed messages.
func isConnClosedErr(err error) bool {
	errContainsConnResetMsg := strings.Contains(err.Error(), possibleConnResetMsg)
	errContainsEOFMsg := strings.Contains(err.Error(), possibleEOFMsg)
	return errContainsConnResetMsg || errContainsEOFMsg || err == io.EOF
}

func (s *session) processActivity(wg *sync.WaitGroup, activityStream pb.CardGameService_ListenForGameActivityClient) {
	for {
		activity, err := activityStream.Recv()
		if err != nil && isConnClosedErr(err) {
			s.callbacks.HandleConnectionError(s, fmt.Errorf("Connection to server closed"))
			break
		}
		if err != nil {
			log.Fatalf("ListenForGameActivity(_) = _, %v", err)
		}
		if s.verbose {
			log.Println(activity)
		}
		switch a := activity.Type.(type) {
		case *pb.GameActivityResponse_PlayerJoined_:
			pj := a.PlayerJoined
			err = s.callbacks.HandlePlayerJoined(s, pj.GetName(), pj.GetGameId())
		case *pb.GameActivityResponse_PlayerLeft_:
			pl := a.PlayerLeft
			err = s.callbacks.HandlePlayerLeft(s, pl.GetName(), pl.GetGameId())
		case *pb.GameActivityResponse_GameReadyToStart_:
			err = s.callbacks.HandleGameReadyToStart(s)
		case *pb.GameActivityResponse_GameStarted_:
			err = s.callbacks.HandleGameStarted(s)
		case *pb.GameActivityResponse_YourTurn_:
			err = s.callbacks.HandleYourTurn(s)
		case *pb.GameActivityResponse_TrickCompleted_:
			tc := a.TrickCompleted
			if trick, err := cards.ParseCards(tc.GetTrick()); err == nil {
				err = s.callbacks.HandleTrickCompleted(s, trick, tc.GetTrickWinnerId(), tc.GetTrickWinnerName())
			}
		case *pb.GameActivityResponse_GameFinished_:
			err = s.callbacks.HandleGameFinished(s)
		case *pb.GameActivityResponse_GameAborted_:
			err = s.callbacks.HandleGameAborted(s)
		}
		if err != nil {
			log.Printf("Error handling activity: %v\n", err)
			break
		}
	}
	wg.Done()
}

func (s *session) ReadyToStartGame(ctx context.Context) error {
	return s.performGameAction(ctx, &pb.GameActionRequest_ReadyToStartGame{})
}

func (s *session) LeaveGame(ctx context.Context) error {
	return s.performGameAction(ctx, &pb.GameActionRequest_LeaveGame{})
}

func (s *session) PlayCard(ctx context.Context, card cards.Card) error {
	return s.performGameAction(ctx,
		&pb.GameActionRequest_PlayCard{
			PlayCard: &pb.PlayCardAction{
				Card: card.String(),
			},
		},
	)
}

func (s *session) performGameAction(ctx context.Context, requestType pb.GameActionRequest_Type) error {
	req := &pb.GameActionRequest{
		PlayerId: s.playerId,
		Type:     requestType,
	}
	status, err := s.client.GameAction(ctx, req)
	if err != nil {
		return err
	}
	if status.Code != 0 {
		return fmt.Errorf("%v", status.Error)
	}
	return nil
}

func (c *connection) GetGameState(ctx context.Context, gameId string) (GameState, error) {
	req := &pb.GameStateRequest{
		Type: &pb.GameStateRequest_GameId{GameId: gameId},
	}
	return getGameState(ctx, c.client, req)
}
func (s *session) GetGameState(ctx context.Context) (GameState, error) {
	req := &pb.GameStateRequest{
		Type: &pb.GameStateRequest_PlayerId{PlayerId: s.playerId},
	}
	return getGameState(ctx, s.client, req)
}
func getGameState(ctx context.Context, client pb.CardGameServiceClient, req *pb.GameStateRequest) (GameState, error) {
	resp, err := client.GetGameState(ctx, req)
	if err != nil {
		return GameState{}, err
	}
	var phase GamePhase
	switch resp.GetPhase() {
	case pb.GameState_Preparing:
		phase = Preparing
	case pb.GameState_Playing:
		phase = Playing
	case pb.GameState_Completed:
		phase = Completed
	case pb.GameState_Aborted:
		phase = Aborted
	}
	players := []PlayerState{}
	for _, p := range resp.GetPlayers() {
		ps, err := toPlayerState(p)
		if err != nil {
			return GameState{}, err
		}
		players = append(players, ps)
	}
	currentTrick, err := cards.ParseCards(resp.GetCurrentTrick().GetCards())
	if err != nil {
		return GameState{}, err
	}
	legalPlays, err := cards.ParseCards(resp.GetLegalPlays().GetCards())
	if err != nil {
		return GameState{}, err
	}
	return GameState{
		Id:           resp.GetId(),
		Phase:        phase,
		Players:      players,
		CurrentTrick: currentTrick,
		LegalPlays:   legalPlays,
	}, nil
}

func toPlayerState(p *pb.GameState_Player) (PlayerState, error) {
	cs, err := cards.ParseCards(p.GetCards().GetCards())
	if err != nil {
		return PlayerState{}, err
	}
	var tricks []cards.Cards
	for _, t := range p.GetTricks() {
		ts, err := cards.ParseCards(t.GetCards())
		if err != nil {
			return PlayerState{}, err
		}
		tricks = append(tricks, ts)
	}
	return PlayerState{
		Id:         p.GetId(),
		Name:       p.GetName(),
		Cards:      cs,
		NumCards:   int(p.GetNumCards()),
		Tricks:     tricks,
		NumTricks:  int(p.GetNumTricks()),
		TrickScore: int(p.GetTrickScore()),
		HandScore:  int(p.GetHandScore()),
	}, nil
}

/*
func newInProcessServer() pb.CardGameServiceClient {
	return &inProcessServer{server: server.NewCardGameService()}
}

type inProcessServer struct {
	server pb.CardGameServiceServer
}

func (s inProcessServer) Ping(ctx context.Context, in *pb.PingRequest, opts ...grpc.CallOption) (*pb.PingResponse, error) {
	return s.server.Ping(ctx, in)
}
func (s inProcessServer) Register(ctx context.Context, in *pb.RegisterRequest, opts ...grpc.CallOption) (*pb.RegisterResponse, error) {
	return s.server.Register(ctx, in)
}
func (s inProcessServer) ListGames(ctx context.Context, in *pb.ListGamesRequest, opts ...grpc.CallOption) (*pb.ListGamesResponse, error) {
	return s.server.ListGames(ctx, in)
}
func (s inProcessServer) JoinGame(ctx context.Context, in *pb.JoinGameRequest, opts ...grpc.CallOption) (*pb.JoinGameResponse, error) {
	return s.server.JoinGame(ctx, in)
}
func (s inProcessServer) LeaveGame(ctx context.Context, in *pb.LeaveGameRequest, opts ...grpc.CallOption) (*pb.LeaveGameResponse, error) {
	return s.server.LeaveGame(ctx, in)
}
func (s inProcessServer) GetGameState(ctx context.Context, in *pb.GameStateRequest, opts ...grpc.CallOption) (*pb.GameState, error) {
	return s.server.GetGameState(ctx, in)
}
func (s inProcessServer) PlayerAction(ctx context.Context, in *pb.PlayerActionRequest, opts ...grpc.CallOption) (*pb.Status, error) {
	return s.server.PlayerAction(ctx, in)
}
func (s inProcessServer) ListenForGameActivity(ctx context.Context, in *pb.GameActivityRequest, opts ...grpc.CallOption) (pb.CardGameService_ListenForGameActivityClient, error) {
	return nil, fmt.Errorf("Listen not implemented")
		// 		service := &listenService{ch: make(chan *pb.GameActivityResponse)}
		// 		err := s.server.ListenForGameActivity(in, service)
		// 		if err != nil {
		// 			return nil, err
		// 		}
		// 		return service, nil
		// type listenService struct {
		// 	ch chan *pb.GameActivityResponse
		// }
}
*/
