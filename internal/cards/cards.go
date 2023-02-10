package cards

import (
	"math/rand"
	"sort"
	"strings"
	"time"
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

func Combine(cardss ...Cards) Cards {
	cs := []Card{}
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
