package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/mpsalisbury/cards/pkg/cards"
	pb "github.com/mpsalisbury/cards/pkg/proto"
	"github.com/mpsalisbury/cards/pkg/server"
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
	InProcessServer
)

// Creates a flag for specifying the server type to use.
func AddServerFlag(target *string, name string) {
	EnumFlag(target, name, []string{"local", "hosted", "inprocess"}, "Type of server to use")
}

// Constructs a player from a player flag value.
func ServerTypeFromFlag(serverType string) (ServerType, error) {
	switch serverType {
	case "", "local":
		return LocalServer, nil
	case "hosted":
		return HostedServer, nil
	case "inprocess":
		return InProcessServer, nil
	default:
		return LocalServer, fmt.Errorf("invalid server type %s", serverType)
	}
}

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
	return NewConnection(conn, client, verbose), nil
}

func createClient(stype ServerType) (*grpc.ClientConn, pb.CardGameServiceClient, error) {
	switch stype {
	case LocalServer, HostedServer:
		return connectToExternalServer(stype)
	case InProcessServer:
		return connectToInProcessServer()
	}
	return nil, nil, fmt.Errorf("server type %v not supported", stype)
}

func connectToExternalServer(stype ServerType) (*grpc.ClientConn, pb.CardGameServiceClient, error) {
	config := configs[stype]

	cred := func() credentials.TransportCredentials {
		if config.secure {
			return credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: false,
			})
		}
		return insecure.NewCredentials()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	conn, err := grpc.DialContext(ctx, config.serverAddr, grpc.WithTransportCredentials(cred), grpc.WithBlock())
	if err != nil {
		return nil, nil, fmt.Errorf("error connecting to server at %s: %v", config.serverAddr, err)
	}
	client := pb.NewCardGameServiceClient(conn)
	return conn, client, nil
}

func connectToInProcessServer() (*grpc.ClientConn, pb.CardGameServiceClient, error) {
	return nil, newInProcessServer(), nil
}

type Connection interface {
	Close()
	Register(ctx context.Context, name string, gameCallbacks GameCallbacks) (Session, error)
	RegisterObserver(ctx context.Context, wg *sync.WaitGroup, name string, registryCallbacks RegistryCallbacks, gameCallbacks GameCallbacks) (Session, error)
	CreateGame(ctx context.Context) (gameId string, err error)
	ListGames(ctx context.Context, phase ...GamePhase) ([]GameSummary, error)
	GetGameState(ctx context.Context, gameId string) (GameState, error)
}
type Session interface {
	GetSessionId() string
	JoinGame(ctx context.Context, wg *sync.WaitGroup, gameId string) error
	ObserveGame(ctx context.Context, wg *sync.WaitGroup, gameId string) error
	ReadyToStartGame(ctx context.Context, gameId string) error
	LeaveGame(ctx context.Context, gameId string) error
	PlayCard(ctx context.Context, gameId string, card cards.Card) error
	GetGameState(ctx context.Context, gameId string) (GameState, error)
}

type GameCallbacks interface {
	HandlePlayerJoined(s Session, name string, gameId string) error
	HandlePlayerLeft(s Session, name string, gameId string) error
	HandleGameReadyToStart(s Session, gameId string) error
	HandleGameStarted(s Session, gameId string) error
	HandleCardPlayed(s Session, gameId string) error
	HandleYourTurn(s Session, gameId string) error
	HandleTrickCompleted(s Session, gameId string, trick cards.Cards, winningCard cards.Card, winnerId, winnerName string) error
	HandleGameFinished(s Session, gameId string)
	HandleGameAborted(s Session, gameId string)
	HandleConnectionError(s Session, err error)
}
type UnimplementedGameCallbacks struct{}

func (UnimplementedGameCallbacks) HandlePlayerJoined(s Session, name string, gameId string) error {
	return nil
}
func (UnimplementedGameCallbacks) HandlePlayerLeft(s Session, name string, gameId string) error {
	return nil
}
func (UnimplementedGameCallbacks) HandleGameReadyToStart(s Session, gameId string) error {
	// By default we reply that this player is ready to start playing, since we got the notification.
	return s.ReadyToStartGame(context.Background(), gameId)
}
func (UnimplementedGameCallbacks) HandleGameStarted(s Session, gameId string) error { return nil }
func (UnimplementedGameCallbacks) HandleCardPlayed(s Session, gameId string) error  { return nil }
func (UnimplementedGameCallbacks) HandleYourTurn(s Session, gameId string) error    { return nil }
func (UnimplementedGameCallbacks) HandleTrickCompleted(
	s Session, gameId string, trick cards.Cards, winningCard cards.Card, winnerId, winnerName string) error {
	return nil
}
func (UnimplementedGameCallbacks) HandleGameFinished(s Session, gameId string) {}
func (UnimplementedGameCallbacks) HandleGameAborted(s Session, gameId string)  {}
func (UnimplementedGameCallbacks) HandleConnectionError(Session, error)        {}

