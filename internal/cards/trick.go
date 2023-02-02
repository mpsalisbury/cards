package cards

type Trick struct {
	CardsInOrder Cards
	PlayerByCard map[Card]int
}

func NewTrick() *Trick {
	return &Trick{[]Card{}, make(map[Card]int)}
}

func (t *Trick) String() string {
	return t.CardsInOrder.String()
}

func (t *Trick) Add(p Player, c Card) {
	t.CardsInOrder = append(t.CardsInOrder, c)
	t.PlayerByCard[c] = p.Id
}

// Returns false if no such card.
func (t *Trick) LeadSuit() (Suit, bool) {
	if len(t.CardsInOrder) > 0 {
		return t.CardsInOrder[0].Suit, true
	}
	return Clubs, false
}
