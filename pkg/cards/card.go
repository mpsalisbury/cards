package cards

import (
	"fmt"
	"strings"
)

// A card's suit.
type Suit int8

const (
	Clubs Suit = iota
	Hearts
	Spades
	Diamonds
)

var Suits = []Suit{
	Clubs,
	Hearts,
	Spades,
	Diamonds,
}

func (s Suit) String() string {
	switch s {
	case Clubs:
		return "c"
	case Hearts:
		return "h"
	case Spades:
		return "s"
	case Diamonds:
		return "d"
	}
	panic("Unknown Suit")
}

func parseSuit(s string) (Suit, error) {
	switch strings.ToLower(s) {
	case "c":
		return Clubs, nil
	case "h":
		return Hearts, nil
	case "s":
		return Spades, nil
	case "d":
		return Diamonds, nil
	}
	return Clubs, fmt.Errorf("no such suit '%s'", s)
}

// A card's value: 2-9,T,J,Q,K,A.
type Value int8

const (
	Two Value = iota
	Three
	Four
	Five
	Six
	Seven
	Eight
	Nine
	Ten
	Jack
	Queen
	King
	Ace
)

var Values = []Value{
	Two,
	Three,
	Four,
	Five,
	Six,
	Seven,
	Eight,
	Nine,
	Ten,
	Jack,
	Queen,
	King,
	Ace,
}

func (v Value) String() string {
	switch v {
	case Two:
		return "2"
	case Three:
		return "3"
	case Four:
		return "4"
	case Five:
		return "5"
	case Six:
		return "6"
	case Seven:
		return "7"
	case Eight:
		return "8"
	case Nine:
		return "9"
	case Ten:
		return "T"
	case Jack:
		return "J"
	case Queen:
		return "Q"
	case King:
		return "K"
	case Ace:
		return "A"
	}
	panic("Unknown Value")
}

func parseValue(v string) (Value, error) {
	switch strings.ToLower(v) {
	case "2":
		return Two, nil
	case "3":
		return Three, nil
	case "4":
		return Four, nil
	case "5":
		return Five, nil
	case "6":
		return Six, nil
	case "7":
		return Seven, nil
	case "8":
		return Eight, nil
	case "9":
		return Nine, nil
	case "t":
		return Ten, nil
	case "j":
		return Jack, nil
	case "q":
		return Queen, nil
	case "k":
		return King, nil
	case "a":
		return Ace, nil
	}
	return Two, fmt.Errorf("no such value '%s'", v)
}

type Card struct {
	Value
	Suit
}

func (c Card) String() string {
	return c.Value.String() + c.Suit.String()
}

func ParseCard(c string) (Card, error) {
	if len(c) != 2 {
		return Card{}, fmt.Errorf("can't parse card '%s'", c)
	}
	v, verr := parseValue(c[0:1])
	s, serr := parseSuit(c[1:2])
	if verr != nil || serr != nil {
		return Card{}, fmt.Errorf("can't parse card '%s'", c)
	}
	return Card{v, s}, nil
}

func (c1 Card) LessThan(c2 Card) bool {
	if c1.Suit == c2.Suit {
		return c1.Value < c2.Value
	}
	return c1.Suit < c2.Suit
}
