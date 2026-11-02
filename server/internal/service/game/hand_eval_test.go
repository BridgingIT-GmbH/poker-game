//nolint:dupl
package game

import (
	"testing"

	. "github.com/BridgingIT-GmbH/poker-game/server/internal/model"
)

func c(rank Rank, suit Suit) Card {
	return Card{Rank: rank, Suit: suit}
}

func makePlayer(id int, name string, holeCard1 Card, holeCard2 Card) *Player {
	return &Player{
		ID:        id,
		Name:      name,
		HoleCards: [2]Card{holeCard1, holeCard2},
		IsActive:  true,
	}
}

func makeTable(card1 Card, card2 Card, card3 Card, card4 Card, card5 Card) *Table {
	return &Table{
		CommunityCards: []Card{card1, card2, card3, card4, card5},
	}
}

func setupTableState(table *Table, players ...*Player) *Table {
	table.Players = players

	return table
}

func runTestEvaluation(t *testing.T, table *Table, expectedWinner *Player) map[*Player]BestHand {
	winners, bestHands, err := EvaluateShowdown(table)
	if err != nil {
		t.Fatalf("EvaluateShowdown returned error: %v", err)
	}

	if len(winners) != 1 || winners[0] != expectedWinner {
		t.Fatalf("expected %v to win, got winners: %+v", expectedWinner.Name, winners)
	}

	return bestHands
}

// --------------------
// Common cases
// --------------------

func TestEvaluateShowdown_Common_OnePairBeatsHighCard(t *testing.T) {
	t.Parallel()

	// Pair of 2s
	p1 := makePlayer(
		1,
		"P1",
		c(RankTwo, Clubs),
		c(RankKing, Diamonds),
	)

	// High card only
	p2 := makePlayer(
		2,
		"P2",
		c(RankJack, Hearts),
		c(RankTen, Diamonds),
	)

	table := setupTableState(makeTable(
		c(RankTwo, Hearts),
		c(RankSeven, Diamonds),
		c(RankNine, Clubs),
		c(RankQueen, Hearts),
		c(RankThree, Spades),
	), p1, p2)

	bestHands := runTestEvaluation(t, table, p1)

	if bestHands[p1].Value.Category != OnePair {
		t.Fatalf("expected P1 to have OnePair, got %v", bestHands[p1].Value.Category)
	}

	if bestHands[p2].Value.Category != HighCard {
		t.Fatalf("expected P2 to have HighCard, got %v", bestHands[p2].Value.Category)
	}
}

func TestEvaluateShowdown_Common_FlushBeatsStraightAndTwoPair(t *testing.T) {
	t.Parallel()

	// P1: flush in hearts (J high)
	p1 := makePlayer(
		1,
		"Flush",
		c(RankJack, Hearts),
		c(RankFour, Hearts),
	)

	// P2: straight 2-3-4-5-6
	p2 := makePlayer(
		2,
		"Straight",
		c(RankSix, Clubs),
		c(RankFour, Clubs),
	)

	// P3: two pair (K and 3)
	p3 := makePlayer(
		3,
		"TwoPair",
		c(RankKing, Hearts), // pairs K♦ on board
		c(RankThree, Diamonds),
	)

	// Board:
	//  2♥ 5♥ 9♥ K♦ 3♠
	table := setupTableState(makeTable(
		c(RankTwo, Hearts),
		c(RankFive, Hearts),
		c(RankNine, Hearts),
		c(RankKing, Diamonds),
		c(RankThree, Spades),
	), p1, p2, p3)

	bestHands := runTestEvaluation(t, table, p1)

	if bestHands[p1].Value.Category != Flush {
		t.Fatalf("expected P1 to have Flush, got %v", bestHands[p1].Value.Category)
	}

	if bestHands[p2].Value.Category != Straight {
		t.Fatalf("expected P2 to have Straight, got %v", bestHands[p2].Value.Category)
	}

	if bestHands[p3].Value.Category != TwoPair {
		t.Fatalf("expected P3 to have TwoPair, got %v", bestHands[p3].Value.Category)
	}
}

