package game

import (
	"github.com/mpsalisbury/cards/pkg/cards"
	pb "github.com/mpsalisbury/cards/pkg/proto"
)

type Game interface {
	Id() string
	Phase() GamePhase
	AcceptingMorePlayers() bool
	AddPlayer(name, playerId string)
	AddObserver(name, playerId string)
	ListenerIds() []string
	PlayerNames() []string
	NextPlayerId() string
	RemovePlayer(playerId string) error
	StartIfReady() bool
	GetGameState(playerId string) (*pb.GameState, error)
	HandlePlayCard(playerId string, card cards.Card, reporter Reporter) error
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
