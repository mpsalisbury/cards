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
	JoinGameAsPlayer(ctx context.Context) error
	JoinGameAsObserver(ctx context.Context) error
	GetGameState(ctx context.Context) (GameState, error)
	PlayCard(ctx context.Context, card cards.Card) error
}

type GameCallbacks interface {
	HandlePlayerJoined(name string)
	HandleGameStarted()
	HandleCardPlayed( /*currentTrick cards.Cards*/ )
	HandleYourTurn()
	HandleTrickCompleted( /*trick cards.Cards, trickWinner string*/ )
	HandleGameFinished()
}
type UnimplementedGameCallbacks struct{}

func (UnimplementedGameCallbacks) HandlePlayerJoined(name string) {}
func (UnimplementedGameCallbacks) HandleGameStarted()             {}
func (UnimplementedGameCallbacks) HandleCardPlayed()              {}
func (UnimplementedGameCallbacks) HandleYourTurn()                {}
func (UnimplementedGameCallbacks) HandleTrickCompleted()          {}
func (UnimplementedGameCallbacks) HandleGameFinished()            {}

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

func (c *connection) JoinGameAsPlayer(ctx context.Context) error {
	return c.JoinGame(ctx, pb.JoinGameRequest_AsPlayer)
}
func (c *connection) JoinGameAsObserver(ctx context.Context) error {
	return c.JoinGame(ctx, pb.JoinGameRequest_AsObserver)
}
func (c *connection) JoinGame(ctx context.Context, mode pb.JoinGameRequest_Mode) error {
	joinReq := &pb.JoinGameRequest{
		SessionId: c.sessionId,
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

func (c *connection) processActivity(activityStream pb.CardGameService_ListenForGameActivityClient) {
	for {
		activity, err := activityStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("ListenForGameActivity(_) = _, %v", err)
		}
		log.Println(activity)
		switch a := activity.Type.(type) {
		case *pb.GameActivityResponse_PlayerJoined_:
			c.callbacks.HandlePlayerJoined(a.PlayerJoined.GetName())
		case *pb.GameActivityResponse_GameStarted_:
			c.callbacks.HandleGameStarted()
		case *pb.GameActivityResponse_YourTurn_:
			c.callbacks.HandleYourTurn()
		case *pb.GameActivityResponse_GameFinished_:
			c.callbacks.HandleGameFinished()
		case *pb.GameActivityResponse_LiveCheck_:
			// Do nothing. This is just to make sure we're still alive.
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
