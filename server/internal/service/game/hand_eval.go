package game

import (
	"errors"
	"fmt"
	"sort"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
)

// HandCategory represents the category of a 5-card poker hand.
type HandCategory int

const (
	HighCard HandCategory = iota
	OnePair
	TwoPair
	ThreeOfAKind
	Straight
	Flush
	FullHouse
	FourOfAKind
	StraightFlush
)

// HandValue encodes the strength of a 5-card hand.
//
// Category: overall hand category (straight flush, four of a kind, ...).
// Ranks:   tie-breakers, ordered from most-significant to least-significant.
//
// For example, for:
//   - OnePair:      [pairRank, kicker1, kicker2, kicker3, 0]
//   - TwoPair:      [higherPair, lowerPair, kicker, 0, 0]
//   - Trips:        [tripRank, kicker1, kicker2, 0, 0]
//   - FullHouse:    [tripRank, pairRank, 0, 0, 0]
//   - FourOfAKind:  [quadRank, kicker, 0, 0, 0]
//   - Straight:     [highCard, ..., ...] (with A‑low straight encoded as high=5)
//   - Flush/HighCard: all 5 ranks in descending order.
type HandValue struct {
	Category HandCategory
	Ranks    [5]model.Rank
}

// BestHand represents the best 5-card hand a player can make.
type BestHand struct {
	Value HandValue
	Cards [5]model.Card
}

var errTableNil = errors.New("table is nil")
var errNoActivePlayers = errors.New("no active, non-folded players to evaluate")
var errEvaluationFailed = errors.New("evaluation failed")

const (
	handSize  = 2
	tableSize = 5
)

// EvaluateShowdown evaluates all active, non-folded players on the table,
// determines each player's best 5-card hand, and returns the winner(s).
//
// Requirements / assumptions:
//   - Exactly 5 community cards must be present (River/Showdown).
//   - Each considered player must have 2 hole cards.
//   - Only players with IsActive == true and HasFolded == false are evaluated.
//
// Return values:
//   - winners:      slice of players who tie for the best hand.
//   - bestHands:    map from player to their BestHand (for UI, logging, etc.).
//   - err:          non-nil if evaluation cannot be performed.
func EvaluateShowdown(t *model.Table) (winners []*model.Player, bestHands map[*model.Player]BestHand, err error) {
	if t == nil {
		return nil, nil, errTableNil
	}

	if len(t.CommunityCards) != tableSize {
		return nil, nil, fmt.Errorf("need exactly 5 community cards to evaluate showdown, have %d: %w",
			len(t.CommunityCards), errEvaluationFailed)
	}

	bestHands = make(map[*model.Player]BestHand)

	var bestOverall BestHand

	var haveBest bool

	for _, p := range t.Players {
		if p == nil {
			continue
		}
		// Only consider players still eligible to win.
		if !p.IsActive || p.HasFolded {
			continue
		}

		// Build the 7-card set: 5 community + 2 hole.
		sevenCards := make([]model.Card, 0, handSize+tableSize)
		sevenCards = append(sevenCards, t.CommunityCards...)
		sevenCards = append(sevenCards, p.HoleCards[0], p.HoleCards[1])

		if len(sevenCards) != (handSize + tableSize) {
			return nil, nil, fmt.Errorf("player %d does not have 7 total cards (got %d): %w",
				p.ID, len(sevenCards), errEvaluationFailed)
		}

		best := bestHandFromSeven(sevenCards)
		bestHands[p] = best

		if !haveBest {
			bestOverall = best
			winners = []*model.Player{p}
			haveBest = true

			continue
		}

		cmp := compareHandValues(best.Value, bestOverall.Value)
		if cmp > 0 {
			// New best hand found.
			bestOverall = best
			winners = []*model.Player{p}
		} else if cmp == 0 {
			// Tie: split pot between all such winners.
			winners = append(winners, p)
		}
	}

	if !haveBest {
		return nil, bestHands, errNoActivePlayers
	}

	return winners, bestHands, nil
}

