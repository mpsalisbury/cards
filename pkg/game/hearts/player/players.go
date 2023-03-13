package player

import (
	"fmt"

	"github.com/mpsalisbury/cards/pkg/client"
)

// Creates a flag for specifying the player type to use.
func AddPlayerFlag(target *string, name string) {
	client.EnumFlag(target, name, []string{"basic", "term", "random"}, "Type of player logic to use")
}

// Constructs a player from a player flag value.
func NewPlayerFromFlag(playerType string, hints bool) (client.GameCallbacks, error) {
	switch playerType {
	case "", "basic":
		return NewBasicPlayer(), nil
	case "term":
		return NewTerminalPlayer(hints), nil
	case "random":
		return NewRandomPlayer(), nil
	default:
		return nil, fmt.Errorf("invalid player type %s", playerType)
	}
}