func TestEvaluateShowdown_Common_KickerDecidesSamePairFromBoard(t *testing.T) {
	t.Parallel()

	// P1: K kicker (AAK87)
	p1 := makePlayer(
		1,
		"P1-KickerK",
		c(RankKing, Clubs),
		c(RankThree, Hearts),
	)

	// P2: Q kicker (AAQ87)
	p2 := makePlayer(
		2,
		"P2-KickerQ",
		c(RankQueen, Clubs),
		c(RankFour, Hearts),
	)

	// Board: A♥ A♦ 7♣ 8♦ 2♠
	table := setupTableState(makeTable(
		c(RankAce, Hearts),
		c(RankAce, Diamonds),
		c(RankSeven, Clubs),
		c(RankEight, Diamonds),
		c(RankTwo, Spades),
	), p1, p2)

	bestHands := runTestEvaluation(t, table, p1)

	if bestHands[p1].Value.Category != OnePair || bestHands[p2].Value.Category != OnePair {
		t.Fatalf("expected both players to have OnePair, got P1=%v, P2=%v",
			bestHands[p1].Value.Category, bestHands[p2].Value.Category)
	}

	if bestHands[p1].Value.Ranks[1] != RankKing {
		t.Fatalf("expected P1 top kicker to be K, got %v", bestHands[p1].Value.Ranks[1])
	}

	if bestHands[p2].Value.Ranks[1] != RankQueen {
		t.Fatalf("expected P2 top kicker to be Q, got %v", bestHands[p2].Value.Ranks[1])
	}
}

// --------------------
// Edge cases
// --------------------

// 1) Ace-low straight vs higher straight.
func TestEvaluateShowdown_Edge_AceLowStraightLosesToHigherStraight(t *testing.T) {
	t.Parallel()

	// P1: A♠ K♦ => wheel A-2-3-4-5 (5-high straight)
	p1 := makePlayer(
		1,
		"Wheel",
		c(RankAce, Spades),
		c(RankKing, Diamonds),
	)

	// P2: 6♥ 7♣ => 3-4-5-6-7 (7-high straight), which beats the wheel.
	p2 := makePlayer(
		2,
		"SevenHighStraight",
		c(RankSix, Hearts),
		c(RankSeven, Clubs),
	)

	// Board: 2♣ 3♦ 4♥ 5♠ 9♣
	table := setupTableState(makeTable(
		c(RankTwo, Clubs),
		c(RankThree, Diamonds),
		c(RankFour, Hearts),
		c(RankFive, Spades),
		c(RankNine, Clubs),
	), p1, p2)

	bestHands := runTestEvaluation(t, table, p2)

	if bestHands[p1].Value.Category != Straight || bestHands[p2].Value.Category != Straight {
		t.Fatalf("expected both to have Straight, got P1=%v, P2=%v",
			bestHands[p1].Value.Category, bestHands[p2].Value.Category)
	}

	// Wheel should be encoded with high card 5.
	if bestHands[p1].Value.Ranks[0] != RankFive {
		t.Fatalf("expected wheel straight high card to be 5, got %v", bestHands[p1].Value.Ranks[0])
	}

	if bestHands[p2].Value.Ranks[0] != RankSeven {
		t.Fatalf("expected P2 straight high card to be 7, got %v", bestHands[p2].Value.Ranks[0])
	}
}

// 2) Multiple winners tie exactly (board flush plays).
func TestEvaluateShowdown_Edge_MultipleWinnersTieExactly(t *testing.T) {
	t.Parallel()

	p1 := makePlayer(
		1,
		"P1",
		c(RankThree, Clubs),
		c(RankFour, Diamonds),
	)
	p2 := makePlayer(
		2,
		"P2",
		c(RankSeven, Clubs),
		c(RankNine, Diamonds),
	)
	p3 := makePlayer(
		3,
		"P3",
		c(RankAce, Clubs),
		c(RankAce, Diamonds),
	)

	// Board: 2♥ 5♥ 8♥ J♥ K♥ -> nut flush on board, everyone shares.
	table := setupTableState(makeTable(
		c(RankTwo, Hearts),
		c(RankFive, Hearts),
		c(RankEight, Hearts),
		c(RankJack, Hearts),
		c(RankKing, Hearts),
	), p1, p2, p3)

	winners, bestHands, err := EvaluateShowdown(table)
	if err != nil {
		t.Fatalf("EvaluateShowdown returned error: %v", err)
	}

	if len(winners) != 3 {
		t.Fatalf("expected all 3 players to tie, got winners: %+v", winners)
	}

	for _, p := range []*Player{p1, p2, p3} {
		if bestHands[p].Value.Category != Flush {
			t.Fatalf("expected player %d to have Flush, got %v", p.ID, bestHands[p].Value.Category)
		}
	}
}

