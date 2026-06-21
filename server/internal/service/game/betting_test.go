package game

import (
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
)

func TestIsBettingRoundOver(t *testing.T) {
	t.Parallel()

	alice := &model.Player{IsActive: true, HasActed: true, CurrentBet: 20}
	bob := &model.Player{IsActive: true, HasActed: true, CurrentBet: 20}
	table := &model.Table{Players: []*model.Player{alice, bob}, CurrentBet: 20}

	if !IsBettingRoundOver(table) {
		t.Error("expected round to be over when all players acted and bets match")
	}

	bob.HasActed = false

	if IsBettingRoundOver(table) {
		t.Error("expected round NOT to be over when Bob hasn't acted")
	}
}

func TestIsBettingRoundOver_FoldedPlayerIgnored(t *testing.T) {
	t.Parallel()

	alice := &model.Player{IsActive: true, HasActed: true, CurrentBet: 20}
	bob := &model.Player{IsActive: false, HasFolded: true, HasActed: false}
	table := &model.Table{Players: []*model.Player{alice, bob}, CurrentBet: 20}

	if !IsBettingRoundOver(table) {
		t.Error("expected round to be over when only non-folded player has acted")
	}
}

func TestNextStreet_PreFlopToFlop(t *testing.T) {
	t.Parallel()

	alice := &model.Player{IsActive: true, HasActed: true}
	bob := &model.Player{IsActive: true, HasActed: true}
	table := model.NewTable([]*model.Player{alice, bob}, 10, 20)
	table.BettingRound = model.PreFlop
	table.DealerPosition = 0
	table.Deck.Shuffle()

	round := &Round{
		Table:            table,
		RaisesThisStreet: map[string]int{"Alice": 1, "Bob": 1},
	}

	showdown := NextStreet(round)

	if showdown {
		t.Error("expected no showdown after pre-flop → flop")
	}

	if table.BettingRound != model.Flop {
		t.Errorf("expected Flop, got %v", table.BettingRound)
	}

	if len(table.CommunityCards) != 3 {
		t.Errorf("expected 3 community cards, got %d", len(table.CommunityCards))
	}

	if round.RaisesThisStreet["Alice"] != 0 || round.RaisesThisStreet["Bob"] != 0 {
		t.Error("raise counters should be reset after advancing street")
	}
}

func TestNextStreet_RiverToShowdown(t *testing.T) {
	t.Parallel()

	alice := &model.Player{IsActive: true}
	table := model.NewTable([]*model.Player{alice}, 10, 20)
	table.BettingRound = model.River

	round := &Round{Table: table, RaisesThisStreet: make(map[string]int)}

	showdown := NextStreet(round)

	if !showdown {
		t.Error("expected showdown=true after River")
	}

	if table.BettingRound != model.Showdown {
		t.Errorf("expected Showdown, got %v", table.BettingRound)
	}
}
