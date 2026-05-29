package model

// Rank represents the rank of a card (2 through Ace).
type Rank uint8

const (
	RankTwo Rank = iota + 2
	RankThree
	RankFour
	RankFive
	RankSix
	RankSeven
	RankEight
	RankNine
	RankTen
	RankJack
	RankQueen
	RankKing
	RankAce
)

// Suit represents the suit of a card.
type Suit uint8

const (
	Hearts Suit = iota
	Diamonds
	Clubs
	Spades
)

// Card represents a single playing card.
type Card struct {
	Rank Rank
	Suit Suit
}

// BettingRound represents the current stage of betting.
type BettingRound int

const (
	PreFlop BettingRound = iota
	Flop
	Turn
	River
	Showdown
)

// Player represents a participant in the game.
type Player struct {
	ID               int
	Name             string
	HoleCards        [2]Card // The 2 private cards dealt to this player
	Chips            int64   // Total chips remaining
	CurrentBet       int64   // Amount bet in the current betting round
	TotalBetThisHand int64   // Total bet across all betting rounds this hand
	IsActive         bool    // True if still eligible to win the pot
	IsAllIn          bool    // True if player has bet all remaining chips
	HasFolded        bool    // True if player folded this hand
	HasActed         bool    // True if player has acted in current betting round
}

// Deck represents the remaining cards in the deck.
type Deck struct {
	Cards []Card
}

// Table represents the shared game state.
type Table struct {
	Players          []*Player
	Deck             Deck
	CommunityCards   []Card       // 0 cards pre-flop, 3 after flop, 4 after turn, 5 after river
	Pot              int64        // Total chips in the pot(s)
	CurrentBet       int64        // The amount the current player must match to stay in
	DealerPosition   int          // Index of the dealer button
	SmallBlindPos    int          // Index of small blind player
	BigBlindPos      int          // Index of big blind player
	CurrentToActPos  int          // Index of player who must act next
	SmallBlindAmount int64        // Small blind amount for this hand
	BigBlindAmount   int64        // Big blind amount for this hand
	BettingRound     BettingRound // Current stage: PreFlop, Flop, Turn, River, or Showdown
	NumPlayersActive int          // Number of players still eligible to win
	NumPlayersAllIn  int          // Number of players all-in
}

// Game represents the complete state of a Texas Hold 'Em game session
// type Game struct {
//	Table              *Table
//	HandNumber         int     // Which hand is being played
//	IsHandInProgress   bool    // True while hand is being played
//	IsGameOver         bool    // True when game session ends
//	RequiredPlayers    int     // Minimum players to start a hand
//	MinBuyIn           int64   // Minimum chip buy-in
//	MaxBuyIn           int64   // Maximum chip buy-in (0 = no limit)
//	SmallBlindSchedule []int64 // Blind levels for tournament or timed game
//	CurrentBlindLevel  int     // Index into SmallBlindSchedule
// }
