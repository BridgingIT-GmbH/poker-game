package game

import "github.com/BridgingIT-GmbH/poker-game/server/internal/model"

const (
	flopCardCount      = 3
	turnRiverCardCount = 1
)

// IsBettingRoundOver returns true when every active, non-folded, non-all-in
// player has acted and matched the current bet.
func IsBettingRoundOver(table *model.Table) bool {
	for _, p := range table.Players {
		if !p.IsActive || p.HasFolded || p.IsAllIn {
			continue
		}

		if !p.HasActed || p.CurrentBet < table.CurrentBet {
			return false
		}
	}

	return true
}

// NextStreet advances the round to the next betting street:
//   - Resets per-round player state (HasActed, CurrentBet) on the table.
//   - Resets the per-street raise counters on the round.
//   - Deals the appropriate community cards (Flop=3, Turn/River=1).
//   - Updates table.BettingRound.
//   - Sets table.CurrentToActPos to the first active player left of the dealer
//     (post-flop convention).
//
// Returns true when the River has been completed and the game moves to Showdown.
// The caller is responsible for broadcasting the new board and prompting the
// next player to act.
func NextStreet(round *Round) (reachedShowdown bool) {
	table := round.Table
	table.ResetForNewBettingRound()

	// Reset per-street raise counters for the new betting round.
	round.RaisesThisStreet = make(map[string]int)

	switch table.BettingRound {
	case model.PreFlop:
		table.CommunityCards = append(table.CommunityCards, table.Deck.Deal(flopCardCount)...)
		table.BettingRound = model.Flop
	case model.Flop:
		table.CommunityCards = append(table.CommunityCards, table.Deck.Deal(turnRiverCardCount)...)
		table.BettingRound = model.Turn
	case model.Turn:
		table.CommunityCards = append(table.CommunityCards, table.Deck.Deal(turnRiverCardCount)...)
		table.BettingRound = model.River
	case model.River:
		table.BettingRound = model.Showdown

		return true
	case model.Showdown:
		// Already at showdown – nothing to advance.
		return false
	}

	// Post-flop: first to act starts left of the dealer.
	table.CurrentToActPos = NextActivePlayerPos(table, (table.DealerPosition+1)%len(table.Players))

	return false
}

// NextActivePlayerPos returns the index of the next active, non-folded,
// non-all-in player starting the search at startPos (inclusive, wrapping).
func NextActivePlayerPos(table *model.Table, startPos int) int {
	n := len(table.Players)
	pos := startPos % n

	for range n {
		p := table.Players[pos]
		if p.IsActive && !p.HasFolded && !p.IsAllIn {
			return pos
		}

		pos = (pos + 1) % n
	}

	return pos // fallback: return last checked position
}
