package model

// NewCard creates a new card.
func NewCard(rank Rank, suit Suit) Card {
	return Card{Rank: rank, Suit: suit}
}

// NewDeck creates a full, unshuffled 52-card deck.
func NewDeck() Deck {
	const deckSize = 52

	deck := Deck{Cards: make([]Card, 0, deckSize)}

	for suit := Hearts; suit <= Spades; suit++ {
		for rank := RankTwo; rank <= RankAce; rank++ {
			deck.Cards = append(deck.Cards, Card{Rank: rank, Suit: suit})
		}
	}

	return deck
}

// NewPlayer creates a new player.
func NewPlayer(id int, name string, initialChips int64) *Player {
	return &Player{
		ID:    id,
		Name:  name,
		Chips: initialChips,
	}
}

// NewTable creates a new table with players.
func NewTable(players []*Player, smallBlind, bigBlind int64) *Table {
	const boardLength = 5

	return &Table{
		Players:          players,
		Deck:             NewDeck(),
		CommunityCards:   make([]Card, 0, boardLength),
		SmallBlindAmount: smallBlind,
		BigBlindAmount:   bigBlind,
		BettingRound:     PreFlop,
	}
}

// NewGame creates a new game session
// func NewGame(players []*Player, smallBlind, bigBlind int64) *Game {
//	return &Game{
//		Table:           NewTable(players, smallBlind, bigBlind),
//		HandNumber:      0,
//		RequiredPlayers: 2,
//		MinBuyIn:        smallBlind * 20,
//		MaxBuyIn:        0,
//	}
// }

// GetActivePlayer returns the player who must currently act.
func (t *Table) GetActivePlayer() *Player {
	return t.Players[t.CurrentToActPos]
}

// CountActivePlayers returns how many players are still eligible to win.
func (t *Table) CountActivePlayers() int {
	count := 0

	for _, p := range t.Players {
		if p.IsActive && !p.HasFolded {
			count++
		}
	}

	return count
}

// GetHoleCards returns a player's hole cards safely.
func (p *Player) GetHoleCards() [2]Card {
	return p.HoleCards
}

// GetAllPlayerCards returns all 7 cards available to a player (hole + community).
func (p *Player) GetAllPlayerCards(communityCards []Card) []Card {
	const holeAndCommunity = 7

	allCards := make([]Card, 0, holeAndCommunity)
	allCards = append(allCards, p.HoleCards[:]...)
	allCards = append(allCards, communityCards...)

	return allCards
}

// ResetForNewBettingRound resets player betting state for a new round.
func (t *Table) ResetForNewBettingRound() {
	for _, p := range t.Players {
		p.CurrentBet = 0
		p.HasActed = false
	}

	t.CurrentBet = 0
}

// ResetPlayersForNewHand resets all player state for a new hand.
func ResetPlayersForNewHand(players []*Player) {
	for _, p := range players {
		p.HoleCards = [2]Card{}
		p.CurrentBet = 0
		p.TotalBetThisHand = 0
		p.IsAllIn = false
		p.HasFolded = false
		p.HasActed = false
	}
}