// 3) Board high-card plays: P2's higher kicker wins.
func TestEvaluateShowdown_Edge_BoardPlaysHighCard(t *testing.T) {
	t.Parallel()

	p1 := makePlayer(
		1,
		"P1",
		c(RankThree, Clubs),
		c(RankFour, Spades),
	)
	p2 := makePlayer(
		2,
		"P2",
		c(RankSeven, Clubs),
		c(RankNine, Spades),
	)

	// Board: 2♥ 5♦ 8♣ J♥ K♦
	table := setupTableState(makeTable(
		c(RankTwo, Hearts),
		c(RankFive, Diamonds),
		c(RankEight, Clubs),
		c(RankJack, Hearts),
		c(RankKing, Diamonds),
	), p1, p2)

	winners, bestHands, err := EvaluateShowdown(table)
	if err != nil {
		t.Fatalf("EvaluateShowdown returned error: %v", err)
	}

	if len(winners) != 1 || winners[0] != p2 {
		t.Fatalf("expected P2 to win by kicker, got winners: %+v", winners)
	}

	for _, p := range []*Player{p1, p2} {
		if bestHands[p].Value.Category != HighCard {
			t.Fatalf("expected HighCard category, got %v for player %d",
				bestHands[p].Value.Category, p.ID)
		}
	}
}

// 4) Folded / inactive players are ignored.
func TestEvaluateShowdown_Edge_FoldedAndInactiveIgnored(t *testing.T) {
	t.Parallel()

	// P1: best hand but folded -> must NOT win
	p1 := makePlayer(
		1,
		"FoldedNuts",
		c(RankSix, Hearts),
		c(RankSeven, Hearts),
	)
	p1.HasFolded = true

	// P2: weaker hand but active -> should win
	p2 := makePlayer(
		2,
		"Active",
		c(RankKing, Clubs),
		c(RankQueen, Clubs),
	)

	// P3: inactive (e.g. busted out)
	p3 := makePlayer(
		3,
		"Inactive",
		c(RankAce, Clubs),
		c(RankAce, Diamonds),
	)
	p3.IsActive = false

	// Board: simple so winner is obvious.
	table := setupTableState(makeTable(
		c(RankTwo, Hearts),
		c(RankThree, Diamonds),
		c(RankFour, Clubs),
		c(RankFive, Hearts),
		c(RankNine, Spades),
	), p1, p2, p3)

	winners, _, err := EvaluateShowdown(table)
	if err != nil {
		t.Fatalf("EvaluateShowdown returned error: %v", err)
	}

	if len(winners) != 1 || winners[0] != p2 {
		t.Fatalf("expected only active player P2 to win, got winners: %+v", winners)
	}
}

// 5) Error when community cards != 5.
func TestEvaluateShowdown_Edge_ErrorWhenNotFiveCommunityCards(t *testing.T) {
	t.Parallel()

	p1 := makePlayer(
		1,
		"P1",
		c(RankAce, Spades),
		c(RankKing, Spades),
	)

	table := setupTableState(makeTable(
		c(RankTwo, Hearts),
		c(RankThree, Hearts),
		c(RankFour, Hearts),
		c(RankFive, Hearts),
		c(RankSix, Hearts),
	), p1)
	table.CommunityCards = table.CommunityCards[:3]

	_, _, err := EvaluateShowdown(table)
	if err == nil {
		t.Fatalf("expected error when community cards != 5, got nil")
	}
}

// 6) Error when no active, non-folded players.
func TestEvaluateShowdown_Edge_ErrorWhenNoActivePlayers(t *testing.T) {
	t.Parallel()

	p1 := makePlayer(
		1,
		"Folded",
		c(RankAce, Spades),
		c(RankKing, Spades),
	)
	p1.HasFolded = true

	p2 := makePlayer(
		2,
		"Inactive",
		c(RankQueen, Spades),
		c(RankJack, Spades),
	)
	p2.IsActive = false

	table := setupTableState(makeTable(
		c(RankTwo, Hearts),
		c(RankThree, Hearts),
		c(RankFour, Hearts),
		c(RankFive, Hearts),
		c(RankSix, Hearts),
	), p1, p2)

	_, _, err := EvaluateShowdown(table)
	if err == nil {
		t.Fatalf("expected error when no active, non-folded players, got nil")
	}
}

// 7) Full house on board: all players share same full house.
func TestEvaluateShowdown_Edge_FullHouseOnBoardTie(t *testing.T) {
	t.Parallel()

	p1 := makePlayer(
		1,
		"P1",
		c(RankTwo, Clubs),
		c(RankThree, Clubs),
	)
	p2 := makePlayer(
		2,
		"P2",
		c(RankAce, Clubs),
		c(RankQueen, Clubs),
	)

	// Board: 7♥ 7♦ 7♣ K♠ K♦ -> 777KK full house on board.
	table := setupTableState(makeTable(
		c(RankSeven, Hearts),
		c(RankSeven, Diamonds),
		c(RankSeven, Clubs),
		c(RankKing, Spades),
		c(RankKing, Diamonds),
	), p1, p2)

	winners, bestHands, err := EvaluateShowdown(table)
	if err != nil {
		t.Fatalf("EvaluateShowdown returned error: %v", err)
	}

	if len(winners) != 2 {
		t.Fatalf("expected both players to tie, got winners: %+v", winners)
	}

	for _, p := range []*Player{p1, p2} {
		if bestHands[p].Value.Category != FullHouse {
			t.Fatalf("expected FullHouse, got %v for player %d",
				bestHands[p].Value.Category, p.ID)
		}
	}
}