const (
	evaluationCardCount = 5
)

// bestHandFromSeven performs a brute-force search over all C(7,5)=21 possible
// 5-card combinations and returns the best one.
func bestHandFromSeven(cards []model.Card) BestHand {
	if len(cards) < evaluationCardCount {
		panic("bestHandFromSeven requires at least 5 cards")
	}

	var (
		best     BestHand
		haveBest bool
	)

	var comb [5]model.Card

	// Backtracking generator for all 5-card combinations.
	var dfs func(start, depth int)

	dfs = func(start, depth int) {
		if depth == evaluationCardCount {
			value := evaluate5CardHand(comb)
			if !haveBest || compareHandValues(value, best.Value) > 0 {
				best.Value = value
				best.Cards = comb
				haveBest = true
			}

			return
		}

		// We need to choose (5 - depth) remaining cards.
		for i := start; i <= len(cards)-(evaluationCardCount-depth); i++ {
			comb[depth] = cards[i]
			dfs(i+1, depth+1)
		}
	}

	dfs(0, 0)

	return best
}

// compareHandValues compares two HandValue instances.
// Returns:
//
//	-1 if a < b
//	 0 if a == b
//	+1 if a > b
func compareHandValues(a, b HandValue) int {
	if a.Category < b.Category {
		return -1
	}

	if a.Category > b.Category {
		return 1
	}

	for i := range len(a.Ranks) {
		if a.Ranks[i] < b.Ranks[i] {
			return -1
		}

		if a.Ranks[i] > b.Ranks[i] {
			return 1
		}
	}

	return 0
}

const (
	fiveCardValue = 5
	fourOfAKind   = 4
	threeOfAKind  = 3
	pairCardCount = 2
)

func detectFlush(suitCounts map[model.Suit]int) bool {
	isFlush := false

	for _, cnt := range suitCounts {
		if cnt == fiveCardValue {
			isFlush = true

			break
		}
	}

	return isFlush
}

// evaluate5CardHand evaluates a *single* 5-card hand and returns its HandValue.
// It correctly handles:
//   - all standard poker hand categories,
//   - ace-low straights (A‑2‑3‑4‑5).
//
//nolint:funlen // complexity is inherent in hand value detection
func evaluate5CardHand(cards [5]model.Card) HandValue {
	rankCounts := make(map[model.Rank]int, evaluationCardCount)
	suitCounts := make(map[model.Suit]int, evaluationCardCount)

	for _, c := range cards {
		rankCounts[c.Rank]++
		suitCounts[c.Suit]++
	}

	// Detect flush.
	isFlush := detectFlush(suitCounts)

	// Detect straight (including ace-low A‑2‑3‑4‑5).
	isStraight, straightHigh := detectStraight(rankCounts)

	// Count groups (4, 3, 2, 1 of a kind).
	var fourRank, threeRank model.Rank

	pairs := make([]model.Rank, 0, pairCardCount)
	kickers := make([]model.Rank, 0, evaluationCardCount)

	for r, cnt := range rankCounts {
		switch cnt {
		case fourOfAKind:
			fourRank = r
		case threeOfAKind:
			threeRank = r
		case pairCardCount:
			pairs = append(pairs, r)
		case 1:
			kickers = append(kickers, r)
		}
	}

	// Sort multi-card groups and kickers descending by rank.
	sort.Slice(pairs, func(i, j int) bool { return pairs[i] > pairs[j] })
	sort.Slice(kickers, func(i, j int) bool { return kickers[i] > kickers[j] })

	// All ranks in descending order (used for flush/high-card).
	allRanks := make([]int, 0, evaluationCardCount)
	for _, c := range cards {
		allRanks = append(allRanks, int(c.Rank))
	}

	sort.Sort(sort.Reverse(sort.IntSlice(allRanks)))

	var allRanksDesc [5]model.Rank
	for i, v := range allRanks {
		allRanksDesc[i] = model.Rank(v) //nolint:gosec
	}

	var hv HandValue

	switch {
	case isStraight && isFlush:
		hv.Category = StraightFlush
		hv.Ranks = buildStraightRanks(straightHigh)

	case fourRank != 0:
		// Four of a kind: [quadRank, kicker, 0, 0, 0]
		hv.Category = FourOfAKind
		hv.Ranks[0] = fourRank
		hv.Ranks[1] = kickers[0]

	case threeRank != 0 && len(pairs) > 0:
		// Full house: [tripRank, pairRank, 0, 0, 0]
		hv.Category = FullHouse
		hv.Ranks[0] = threeRank
		hv.Ranks[1] = pairs[0]

	case isFlush:
		// Flush: 5 ranks in descending order.
		hv.Category = Flush
		hv.Ranks = allRanksDesc

	case isStraight:
		// Straight: [highCard, ...]
		hv.Category = Straight
		hv.Ranks = buildStraightRanks(straightHigh)

	case threeRank != 0:
		// Three of a kind: [tripRank, kicker1, kicker2, 0, 0]
		hv.Category = ThreeOfAKind
		hv.Ranks[0] = threeRank
		hv.Ranks[1] = kickers[0]
		hv.Ranks[2] = kickers[1]

	case len(pairs) >= pairCardCount:
		// Two pair: [higherPair, lowerPair, kicker, 0, 0]
		hv.Category = TwoPair
		hv.Ranks[0] = pairs[0]
		hv.Ranks[1] = pairs[1]
		hv.Ranks[2] = kickers[0]

	case len(pairs) == 1:
		// One pair: [pairRank, kicker1, kicker2, kicker3, 0]
		hv.Category = OnePair
		hv.Ranks[0] = pairs[0]
		hv.Ranks[1] = kickers[0]
		hv.Ranks[2] = kickers[1]
		hv.Ranks[3] = kickers[2]

	default:
		// High card: 5 ranks in descending order.
		hv.Category = HighCard
		hv.Ranks = allRanksDesc
	}

	return hv
}

