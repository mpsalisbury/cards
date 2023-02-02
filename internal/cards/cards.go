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

func Sort(d Cards) {
	sort.Slice(d, func(i, j int) bool {
		return d[i].LessThan(d[j])
	})
}

func Shuffle(d Cards) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(d), func(i, j int) { d[i], d[j] = d[j], d[i] })
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

func (cs Cards) String() string {
	cardStrings := []string{}
	for _, c := range cs {
		cardStrings = append(cardStrings, c.String())
	}
	return strings.Join(cardStrings, " ")
}

func (cs Cards) HandString() string {
	cbs := cs.SplitBySuit()
	suitStrings := []string{}
	for _, s := range Suits {
		scs := cbs[s]
		if len(scs) > 0 {
			Sort(scs)
			suitStrings = append(suitStrings, scs.String())
		}
	}
	return strings.Join(suitStrings, "   ")
}

func Deal(numHands int) []Cards {
	hs := make([]Cards, numHands)
	d := MakeDeck()
	Shuffle(d)
	for i, c := range d {
		hi := i % numHands
		hs[hi] = append(hs[hi], c)
	}
	for _, h := range hs {
		Sort(h)
	}
	return hs
}
