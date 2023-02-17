package cards

import "testing"

func TestParseValidCard(t *testing.T) {
	tests := []struct {
		c    string
		want Card
	}{
		{"2c", Card{Two, Clubs}},
		{"3c", Card{Three, Clubs}},
		{"4c", Card{Four, Clubs}},
		{"5c", Card{Five, Clubs}},
		{"6c", Card{Six, Clubs}},
		{"7c", Card{Seven, Clubs}},
		{"8c", Card{Eight, Clubs}},
		{"9c", Card{Nine, Clubs}},
		{"tc", Card{Ten, Clubs}},
		{"jc", Card{Jack, Clubs}},
		{"qc", Card{Queen, Clubs}},
		{"kc", Card{King, Clubs}},
		{"ac", Card{Ace, Clubs}},
		{"TS", Card{Ten, Spades}},
		{"jH", Card{Jack, Hearts}},
		{"ad", Card{Ace, Diamonds}},
	}
	for _, tc := range tests {
		got, err := ParseCard(tc.c)
		if err != nil {
			t.Errorf("ParseCard(%s)=error(%s), want %s", tc.c, err, tc.want)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseCard(%s)=%s, want %s", tc.c, got, tc.want)
		}
	}
}

func TestParseInvalidCard(t *testing.T) {
	tests := []string{"xc", "7x", "2cc", "22c", "", "5"}
	for _, tc := range tests {
		got, err := ParseCard(tc)
		if err == nil {
			t.Errorf("ParseCard(%s)=%s, want err", tc, got)
		}
	}
}
