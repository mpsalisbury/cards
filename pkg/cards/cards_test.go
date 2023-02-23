package cards

import (
	"testing"
)

func absDiff(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

func TestDeal(t *testing.T) {
	fullDeck := MakeDeck()
	for numHands := 2; numHands <= 6; numHands++ {
		hands := Deal(numHands)
		if len(hands) != numHands {
			t.Errorf("Deal(%d)=%d hands, want %d", numHands, len(hands), numHands)
		}
		// Make sure all hands have number of cards within 1 of each other.
		cardsPerHand := len(hands[0])
		for _, h := range hands {
			numCards := len(h)
			if absDiff(numCards, cardsPerHand) > 1 {
				t.Errorf("Deal(%d): Expected each hand to have similar count. Found numCards = %d vs %d",
					numHands, numCards, cardsPerHand)
			}
		}
		// Make sure all cards were dealt.
		allCards := Combine(hands...)
		allCards.Sort()
		if allCards.String() != fullDeck.String() {
			t.Errorf("Deal(%d)='%s', expected full deck", numHands, allCards)
		}
	}
}

func TestFilterBySuit(t *testing.T) {
	tests := []struct {
		name  string
		hand  Cards
		suits []Suit
		want  Cards
	}{
		{
			name:  "Just clubs",
			hand:  Cards{C2c, C3h, C4s, C5d},
			suits: []Suit{Clubs},
			want:  Cards{C2c},
		},
		{
			name:  "Just hearts",
			hand:  Cards{C2c, C3h, C4s, C5d},
			suits: []Suit{Hearts},
			want:  Cards{C3h},
		},
		{
			name:  "Just spades",
			hand:  Cards{C2c, C3h, C4s, C5d},
			suits: []Suit{Spades},
			want:  Cards{C4s},
		},
		{
			name:  "Just diamonds",
			hand:  Cards{C2c, C3h, C4s, C5d},
			suits: []Suit{Diamonds},
			want:  Cards{C5d},
		},
		{
			name:  "Filter all out",
			hand:  Cards{C2c, C3c, C4s, C5d},
			suits: []Suit{Hearts},
			want:  Cards{},
		},
		{
			name:  "Start with empty hand",
			hand:  Cards{},
			suits: []Suit{Hearts},
			want:  Cards{},
		},
		{
			name:  "Filter multiple suits",
			hand:  Cards{C2c, C3h, C4s, C5d},
			suits: []Suit{Hearts, Spades},
			want:  Cards{C3h, C4s},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.hand.FilterBySuit(tc.suits...)
			if !got.Equals(tc.want) {
				t.Errorf("FilterBySuit(%s,%v)=%s, want %s", tc.hand, tc.suits, got, tc.want)
			}
		})
	}
}

func TestFilterLE(t *testing.T) {
	tests := []struct {
		name  string
		hand  Cards
		value Value
		want  Cards
	}{
		{
			name:  "<= Ace",
			hand:  Cards{C2c, C3h, C4s, C5d, Cjh, Cqh, Ckh, Cah},
			value: Ace,
			want:  Cards{C2c, C3h, C4s, C5d, Cjh, Cqh, Ckh, Cah},
		},
		{
			name:  "<= Jack",
			hand:  Cards{C2c, C3h, C4s, C5d, Cjh, Cqh, Ckh, Cah},
			value: Jack,
			want:  Cards{C2c, C3h, C4s, C5d, Cjh},
		},
		{
			name:  "<= Two",
			hand:  Cards{C2c, C3h, C4s, C5d, Cjh, Cqh, Ckh, Cah},
			value: Two,
			want:  Cards{C2c},
		},
		{
			name:  "Filter all out",
			hand:  Cards{C8c, C9c, Cts, Cjd},
			value: Seven,
			want:  Cards{},
		},
		{
			name:  "Start with empty hand",
			hand:  Cards{},
			value: Seven,
			want:  Cards{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.hand.FilterLE(tc.value)
			if !got.Equals(tc.want) {
				t.Errorf("FilterLE(%s,%v)=%s, want %s", tc.hand, tc.value, got, tc.want)
			}
		})
	}
}

func TestFilterGE(t *testing.T) {
	tests := []struct {
		name  string
		hand  Cards
		value Value
		want  Cards
	}{
		{
			name:  ">= Two",
			hand:  Cards{C2c, C3h, C4s, C5d, Cjh, Cqh, Ckh, Cah},
			value: Two,
			want:  Cards{C2c, C3h, C4s, C5d, Cjh, Cqh, Ckh, Cah},
		},
		{
			name:  ">= Jack",
			hand:  Cards{C2c, C3h, C4s, C5d, Cjh, Cqh, Ckh, Cah},
			value: Jack,
			want:  Cards{Cjh, Cqh, Ckh, Cah},
		},
		{
			name:  ">= Ace",
			hand:  Cards{C2c, C3h, C4s, C5d, Cjh, Cqh, Ckh, Cah},
			value: Ace,
			want:  Cards{Cah},
		},
		{
			name:  "Filter all out",
			hand:  Cards{C8c, C9c, Cts, Cjd},
			value: Queen,
			want:  Cards{},
		},
		{
			name:  "Start with empty hand",
			hand:  Cards{},
			value: Seven,
			want:  Cards{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.hand.FilterGE(tc.value)
			if !got.Equals(tc.want) {
				t.Errorf("FilterLE(%s,%v)=%s, want %s", tc.hand, tc.value, got, tc.want)
			}
		})
	}
}