// 8) Full house where hole cards upgrade board full house.
func TestEvaluateShowdown_Edge_FullHouseUpgradedByHoleCards(t *testing.T) {
	t.Parallel()

	// P1: K♥ K♦ -> K K K 7 7 (Kings full of Sevens)
	p1 := makePlayer(
		1,
		"TripsKingsFull",
		c(RankKing, Hearts),
		c(RankKing, Diamonds),
	)

	// P2: 3♣ 4♣ -> 7 7 7 K 2 (Three of a kind, not full house)
	p2 := makePlayer(
		2,
		"BoardTrips",
		c(RankThree, Clubs),
		c(RankFour, Clubs),
	)

	// Board: 7♥ 7♦ 7♣ K♠ 2♦
	table := setupTableState(makeTable(
		c(RankSeven, Hearts),
		c(RankSeven, Diamonds),
		c(RankSeven, Clubs),
		c(RankKing, Spades),
		c(RankTwo, Diamonds),
	), p1, p2)

	bestHands := runTestEvaluation(t, table, p1)

	if bestHands[p1].Value.Category != FullHouse || bestHands[p2].Value.Category != ThreeOfAKind {
		t.Fatalf("expected P1=FullHouse and P2=ThreeOfAKind, got P1=%v, P2=%v",
			bestHands[p1].Value.Category, bestHands[p2].Value.Category)
	}

	// P1 trip rank should be K, P2 trip rank should be 7.
	if bestHands[p1].Value.Ranks[0] != RankKing {
		t.Fatalf("expected P1 trip rank K, got %v", bestHands[p1].Value.Ranks[0])
	}

	if bestHands[p2].Value.Ranks[0] != RankSeven {
		t.Fatalf("expected P2 trip rank 7, got %v", bestHands[p2].Value.Ranks[0])
	}
}

// 9) Four of a kind on board: kicker from hole cards decides.
func TestEvaluateShowdown_Edge_FourOfAKindKickerDecides(t *testing.T) {
	t.Parallel()

	// P1: A♣ 3♣ -> quad 9s with A kicker
	p1 := makePlayer(
		1,
		"Quad9sA",
		c(RankAce, Clubs),
		c(RankThree, Clubs),
	)

	// P2: K♣ 4♣ -> quad 9s with K kicker
	p2 := makePlayer(
		2,
		"Quad9sK",
		c(RankKing, Clubs),
		c(RankFour, Clubs),
	)

	// Board: 9♥ 9♦ 9♣ 9♠ 2♦
	table := setupTableState(makeTable(
		c(RankNine, Hearts),
		c(RankNine, Diamonds),
		c(RankNine, Clubs),
		c(RankNine, Spades),
		c(RankTwo, Diamonds),
	), p1, p2)

	bestHands := runTestEvaluation(t, table, p1)

	if bestHands[p1].Value.Category != FourOfAKind || bestHands[p2].Value.Category != FourOfAKind {
		t.Fatalf("expected both to have FourOfAKind, got P1=%v, P2=%v",
			bestHands[p1].Value.Category, bestHands[p2].Value.Category)
	}

	if bestHands[p1].Value.Ranks[1] != RankAce {
		t.Fatalf("expected P1 kicker A, got %v", bestHands[p1].Value.Ranks[1])
	}

	if bestHands[p2].Value.Ranks[1] != RankKing {
		t.Fatalf("expected P2 kicker K, got %v", bestHands[p2].Value.Ranks[1])
	}
}

