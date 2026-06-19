package model

import (
	"strings"
	"testing"
)

func TestNewDeck_Size(t *testing.T) {
	t.Parallel()

	deck := NewDeck()

	const want = 52
	if len(deck.Cards) != want {
		t.Fatalf("expected %d cards, got %d", want, len(deck.Cards))
	}
}

func TestNewDeck_NoDuplicates(t *testing.T) {
	t.Parallel()

	deck := NewDeck()
	seen := make(map[Card]bool, len(deck.Cards))

	for _, c := range deck.Cards {
		if seen[c] {
			t.Fatalf("duplicate card in new deck: %s", c)
		}

		seen[c] = true
	}
}

func TestNewDeck_AllSuitsAndRanks(t *testing.T) {
	t.Parallel()

	deck := NewDeck()
	bySuit := make(map[Suit]int)
	byRank := make(map[Rank]int)

	for _, c := range deck.Cards {
		bySuit[c.Suit]++
		byRank[c.Rank]++
	}

	suits := []Suit{Hearts, Diamonds, Clubs, Spades}
	for _, s := range suits {
		if bySuit[s] != 13 {
			t.Errorf("suit %s: expected 13 cards, got %d", s.String(), bySuit[s])
		}
	}

	for r := RankTwo; r <= RankAce; r++ {
		if byRank[r] != 4 {
			t.Errorf("rank %s: expected 4 cards, got %d", r.String(), byRank[r])
		}
	}
}

func TestShuffle_PreservesSize(t *testing.T) {
	t.Parallel()

	deck := NewDeck()
	deck.Shuffle()

	const want = 52
	if len(deck.Cards) != want {
		t.Fatalf("expected %d cards after shuffle, got %d", want, len(deck.Cards))
	}
}

func TestShuffle_PreservesAllCards(t *testing.T) {
	t.Parallel()

	deck := NewDeck()
	before := make(map[Card]bool, len(deck.Cards))

	for _, c := range deck.Cards {
		before[c] = true
	}

	deck.Shuffle()

	for _, c := range deck.Cards {
		if !before[c] {
			t.Fatalf("card %s appeared after shuffle but was not in the original deck", c)
		}

		delete(before, c)
	}

	if len(before) != 0 {
		t.Fatalf("%d card(s) missing after shuffle", len(before))
	}
}

func TestShuffle_ChangesOrder(t *testing.T) {
	t.Parallel()

	// Two independently shuffled decks should (with overwhelming probability)
	// differ from each other and from the original order.
	d1 := NewDeck()
	d2 := NewDeck()
	original := NewDeck()

	d1.Shuffle()
	d2.Shuffle()

	sameAsOriginal1 := true
	sameAsOriginal2 := true

	for i := range original.Cards {
		if d1.Cards[i] != original.Cards[i] {
			sameAsOriginal1 = false
		}

		if d2.Cards[i] != original.Cards[i] {
			sameAsOriginal2 = false
		}
	}

	if sameAsOriginal1 && sameAsOriginal2 {
		t.Error("both shuffled decks have the same order as the original – shuffle may be broken")
	}
}

func TestDeal_RemovesCardsFromDeck(t *testing.T) {
	t.Parallel()

	deck := NewDeck()
	deck.Deal(5)

	if len(deck.Cards) != 47 {
		t.Fatalf("expected 47 remaining cards, got %d", len(deck.Cards))
	}
}

func TestDeal_ReturnsRequestedCount(t *testing.T) {
	t.Parallel()

	deck := NewDeck()
	dealt := deck.Deal(7)

	if len(dealt) != 7 {
		t.Fatalf("expected 7 dealt cards, got %d", len(dealt))
	}
}

func TestDeal_DealtCardsNotInDeck(t *testing.T) {
	t.Parallel()

	deck := NewDeck()
	dealt := deck.Deal(5)

	remaining := make(map[Card]bool, len(deck.Cards))
	for _, c := range deck.Cards {
		remaining[c] = true
	}

	for _, c := range dealt {
		if remaining[c] {
			t.Fatalf("dealt card %s is still in the deck", c)
		}
	}
}

func TestDeal_MoreThanAvailable(t *testing.T) {
	t.Parallel()

	deck := NewDeck()
	// Deal all 52 cards first, then try to deal more.
	deck.Deal(52)
	dealt := deck.Deal(5)

	if len(dealt) != 0 {
		t.Fatalf("expected 0 cards from empty deck, got %d", len(dealt))
	}
}

func TestDeal_ExactlyAvailable(t *testing.T) {
	t.Parallel()

	deck := NewDeck()
	dealt := deck.Deal(52)

	if len(dealt) != 52 {
		t.Fatalf("expected 52 cards, got %d", len(dealt))
	}

	if len(deck.Cards) != 0 {
		t.Fatalf("expected empty deck, got %d cards remaining", len(deck.Cards))
	}
}

func TestCardString_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		card Card
		want string
	}{
		{Card{RankAce, Spades}, "As"},
		{Card{RankKing, Hearts}, "Kh"},
		{Card{RankQueen, Diamonds}, "Qd"},
		{Card{RankJack, Clubs}, "Jc"},
		{Card{RankTen, Hearts}, "Th"},
		{Card{RankTwo, Spades}, "2s"},
		{Card{RankNine, Diamonds}, "9d"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := tt.card.String(); got != tt.want {
				t.Fatalf("Card{%v,%v}.String() = %q, want %q", tt.card.Rank, tt.card.Suit, got, tt.want)
			}
		})
	}
}

func TestRankString_AllRanks(t *testing.T) {
	t.Parallel()

	expected := []string{"2", "3", "4", "5", "6", "7", "8", "9", "T", "J", "Q", "K", "A"}

	rank := RankTwo
	for _, want := range expected {
		if got := rank.String(); got != want {
			t.Errorf("Rank(%d).String() = %q, want %q", rank, got, want)
		}

		rank++
	}
}

func TestSuitString_AllSuits(t *testing.T) {
	t.Parallel()

	cases := map[Suit]string{
		Hearts:   "h",
		Diamonds: "d",
		Clubs:    "c",
		Spades:   "s",
	}

	for suit, want := range cases {
		if got := suit.String(); got != want {
			t.Errorf("Suit(%d).String() = %q, want %q", suit, got, want)
		}
	}
}

func TestAllCards_StringLength(t *testing.T) {
	t.Parallel()

	// Every card in a full deck must produce a 2-character string.
	deck := NewDeck()
	for _, c := range deck.Cards {
		s := c.String()
		if len(s) != 2 {
			t.Errorf("card %+v: String() = %q (len %d), want length 2", c, s, len(s))
		}
	}
}

func TestAllCards_StringUnique(t *testing.T) {
	t.Parallel()

	deck := NewDeck()
	seen := make(map[string]bool, len(deck.Cards))

	for _, c := range deck.Cards {
		s := c.String()
		if seen[s] {
			t.Errorf("duplicate card string %q in full deck", s)
		}

		seen[s] = true
	}
}

func TestAllCards_StringContainsValidSuit(t *testing.T) {
	t.Parallel()

	validSuits := "hdcs"
	deck := NewDeck()

	for _, c := range deck.Cards {
		s := c.String()
		if !strings.ContainsRune(validSuits, rune(s[1])) {
			t.Errorf("card %+v: String() = %q has invalid suit character", c, s)
		}
	}
}
