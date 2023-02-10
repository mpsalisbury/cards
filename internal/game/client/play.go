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
	JoinGame(ctx context.Context) error
	GetGameState(ctx context.Context) (GameState, error)
	PlayCard(ctx context.Context, card cards.Card) error
}

type GameCallbacks interface {
	HandlePlayerJoined(name string)
	HandleGameStarted()
	HandleYourTurn()
	HandleGameFinished()
}

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
	Name              string
	Cards             cards.Cards
	NumCardsRemaining int
	NumTricksTaken    int
	TrickScore        int
}

func (g GameState) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Game Phase: %s\n", g.Phase))
	if g.Phase != Preparing {
		for _, p := range g.Players {
			sb.WriteString(p.String())
		}
		sb.WriteString(fmt.Sprintf("Current Trick: %s", g.CurrentTrick))
	}
	return sb.String()
}

func (p PlayerState) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Name: %s\n", p.Name))
	if len(p.Cards) > 0 {
		sb.WriteString(fmt.Sprintf("Cards: %s\n", p.Cards.HandString()))
	} else {
		sb.WriteString(fmt.Sprintf("Num Cards: %d\n", p.NumCardsRemaining))
	}
	sb.WriteString(fmt.Sprintf("Num Tricks Taken: %d\n", p.NumTricksTaken))
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

func (c *connection) JoinGame(ctx context.Context) error {
	joinReq := &pb.JoinGameRequest{
		SessionId: c.sessionId,
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
		case *pb.GameActivityResponse_PlayerJoined:
			c.callbacks.HandlePlayerJoined(a.PlayerJoined.GetName())
		case *pb.GameActivityResponse_GameStarted:
			c.callbacks.HandleGameStarted()
		case *pb.GameActivityResponse_YourTurn:
			c.callbacks.HandleYourTurn()
		case *pb.GameActivityResponse_GameFinished:
			c.callbacks.HandleGameFinished()
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
	switch resp.GetState() {
	case pb.GameStateResponse_Preparing:
		phase = Preparing
	case pb.GameStateResponse_Playing:
		phase = Playing
	case pb.GameStateResponse_Completed:
		phase = Completed
	case pb.GameStateResponse_Aborted:
		phase = Aborted
	}
	myPlayer, err := yourPlayerToPlayerState(resp.GetPlayer())
	if err != nil {
		return GameState{}, err
	}
	players := []PlayerState{myPlayer}
	for _, p := range resp.GetOtherPlayers() {
		players = append(players, otherPlayerToPlayerState(p))
	}
	currentTrick, err := cards.ParseCards(resp.GetCurrentTrickCards())
	if err != nil {
		return GameState{}, err
	}
	return GameState{
		Phase:        phase,
		Players:      players,
		CurrentTrick: currentTrick,
	}, nil
}

func yourPlayerToPlayerState(p *pb.GameStateResponse_YourPlayerState) (PlayerState, error) {
	cards, err := cards.ParseCards(p.GetCards())
	if err != nil {
		return PlayerState{}, err
	}
	return PlayerState{
		Name:           p.GetName(),
		Cards:          cards,
		NumTricksTaken: int(p.GetNumTricksTaken()),
		TrickScore:     int(p.GetTrickScore()),
	}, nil
}
func otherPlayerToPlayerState(p *pb.GameStateResponse_OtherPlayerState) PlayerState {
	return PlayerState{
		Name:              p.GetName(),
		NumCardsRemaining: int(p.GetNumCardsRemaining()),
		NumTricksTaken:    int(p.GetNumTricksTaken()),
	}
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