type RegistryCallbacks interface {
	InstallSession(Session)
	HandleGameCreated(c Connection, gameId string) error
	HandleGameDeleted(c Connection, gameId string) error
	HandleFullGamesList(c Connection, gameIds []string) error
	HandleConnectionError(c Connection, err error)
}
type UnimplementedRegistryCallbacks struct{}

func (UnimplementedRegistryCallbacks) InstallSession(Session) {}
func (UnimplementedRegistryCallbacks) HandleGameCreated(c Connection, gameId string) error {
	return nil
}
func (UnimplementedRegistryCallbacks) HandleGameDeleted(c Connection, gameId string) error {
	return nil
}
func (UnimplementedRegistryCallbacks) HandleFullGamesList(c Connection, gameIds []string) error {
	return nil
}
func (UnimplementedRegistryCallbacks) HandleConnectionError(Connection, error) {}

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

func NewConnection(conn *grpc.ClientConn, client pb.CardGameServiceClient, verbose bool) Connection {
	return &connection{
		conn:    conn,
		client:  client,
		verbose: verbose,
	}
}

type connection struct {
	conn              *grpc.ClientConn
	client            pb.CardGameServiceClient
	registryCallbacks RegistryCallbacks
	verbose           bool
}

func (c *connection) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

func (c *connection) Register(ctx context.Context, name string, gameCallbacks GameCallbacks) (Session, error) {
	// Client won't be waiting for registry callbacks.
	wg := new(sync.WaitGroup)
	wg.Add(1)
	registryCallbacks := &UnimplementedRegistryCallbacks{}
	return c.RegisterObserver(ctx, wg, name, registryCallbacks, gameCallbacks)
}
func (c *connection) RegisterObserver(ctx context.Context, wg *sync.WaitGroup, name string,
	registryCallbacks RegistryCallbacks, gameCallbacks GameCallbacks) (Session, error) {
	if name == "" {
		name = chooseRandomName()
	}
	req := &pb.RegisterRequest{
		Name: name,
	}
	registryActivityStream, err := c.client.Register(ctx, req)
	if err != nil {
		return nil, err
	}
	c.registryCallbacks = registryCallbacks
	sessionIdChan := make(chan string)
	wg.Add(1)
	go c.processRegistryActivity(wg, sessionIdChan, registryActivityStream)
	// TODO: add timeout
	sessionId := <-sessionIdChan
	session := newSession(c.client, sessionId, gameCallbacks, c.verbose)
	registryCallbacks.InstallSession(session)
	return session, nil
}

func newSession(client pb.CardGameServiceClient, sessionId string, gameCallbacks GameCallbacks, verbose bool) Session {
	return &session{
		client:        client,
		sessionId:     sessionId,
		gameCallbacks: gameCallbacks,
		verbose:       verbose,
	}
}

type session struct {
	client        pb.CardGameServiceClient
	sessionId     string
	gameCallbacks GameCallbacks
	verbose       bool
}

type GameSummary struct {
	Id    string
	Phase GamePhase
	Names []string
}

