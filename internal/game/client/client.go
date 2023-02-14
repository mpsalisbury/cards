package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/mpsalisbury/cards/internal/cards"
	pb "github.com/mpsalisbury/cards/internal/game/proto"
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
)

var configs = map[ServerType]struct {
	serverAddr string
	secure     bool
}{
	LocalServer:  {localServer, false},
	HostedServer: {hostedServer, true},
}

func Connect(stype ServerType) (Connection, error) {
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
		return nil, err
	}
	client := pb.NewCardGameServiceClient(conn)
	return &connection{conn: conn, client: client}, nil
}

type Connection interface {
	Close()
	Register(ctx context.Context, name string, callbacks GameCallbacks) error
	ListGames(ctx context.Context, phase GamePhase) ([]GameSummary, error)
	JoinGameAsPlayer(ctx context.Context, gameId string) error
	JoinGameAsObserver(ctx context.Context, gameId string) error
	GetGameState(ctx context.Context) (GameState, error)
	PlayCard(ctx context.Context, card cards.Card) error
}

type GameCallbacks interface {
	HandlePlayerJoined(name string, gameId string) error
	HandlePlayerLeft(name string, gameId string) error
	HandleGameStarted() error
	HandleCardPlayed( /*currentTrick cards.Cards*/ ) error
	HandleYourTurn() error
	HandleTrickCompleted( /*trick cards.Cards, trickWinner string*/ ) error
	HandleGameFinished() error
	HandleGameAborted() error
	HandleConnectionError(err error)
}
type UnimplementedGameCallbacks struct{}

func (UnimplementedGameCallbacks) HandlePlayerJoined(name string, gameId string) error { return nil }
func (UnimplementedGameCallbacks) HandlePlayerLeft(name string, gameId string) error   { return nil }
func (UnimplementedGameCallbacks) HandleGameStarted() error                            { return nil }
func (UnimplementedGameCallbacks) HandleCardPlayed() error                             { return nil }
func (UnimplementedGameCallbacks) HandleYourTurn() error                               { return nil }
func (UnimplementedGameCallbacks) HandleTrickCompleted() error                         { return nil }
func (UnimplementedGameCallbacks) HandleGameFinished() error                           { return nil }
func (UnimplementedGameCallbacks) HandleGameAborted() error                            { return nil }
func (UnimplementedGameCallbacks) HandleConnectionError(error)                         {}

type GameState struct {
	Phase        GamePhase
	Players      []PlayerState
	CurrentTrick cards.Cards
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

type PlayerState struct {
	Name       string
	Cards      cards.Cards
	NumCards   int
	Tricks     []cards.Cards
	NumTricks  int
	TrickScore int
}

func (g GameState) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Game Phase: %s\n", g.Phase))
	if g.Phase != Preparing {
		for _, p := range g.Players {
			sb.WriteString(p.String())
		}
		if len(g.CurrentTrick) > 0 {
			sb.WriteString(fmt.Sprintf("Current Trick: %s", g.CurrentTrick))
		}
	}
	return sb.String()
}

func (p PlayerState) String() string {
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
	if p.TrickScore > 0 {
		sb.WriteString(fmt.Sprintf("Trick Score: %d\n", p.TrickScore))
	}
	return sb.String()
}

type connection struct {
	conn      *grpc.ClientConn
	client    pb.CardGameServiceClient
	sessionId string
	callbacks GameCallbacks
}

func (c *connection) Close() {
	c.conn.Close()
}

func (c *connection) Register(ctx context.Context, name string, callbacks GameCallbacks) error {
	req := &pb.RegisterRequest{
		Name: name,
	}
	resp, err := c.client.Register(ctx, req)
	if err != nil {
		return err
	}
	c.sessionId = resp.GetSessionId()
	c.callbacks = callbacks
	return nil
}

type GameSummary struct {
	Id    string
	Phase GamePhase
	Names []string
}

func (c *connection) ListGames(ctx context.Context, phase GamePhase) ([]GameSummary, error) {
	req := &pb.ListGamesRequest{
		Phase: []pb.GameState_Phase{phase.toProto()},
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

func (c *connection) JoinGameAsPlayer(ctx context.Context, gameId string) error {
	return c.JoinGame(ctx, gameId, pb.JoinGameRequest_AsPlayer)
}
func (c *connection) JoinGameAsObserver(ctx context.Context, gameId string) error {
	return c.JoinGame(ctx, gameId, pb.JoinGameRequest_AsObserver)
}
func (c *connection) JoinGame(ctx context.Context, gameId string, mode pb.JoinGameRequest_Mode) error {
	joinReq := &pb.JoinGameRequest{
		SessionId: c.sessionId,
		GameId:    gameId,
		Mode:      mode,
	}
	_, err := c.client.JoinGame(ctx, joinReq)
	if err != nil {
		return err
	}

	activityReq := &pb.GameActivityRequest{
		SessionId: c.sessionId,
	}
	activityStream, err := c.client.ListenForGameActivity(ctx, activityReq)
	if err != nil {
		return err
	}
	go c.processActivity(activityStream)
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

func (c *connection) processActivity(activityStream pb.CardGameService_ListenForGameActivityClient) {
	for {
		activity, err := activityStream.Recv()
		if err != nil && isConnClosedErr(err) {
			c.callbacks.HandleConnectionError(fmt.Errorf("Connection to server closed"))
			break
		}
		if err != nil {
			log.Fatalf("ListenForGameActivity(_) = _, %v", err)
		}
		log.Println(activity)
		switch a := activity.Type.(type) {
		case *pb.GameActivityResponse_PlayerJoined_:
			pj := a.PlayerJoined
			err = c.callbacks.HandlePlayerJoined(pj.GetName(), pj.GetGameId())
		case *pb.GameActivityResponse_PlayerLeft_:
			pl := a.PlayerLeft
			err = c.callbacks.HandlePlayerLeft(pl.GetName(), pl.GetGameId())
		case *pb.GameActivityResponse_GameStarted_:
			err = c.callbacks.HandleGameStarted()
		case *pb.GameActivityResponse_YourTurn_:
			err = c.callbacks.HandleYourTurn()
		case *pb.GameActivityResponse_GameFinished_:
			err = c.callbacks.HandleGameFinished()
		case *pb.GameActivityResponse_GameAborted_:
			err = c.callbacks.HandleGameAborted()
		}
		if err != nil {
			log.Printf("Error handling activity: %v\n", err)
			break
		}
	}
}

func (c *connection) GetGameState(ctx context.Context) (GameState, error) {
	req := &pb.GameStateRequest{
		SessionId: c.sessionId,
	}
	resp, err := c.client.GetGameState(ctx, req)
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
	return GameState{
		Phase:        phase,
		Players:      players,
		CurrentTrick: currentTrick,
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
		Name:       p.GetName(),
		Cards:      cs,
		NumCards:   int(p.GetNumCards()),
		Tricks:     tricks,
		NumTricks:  int(p.GetNumTricks()),
		TrickScore: int(p.GetTrickScore()),
	}, nil
}

func (c *connection) PlayCard(ctx context.Context, card cards.Card) error {
	req := &pb.PlayerActionRequest{
		SessionId: c.sessionId,
		Type: &pb.PlayerActionRequest_PlayCard{
			PlayCard: &pb.PlayCardAction{
				Card: card.String(),
			},
		},
	}
	status, err := c.client.PlayerAction(ctx, req)
	if err != nil {
		return err
	}
	if status.Code != 0 {
		return fmt.Errorf("%v", status.Error)
	}
	return nil
}
