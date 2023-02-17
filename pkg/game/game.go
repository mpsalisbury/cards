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
	ReportPlayerJoined(name string, gameId string)
	ReportPlayerLeft(name string, gameId string)
	ReportGameStarted()
	ReportCardPlayed()
	ReportTrickCompleted()
	ReportGameFinished()
	ReportGameAborted()
	ReportYourTurn(pId string)
	BroadcastMessage(msg string)
}
