package player

import (
	"flag"
	"fmt"

	"github.com/mpsalisbury/cards/pkg/client"
)

func enumFlag(target *string, name string, safelist []string, usage string) {
	usageWithValues := fmt.Sprintf("%s, must be one of %v", usage, safelist)
	flag.Func(name, usageWithValues, func(flagValue string) error {
		for _, allowedValue := range safelist {
			if flagValue == allowedValue {
				*target = flagValue
				return nil
			}
		}
		return fmt.Errorf("must be one of %v", safelist)
	})
}

// Creates a flag for specifying the player type to use.
func AddPlayerFlag(target *string, name string) {
	enumFlag(target, name, []string{"basic", "term", "random"}, "Type of player logic to use")
}

// Constructs a player from a player flag value.
func NewPlayerFromFlag(playerType string) (client.GameCallbacks, error) {
	switch playerType {
	case "", "basic":
		return NewBasicPlayer(), nil
	case "term":
		return NewTerminalPlayer(), nil
	case "random":
		return NewRandomPlayer(), nil
	default:
		return nil, fmt.Errorf("invalid player type %s", playerType)
	}
}
