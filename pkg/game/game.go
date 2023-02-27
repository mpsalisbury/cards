package game

import (
	"time"

	"github.com/mpsalisbury/cards/pkg/cards"
	pb "github.com/mpsalisbury/cards/pkg/proto"
)

type Game interface {
	Id() string
	Phase() GamePhase
	GetLastActivityTime() time.Time
	AcceptingMorePlayers() bool
	AddPlayer(name, sessionId string)
	PlayerNames() []string
	NextPlayerId() string
	RemovePlayer(sessionId string) error
	IsEnoughPlayersToStart() bool
	ConfirmPlayerReadyToStart(sessionId string) error
	UnconfirmedPlayerIds() []string
	StartGame()
	GetGameState(sessionId string) (*pb.GameState, error)
	HandlePlayCard(sessionId string, card cards.Card, reporter Reporter) error
	Abort()
}

type GamePhase int8

const (
	Preparing GamePhase = iota
	Playing
	Completed
	Aborted
)

func (ph GamePhase) ToProto() pb.GameState_Phase {
	switch ph {
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

// Report activity back to the players.
type Reporter interface {
	ReportPlayerJoined(g Game, name string)
	ReportPlayerLeft(g Game, name string)
	ReportGameStarted(g Game)
	ReportCardPlayed(g Game)
	ReportTrickCompleted(g Game, trick cards.Cards, trickWinnerId, trickWinnerName string)
	ReportGameFinished(g Game)
	ReportGameAborted(g Game)
	ReportNextTurn(g Game)
	BroadcastMessage(g Game, msg string)
}
