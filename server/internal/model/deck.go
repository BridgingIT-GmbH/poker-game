package model

import (
	"fmt"
	"math/rand"
)

// Shuffle randomly re-orders the cards in-place using the global PRNG.
func (d *Deck) Shuffle() {
	rand.Shuffle(len(d.Cards), func(i, j int) {
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	})
}

// Deal removes and returns n cards from the top of the deck.
// If fewer than n cards remain, all remaining cards are returned.
func (d *Deck) Deal(n int) []Card {
	if n > len(d.Cards) {
		n = len(d.Cards)
	}

	top := len(d.Cards) - n
	dealt := make([]Card, n)
	copy(dealt, d.Cards[top:])
	d.Cards = d.Cards[:top]

	return dealt
}

// String returns a compact two-character representation of a card,
// e.g. "As" for Ace of Spades, "Th" for Ten of Hearts.
func (c Card) String() string {
	return c.Rank.String() + c.Suit.String()
}

// String returns the rank as a short string (2–9, T, J, Q, K, A).
func (r Rank) String() string {
	switch r {
	case RankTwo:
		return "2"
	case RankThree:
		return "3"
	case RankFour:
		return "4"
	case RankFive:
		return "5"
	case RankSix:
		return "6"
	case RankSeven:
		return "7"
	case RankEight:
		return "8"
	case RankNine:
		return "9"
	case RankTen:
		return "T"
	case RankJack:
		return "J"
	case RankQueen:
		return "Q"
	case RankKing:
		return "K"
	case RankAce:
		return "A"
	default:
		return fmt.Sprintf("?(%d)", r)
	}
}

// String returns the suit as a single lowercase letter (h, d, c, s).
func (s Suit) String() string {
	switch s {
	case Hearts:
		return "h"
	case Diamonds:
		return "d"
	case Clubs:
		return "c"
	case Spades:
		return "s"
	default:
		return "?"
	}
}