// detectStraight determines whether the ranks form a straight and, if so,
// returns (true, highCard). It correctly treats A‑2‑3‑4‑5 as a straight
// with highCard = RankFive.
func detectStraight(rankCounts map[model.Rank]int) (bool, model.Rank) {
	if len(rankCounts) != fiveCardValue {
		// Any straight must have 5 distinct ranks in 5 cards.
		return false, 0
	}

	ranks := make([]int, 0, fiveCardValue)
	for r := range rankCounts {
		ranks = append(ranks, int(r))
	}

	sort.Ints(ranks)

	// Check for normal straight.
	consecutive := true

	for i := 1; i < 5; i++ {
		if ranks[i] != ranks[0]+i {
			consecutive = false

			break
		}
	}

	if consecutive {
		return true, model.Rank(ranks[4]) //nolint:gosec
	}

	// Check for wheel (A‑2‑3‑4‑5): ranks == [2,3,4,5,14]
	if ranks[0] == int(model.RankTwo) &&
		ranks[1] == int(model.RankThree) &&
		ranks[2] == int(model.RankFour) &&
		ranks[3] == int(model.RankFive) &&
		ranks[4] == int(model.RankAce) {
		return true, model.RankFive
	}

	return false, 0
}

// buildStraightRanks builds the Ranks array for a straight-like hand
// (straight or straight flush), given the high card.
//
// For A‑low straights, high == RankFive and we encode as [5,4,3,2,A].
//
//nolint:gosec // gosec is broken on golangci-lint 2.8
func buildStraightRanks(high model.Rank) [5]model.Rank {
	var r [5]model.Rank
	if high == model.RankFive {
		// Wheel: 5‑4‑3‑2‑A
		r[0] = model.RankFive
		r[1] = model.RankFour
		r[2] = model.RankThree
		r[3] = model.RankTwo
		r[4] = model.RankAce

		return r
	}

	// General case: e.g. high = Ten -> 10,9,8,7,6.
	for i := range 5 {
		r[i] = high - model.Rank(i)
	}

	return r
}