func (c *connection) CreateGame(ctx context.Context) (gameId string, err error) {
	req := &pb.CreateGameRequest{}
	resp, err := c.client.CreateGame(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.GetGameId(), nil
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
func (s *session) GetSessionId() string {
	return s.sessionId
}
func (s *session) JoinGame(ctx context.Context, wg *sync.WaitGroup, gameId string) error {
	req := &pb.JoinGameRequest{
		SessionId: s.sessionId,
		GameId:    gameId,
	}
	gameActivityStream, err := s.client.JoinGame(ctx, req)
	if err != nil {
		return err
	}
	wg.Add(1)
	go s.processGameActivity(wg, gameActivityStream)
	return nil
}
func (s *session) ObserveGame(ctx context.Context, wg *sync.WaitGroup, gameId string) error {
	req := &pb.ObserveGameRequest{
		SessionId: s.sessionId,
		GameId:    gameId,
	}
	gameActivityStream, err := s.client.ObserveGame(ctx, req)
	if err != nil {
		return err
	}
	wg.Add(1)
	go s.processGameActivity(wg, gameActivityStream)
	return nil
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

// Handles both JoinGame and ObserveGame streams.
func (s *session) processGameActivity(wg *sync.WaitGroup, gameActivityStream pb.CardGameService_ObserveGameClient) {
	defer wg.Done()
	for {
		activity, err := gameActivityStream.Recv()
		if err != nil && isConnClosedErr(err) {
			s.gameCallbacks.HandleConnectionError(s, fmt.Errorf("Connection to server closed"))
			return
		}
		if err != nil {
			log.Fatalf("ListenForGameActivity(_) = _, %v", err)
		}
		if s.verbose {
			log.Println(activity)
		}
		gameId := activity.GetGameId()
		switch a := activity.Type.(type) {
		case *pb.GameActivity_PlayerJoined_:
			pj := a.PlayerJoined
			err = s.gameCallbacks.HandlePlayerJoined(s, pj.GetName(), gameId)
		case *pb.GameActivity_PlayerLeft_:
			pl := a.PlayerLeft
			err = s.gameCallbacks.HandlePlayerLeft(s, pl.GetName(), gameId)
		case *pb.GameActivity_GameReadyToStart_:
			err = s.gameCallbacks.HandleGameReadyToStart(s, gameId)
		case *pb.GameActivity_GameStarted_:
			err = s.gameCallbacks.HandleGameStarted(s, gameId)
		case *pb.GameActivity_YourTurn_:
			err = s.gameCallbacks.HandleYourTurn(s, gameId)
		case *pb.GameActivity_TrickCompleted_:
			tc := a.TrickCompleted
			trick, err1 := cards.ParseCards(tc.GetTrick())
			winningCard, err2 := cards.ParseCard(tc.GetWinningCard())
			if err1 == nil && err2 == nil {
				err = s.gameCallbacks.HandleTrickCompleted(s, gameId, trick, winningCard, tc.GetWinnerId(), tc.GetWinnerName())
			}
		case *pb.GameActivity_GameFinished_:
			s.gameCallbacks.HandleGameFinished(s, gameId)
			return
		case *pb.GameActivity_GameAborted_:
			s.gameCallbacks.HandleGameAborted(s, gameId)
			return
		}
		if err != nil {
			log.Printf("Error handling activity: %v\n", err)
			return
		}
	}
}

func (s *session) ReadyToStartGame(ctx context.Context, gameId string) error {
	return s.performGameAction(ctx, gameId, &pb.GameActionRequest_ReadyToStartGame{})
}

func (s *session) LeaveGame(ctx context.Context, gameId string) error {
	return s.performGameAction(ctx, gameId, &pb.GameActionRequest_LeaveGame{})
}

func (s *session) PlayCard(ctx context.Context, gameId string, card cards.Card) error {
	return s.performGameAction(ctx,
		gameId,
		&pb.GameActionRequest_PlayCard{
			PlayCard: &pb.PlayCardAction{
				Card: card.String(),
			},
		},
	)
}

func (s *session) performGameAction(ctx context.Context, gameId string, requestType pb.GameActionRequest_Type) error {
	req := &pb.GameActionRequest{
		SessionId: s.sessionId,
		GameId:    gameId,
		Type:      requestType,
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
		GameId: gameId,
	}
	return getGameState(ctx, c.client, req)
}
func (s *session) GetGameState(ctx context.Context, gameId string) (GameState, error) {
	req := &pb.GameStateRequest{
		SessionId: s.sessionId,
		GameId:    gameId,
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

func (c *connection) processRegistryActivity(wg *sync.WaitGroup, sessionIdChan chan string, registryActivityStream pb.CardGameService_RegisterClient) {
	defer wg.Done()
	for {
		activity, err := registryActivityStream.Recv()
		if err != nil && isConnClosedErr(err) {
			c.registryCallbacks.HandleConnectionError(c, fmt.Errorf("Connection to server closed"))
			return
		}
		if err != nil {
			log.Fatalf("ListenForRegistryActivity(_) = _, %v", err)
		}
		if c.verbose {
			log.Println(activity)
		}
		switch a := activity.Type.(type) {
		case *pb.RegistryActivity_SessionCreated_:
			sessionId := a.SessionCreated.GetSessionId()
			sessionIdChan <- sessionId
		case *pb.RegistryActivity_GameCreated_:
			gameId := a.GameCreated.GetGameId()
			err = c.registryCallbacks.HandleGameCreated(c, gameId)
		case *pb.RegistryActivity_GameDeleted_:
			gameId := a.GameDeleted.GetGameId()
			err = c.registryCallbacks.HandleGameDeleted(c, gameId)
		case *pb.RegistryActivity_FullGamesList_:
			gameIds := a.FullGamesList.GetGameIds()
			err = c.registryCallbacks.HandleFullGamesList(c, gameIds)
		}
		if err != nil {
			log.Printf("Error handling activity: %v\n", err)
			return
		}
	}
}

func newInProcessServer() pb.CardGameServiceClient {
	return &inProcessServer{server: server.NewCardGameService()}
}

type inProcessServer struct {
	server pb.CardGameServiceServer
}

func (s inProcessServer) Ping(ctx context.Context, in *pb.PingRequest, opts ...grpc.CallOption) (*pb.PingResponse, error) {
	return s.server.Ping(ctx, in)
}
func (s inProcessServer) Register(ctx context.Context, in *pb.RegisterRequest, opts ...grpc.CallOption) (pb.CardGameService_RegisterClient, error) {
	client, server := makeRegisterLocalConnectors()
	go s.server.Register(in, server)
	return client, nil
}
func (s inProcessServer) CreateGame(ctx context.Context, in *pb.CreateGameRequest, opts ...grpc.CallOption) (*pb.CreateGameResponse, error) {
	return s.server.CreateGame(ctx, in)
}
func (s inProcessServer) ListGames(ctx context.Context, in *pb.ListGamesRequest, opts ...grpc.CallOption) (*pb.ListGamesResponse, error) {
	return s.server.ListGames(ctx, in)
}
func (s inProcessServer) JoinGame(ctx context.Context, in *pb.JoinGameRequest, opts ...grpc.CallOption) (pb.CardGameService_JoinGameClient, error) {
	client, server := makeObserveGameLocalConnectors()
	go s.server.JoinGame(in, server)
	return client, nil
}
func (s inProcessServer) ObserveGame(ctx context.Context, in *pb.ObserveGameRequest, opts ...grpc.CallOption) (pb.CardGameService_ObserveGameClient, error) {
	client, server := makeObserveGameLocalConnectors()
	go s.server.ObserveGame(in, server)
	return client, nil
}
func (s inProcessServer) GameAction(ctx context.Context, in *pb.GameActionRequest, opts ...grpc.CallOption) (*pb.Status, error) {
	return s.server.GameAction(ctx, in)
}
func (s inProcessServer) GetGameState(ctx context.Context, in *pb.GameStateRequest, opts ...grpc.CallOption) (*pb.GameState, error) {
	return s.server.GetGameState(ctx, in)
}

func makeRegisterLocalConnectors() (pb.CardGameService_RegisterClient, pb.CardGameService_RegisterServer) {
	ch := make(chan *pb.RegistryActivity)
	client := &localRegistryActivityClient{ch: ch}
	server := &localRegistryActivityServer{ch: ch, svrctx: context.Background()}
	return client, server
}

type localRegistryActivityServer struct {
	grpc.ServerStream
	ch     chan *pb.RegistryActivity
	svrctx context.Context
}

func (s localRegistryActivityServer) Send(ra *pb.RegistryActivity) error {
	s.ch <- ra
	return nil
}

func (s localRegistryActivityServer) Context() context.Context {
	return s.svrctx
}

type localRegistryActivityClient struct {
	grpc.ClientStream
	ch chan *pb.RegistryActivity
}

func (c localRegistryActivityClient) Recv() (*pb.RegistryActivity, error) {
	ra := <-c.ch
	return ra, nil
}

// Works for both JoinGame and ObserveGame servers.
func makeObserveGameLocalConnectors() (pb.CardGameService_ObserveGameClient, pb.CardGameService_ObserveGameServer) {
	ch := make(chan *pb.GameActivity)
	client := &localGameActivityClient{ch: ch}
	server := &localGameActivityServer{ch: ch, svrctx: context.Background()}
	return client, server
}

type localGameActivityServer struct {
	grpc.ServerStream
	ch     chan *pb.GameActivity
	svrctx context.Context
}

func (s localGameActivityServer) Context() context.Context {
	return s.svrctx
}

func (s localGameActivityServer) Send(ga *pb.GameActivity) error {
	s.ch <- ga
	return nil
}

type localGameActivityClient struct {
	grpc.ClientStream
	ch chan *pb.GameActivity
}

func (c localGameActivityClient) Recv() (*pb.GameActivity, error) {
	ga := <-c.ch
	return ga, nil
}
