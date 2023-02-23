package cards

import (
	"log"
	"math/rand"
	"sort"
	"strings"
	"time"

	pb "github.com/mpsalisbury/cards/pkg/proto"
	"golang.org/x/exp/slices"
)

type Cards []Card

func MakeDeck() Cards {
	d := make([]Card, 0, len(Suits)*len(Values))
	for _, s := range Suits {
		for _, v := range Values {
			d = append(d, Card{v, s})
		}
	}
	return d
}

func (cs Cards) Copy() Cards {
	cardsCopy := make([]Card, len(cs))
	copy(cardsCopy, cs)
	return cardsCopy
}

func (cs Cards) Equals(other Cards) bool {
	sorted := cs.Copy()
	sorted.Sort()
	otherSorted := other.Copy()
	otherSorted.Sort()
	return slices.Equal(sorted, otherSorted)
}

func (cs Cards) Contains(match func(Card) bool) bool {
	for _, c := range cs {
		if match(c) {
			return true
		}
	}
	return false
}

func (cs Cards) ContainsCard(c Card) bool {
	return cs.Contains(func(oc Card) bool { return oc == c })
}

func (cs Cards) ContainsSuit(s Suit) bool {
	return cs.Contains(func(c Card) bool { return c.Suit == s })
}

func (cs Cards) ContainsAny(other ...Card) bool {
	for _, c := range other {
		if cs.ContainsCard(c) {
			return true
		}
	}
	return false
}

func (cs Cards) Count(match func(Card) bool) int {
	count := 0
	for _, c := range cs {
		if match(c) {
			count++
		}
	}
	return count
}
func (cs Cards) CountSuit(s Suit) int {
	return cs.Count(func(c Card) bool { return c.Suit == s })
}

func (cs Cards) ToProto() *pb.GameState_Cards {
	return &pb.GameState_Cards{
		Cards: cs.Strings(),
	}
}

func ToProtos(tricks []Cards) []*pb.GameState_Cards {
	ts := []*pb.GameState_Cards{}
	for _, t := range tricks {
		ts = append(ts, t.ToProto())
	}
	return ts
}

func (cs Cards) Remove(c Card) Cards {
	for i, f := range cs {
		if f == c {
			copy(cs[i:], cs[i+1:])
			return cs[:len(cs)-1]
		}
	}
	return cs
}

func (cs Cards) Sort() {
	sort.Slice(cs, func(i, j int) bool {
		return cs[i].LessThan(cs[j])
	})
}

func (cs Cards) Shuffle() {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(cs), func(i, j int) { cs[i], cs[j] = cs[j], cs[i] })
}

// Returns a card that is better than all other cards according to the better func (is c1 better than c2).
// If no cards are present, fatal error.
func (cs Cards) GetExtreme(better func(c1, c2 Card) bool) Card {
	if len(cs) == 0 {
		log.Fatal("Can't get extreme for empty list of cards")
	}
	best := cs[0]
	for _, c := range cs {
		if better(c, best) {
			best = c
		}
	}
	return best
}
func (cs Cards) Lowest() Card {
	return cs.GetExtreme(func(c1, c2 Card) bool { return c1.Value < c2.Value })
}
func (cs Cards) Highest() Card {
	return cs.GetExtreme(func(c1, c2 Card) bool {
		return c1.Value > c2.Value
	})
}
func GetExtremeHand(hands []Cards, better func(c1, c2 Cards) bool) Cards {
	if len(hands) == 0 {
		log.Fatal("Can't get extreme for empty list of hands")
	}
	best := hands[0]
	for _, c := range hands {
		if better(c, best) {
			best = c
		}
	}
	return best
}

// This is to play a safe card if possible, or else the safest one left.
func (cs Cards) HighestUnderValueOrLowest(value Value) Card {
	cardsUnderValue := cs.Filter(func(c Card) bool { return c.Value < value })
	if len(cardsUnderValue) > 0 {
		return cardsUnderValue.Highest()
	}
	return cs.Lowest()
}

func (cs Cards) Filter(match func(c Card) bool) Cards {
	var filtered Cards
	for _, c := range cs {
		if match(c) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func (cs Cards) FilterBySuit(suits ...Suit) Cards {
	return cs.Filter(func(c Card) bool {
		for _, s := range suits {
			if c.Suit == s {
				return true
			}
		}
		return false
	})
}
func (cs Cards) FilterLE(value Value) Cards {
	return cs.Filter(func(c Card) bool { return c.Value <= value })
}
func (cs Cards) FilterGE(value Value) Cards {
	return cs.Filter(func(c Card) bool { return c.Value >= value })
}

// Highest card of same suit as first.
func (cs Cards) LeadingCardOfTrick() Card {
	if len(cs) == 0 {
		log.Fatalf("Can't find leading card of empty trick")
	}
	return cs.FilterBySuit(cs[0].Suit).Highest()
}

func Combine(cardss ...Cards) Cards {
	var cs Cards
	for _, cards := range cardss {
		for _, c := range cards {
			cs = append(cs, c)
		}
	}
	return cs
}

func (cs Cards) SplitBySuit() map[Suit]Cards {
	cbs := make(map[Suit]Cards)
	for _, c := range cs {
		cbs[c.Suit] = append(cbs[c.Suit], c)
	}
	return cbs
}

func (cs Cards) Strings() []string {
	cardStrings := []string{}
	for _, c := range cs {
		cardStrings = append(cardStrings, c.String())
	}
	return cardStrings
}

func (cs Cards) String() string {
	cardStrings := cs.Strings()
	return strings.Join(cardStrings, " ")
}

func (cs Cards) HandString() string {
	cbs := cs.SplitBySuit()
	suitStrings := []string{}
	for _, s := range Suits {
		scs := cbs[s]
		if len(scs) > 0 {
			scs.Sort()
			suitStrings = append(suitStrings, scs.String())
		}
	}
	return strings.Join(suitStrings, "   ")
}

func ParseCards(cs []string) (Cards, error) {
	var cards Cards
	for _, c := range cs {
		card, err := ParseCard(c)
		if err != nil {
			return Cards{}, err
		}
		cards = append(cards, card)
	}
	return cards, nil
}

func Deal(numHands int) []Cards {
	hs := make([]Cards, numHands)
	d := MakeDeck()
	d.Shuffle()
	for i, c := range d {
		hi := i % numHands
		hs[hi] = append(hs[hi], c)
	}
	for _, h := range hs {
		h.Sort()
	}
	return hs
}