func TestLowest(t *testing.T) {
	tests := []struct {
		hand      Cards
		wantOneOf Cards
	}{
		{
			hand:      Cards{C2c, C3h, C4s, C5d, Cjh, Cqh, Ckh, Cah},
			wantOneOf: Cards{C2c},
		},
		{
			hand:      Cards{C3h, C4s, C5d, C2c, Ckh, Cah},
			wantOneOf: Cards{C2c},
		},
		{
			hand:      Cards{C9h, Cjs, C5d, Ckh, Cah},
			wantOneOf: Cards{C5d},
		},
		{
			hand:      Cards{Cjs},
			wantOneOf: Cards{Cjs},
		},
	}
	for _, tc := range tests {
		got := tc.hand.Lowest()
		if !tc.wantOneOf.ContainsCard(got) {
			t.Errorf("Lowest(%s)=%s, wantOneOf %s", tc.hand, got, tc.wantOneOf)
		}
	}
}

func TestHighest(t *testing.T) {
	tests := []struct {
		hand      Cards
		wantOneOf Cards
	}{
		{
			hand:      Cards{C2c, C3h, C4s, C5d, Cjh, Cqh, Ckh, Cah},
			wantOneOf: Cards{Cah},
		},
		{
			hand:      Cards{C3h, Cah, C4s, C5d, Cqh, Ckh},
			wantOneOf: Cards{Cah},
		},
		{
			hand:      Cards{C9h, Cjs, C5d, Cth, C9d},
			wantOneOf: Cards{Cjs},
		},
		{
			hand:      Cards{Cjs},
			wantOneOf: Cards{Cjs},
		},
	}
	for _, tc := range tests {
		got := tc.hand.Highest()
		if !tc.wantOneOf.ContainsCard(got) {
			t.Errorf("Highest(%s)=%s, wantOneOf %s", tc.hand, got, tc.wantOneOf)
		}
	}
}

func TestHighestUnderValueOrLowest(t *testing.T) {
	tests := []struct {
		hand  Cards
		value Value
		want  Card
	}{
		{
			hand:  Cards{C3h, C4h, C7h, Cth, Ckh, Cah, C2h},
			value: Nine,
			want:  C7h,
		},
		{
			hand:  Cards{C9h, Cth, Ckh, Cah},
			value: Nine,
			want:  C9h,
		},
		{
			hand:  Cards{Ckh, Cah, Cqh},
			value: Eight,
			want:  Cqh,
		},
		{
			hand:  Cards{Cjs},
			value: Queen,
			want:  Cjs,
		},
	}
	for _, tc := range tests {
		got := tc.hand.HighestUnderValueOrLowest(tc.value)
		if got != tc.want {
			t.Errorf("HighestUnderValueOrLowest(%s,%s)=%s, want %s", tc.hand, tc.value, got, tc.want)
		}
	}
}

func TestLeadingCardOfTrick(t *testing.T) {
	tests := []struct {
		trick Cards
		want  Card
	}{
		{
			trick: Cards{C3h, Cqh, C9h, C7h},
			want:  Cqh,
		},
		{
			trick: Cards{C3s, Cqh, C9h, C7h},
			want:  C3s,
		},
		{
			trick: Cards{Cth, Cqs, C9h, C7h},
			want:  Cth,
		},
	}
	for _, tc := range tests {
		got := tc.trick.LeadingCardOfTrick()
		if got != tc.want {
			t.Errorf("LeadingCardOfTrick(%s)=%s, want %s", tc.trick, got, tc.want)
		}
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		hand     Cards
		contains Cards
		want     bool
	}{
		{
			hand:     Cards{C3h, Cqh, C9h, C7h},
			contains: Cards{C3h},
			want:     true,
		},
		{
			hand:     Cards{C3h, Cqh, C9h, C7h},
			contains: Cards{C7h},
			want:     true,
		},
		{
			hand:     Cards{C3h, Cqh, C9h, C7h},
			contains: Cards{C9s},
			want:     false,
		},
		{
			hand:     Cards{C3h, Cqh, C9h, C7h},
			contains: Cards{C9s, C7h},
			want:     true,
		},
		{
			hand:     Cards{C3h, Cqh, C9h, C7h},
			contains: Cards{C9s, C7d, C3c, Cqd},
			want:     false,
		},
		{
			hand:     Cards{C3h, Cqh, C9h, C7h},
			contains: Cards{},
			want:     false,
		},
	}
	for _, tc := range tests {
		got := tc.hand.ContainsAny(tc.contains...)
		if got != tc.want {
			t.Errorf("ContainsAny(%s, %s)=%t, want %t", tc.hand, tc.contains, got, tc.want)
		}
	}
}

func TestContainsSuit(t *testing.T) {
	tests := []struct {
		hand Cards
		suit Suit
		want bool
	}{
		{
			hand: Cards{C3h, Cqh, C9h, C7h},
			suit: Hearts,
			want: true,
		},
		{
			hand: Cards{C3h, Cqh, C9h, C7h},
			suit: Spades,
			want: false,
		},
		{
			hand: Cards{C3h, Cqs, C9c, C7h},
			suit: Diamonds,
			want: false,
		},
		{
			hand: Cards{},
			suit: Spades,
			want: false,
		},
	}
	for _, tc := range tests {
		got := tc.hand.ContainsSuit(tc.suit)
		if got != tc.want {
			t.Errorf("ContainsSuit(%s, %s)=%t, want %t", tc.hand, tc.suit, got, tc.want)
		}
	}
}