// 10) Straight flush vs lower straight flush.
func TestEvaluateShowdown_Edge_StraightFlushBeatsLowerStraightFlush(t *testing.T) {
	t.Parallel()

	// P1: K♥ 3♣ -> 9♥ T♥ J♥ Q♥ K♥ (K-high straight flush)
	p1 := makePlayer(
		1,
		"StraightFlushK",
		c(RankKing, Hearts),
		c(RankThree, Clubs),
	)

	// P2: A♥ 4♣ -> 9♥ T♥ J♥ Q♥ A♥ (not a straight flush; P1 wins)
	p2 := makePlayer(
		2,
		"NonStraightFlush",
		c(RankAce, Hearts),
		c(RankFour, Clubs),
	)

	// Board: 9♥ T♥ J♥ Q♥ 2♣
	table := setupTableState(makeTable(
		c(RankNine, Hearts),
		c(RankTen, Hearts),
		c(RankJack, Hearts),
		c(RankQueen, Hearts),
		c(RankTwo, Clubs),
	), p1, p2)

	bestHands := runTestEvaluation(t, table, p1)

	if bestHands[p1].Value.Category != StraightFlush || bestHands[p2].Value.Category != Flush {
		t.Fatalf("expected P1=StraightFlush and P2=Flush, got P1=%v, P2=%v",
			bestHands[p1].Value.Category, bestHands[p2].Value.Category)
	}

	if bestHands[p1].Value.Ranks[0] != RankKing {
		t.Fatalf("expected P1 straight flush high card K, got %v", bestHands[p1].Value.Ranks[0])
	}

	if bestHands[p2].Value.Ranks[0] != RankAce {
		t.Fatalf("expected P2 flush high card A, got %v", bestHands[p2].Value.Ranks[0])
	}
}

// 11) Ace-low straight flush.
func TestEvaluateShowdown_Edge_AceLowStraightFlush(t *testing.T) {
	t.Parallel()

	// P1: A♥ K♣ -> A♥ 2♥ 3♥ 4♥ 5♥ -> 5-high straight flush (wheel)
	p1 := makePlayer(
		1,
		"WheelStraightFlush",
		c(RankAce, Hearts),
		c(RankKing, Clubs),
	)

	// P2: K♥ Q♣ -> K♥ 5♥ 4♥ 3♥ 2♥ -> flush K-high, no straight
	p2 := makePlayer(
		2,
		"HighFlush",
		c(RankKing, Hearts),
		c(RankQueen, Clubs),
	)

	// Board: 2♥ 3♥ 4♥ 5♥ 9♣
	table := setupTableState(makeTable(
		c(RankTwo, Hearts),
		c(RankThree, Hearts),
		c(RankFour, Hearts),
		c(RankFive, Hearts),
		c(RankNine, Clubs),
	), p1, p2)

	bestHands := runTestEvaluation(t, table, p1)

	if bestHands[p1].Value.Category != StraightFlush {
		t.Fatalf("expected P1 StraightFlush, got %v", bestHands[p1].Value.Category)
	}

	if bestHands[p1].Value.Ranks[0] != RankFive {
		t.Fatalf("expected wheel straight flush high card 5, got %v", bestHands[p1].Value.Ranks[0])
	}

	if bestHands[p2].Value.Category != Flush {
		t.Fatalf("expected P2 Flush, got %v", bestHands[p2].Value.Category)
	}
}

// 12) Trips on board, kicker decides from hole cards.
func TestEvaluateShowdown_Edge_TripsOnBoardKickerDecides(t *testing.T) {
	t.Parallel()

	// P1: A♥ K♦ -> 777AK
	p1 := makePlayer(
		1,
		"AK",
		c(RankAce, Hearts),
		c(RankKing, Diamonds),
	)

	// P2: Q♥ J♦ -> 777QJ
	p2 := makePlayer(
		2,
		"QJ",
		c(RankQueen, Hearts),
		c(RankJack, Diamonds),
	)

	// Board: 7♥ 7♦ 7♣ 2♠ 3♦
	table := setupTableState(makeTable(
		c(RankSeven, Hearts),
		c(RankSeven, Diamonds),
		c(RankSeven, Clubs),
		c(RankTwo, Spades),
		c(RankThree, Diamonds),
	), p1, p2)

	bestHands := runTestEvaluation(t, table, p1)

	if bestHands[p1].Value.Category != ThreeOfAKind || bestHands[p2].Value.Category != ThreeOfAKind {
		t.Fatalf("expected both ThreeOfAKind, got P1=%v, P2=%v",
			bestHands[p1].Value.Category, bestHands[p2].Value.Category)
	}

	if bestHands[p1].Value.Ranks[1] != RankAce || bestHands[p1].Value.Ranks[2] != RankKing {
		t.Fatalf("expected P1 kickers A,K, got %v,%v",
			bestHands[p1].Value.Ranks[1], bestHands[p1].Value.Ranks[2])
	}

	if bestHands[p2].Value.Ranks[1] != RankQueen || bestHands[p2].Value.Ranks[2] != RankJack {
		t.Fatalf("expected P2 kickers Q,J, got %v,%v",
			bestHands[p2].Value.Ranks[1], bestHands[p2].Value.Ranks[2])
	}
}
