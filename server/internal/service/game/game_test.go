package game

import (
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
)

func TestApplyFold(t *testing.T) {
	t.Parallel()

	table := &model.Table{NumPlayersActive: 2}
	player := &model.Player{IsActive: true}

	ApplyFold(table, player)

	if !player.HasFolded {
		t.Error("expected HasFolded = true")
	}

	if player.IsActive {
		t.Error("expected IsActive = false")
	}

	if !player.HasActed {
		t.Error("expected HasActed = true")
	}

	if table.NumPlayersActive != 1 {
		t.Errorf("expected NumPlayersActive=1, got %d", table.NumPlayersActive)
	}
}

func TestApplyCheck_Valid(t *testing.T) {
	t.Parallel()

	table := &model.Table{CurrentBet: 20}
	player := &model.Player{CurrentBet: 20, Chips: 980}

	if err := ApplyCheck(table, player); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !player.HasActed {
		t.Error("expected HasActed = true after check")
	}
}

func TestApplyCheck_OutstandingBet(t *testing.T) {
	t.Parallel()

	table := &model.Table{CurrentBet: 20}
	player := &model.Player{CurrentBet: 10, Chips: 990}

	if err := ApplyCheck(table, player); err == nil {
		t.Error("expected error when there is an outstanding bet")
	}

	if player.HasActed {
		t.Error("player should not be marked as acted after failed check")
	}
}

func TestApplyCall_Valid(t *testing.T) {
	t.Parallel()

	table := &model.Table{CurrentBet: 20, Pot: 30}
	player := &model.Player{CurrentBet: 10, Chips: 990, TotalBetThisHand: 10}

	if err := ApplyCall(table, player); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if player.Chips != 980 {
		t.Errorf("expected chips=980, got %d", player.Chips)
	}

	if player.CurrentBet != 20 {
		t.Errorf("expected CurrentBet=20, got %d", player.CurrentBet)
	}

	if table.Pot != 40 {
		t.Errorf("expected pot=40, got %d", table.Pot)
	}

	if !player.HasActed {
		t.Error("expected HasActed = true")
	}
}

func TestApplyCall_InsufficientChips(t *testing.T) {
	t.Parallel()

	table := &model.Table{CurrentBet: 20}
	player := &model.Player{CurrentBet: 0, Chips: 5}

	if err := ApplyCall(table, player); err == nil {
		t.Error("expected error when player cannot afford the call")
	}

	if player.HasActed {
		t.Error("player should not be marked as acted after failed call")
	}
}

func TestApplyRaise_Valid(t *testing.T) {
	t.Parallel()

	other := &model.Player{IsActive: true, HasActed: true}
	table := &model.Table{CurrentBet: 20, BigBlindAmount: 20, Pot: 30,
		Players: []*model.Player{nil, other}}
	player := &model.Player{CurrentBet: 10, Chips: 990}
	table.Players[0] = player

	if err := ApplyRaise(table, player); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if table.CurrentBet != 40 {
		t.Errorf("expected table.CurrentBet=40, got %d", table.CurrentBet)
	}

	if player.CurrentBet != 40 {
		t.Errorf("expected player.CurrentBet=40, got %d", player.CurrentBet)
	}

	if !player.HasActed {
		t.Error("expected HasActed = true for raiser")
	}

	if other.HasActed {
		t.Error("expected HasActed = false for other player after raise")
	}
}

func TestApplyRaise_InsufficientChips(t *testing.T) {
	t.Parallel()

	table := &model.Table{CurrentBet: 20, BigBlindAmount: 20, Players: []*model.Player{}}
	player := &model.Player{CurrentBet: 10, Chips: 5}

	if err := ApplyRaise(table, player); err == nil {
		t.Error("expected error when player cannot afford the raise")
	}
}

func TestApplyPlayerAction_Fold(t *testing.T) {
	t.Parallel()

	player := &model.Player{IsActive: true}
	table := &model.Table{
		Players:          []*model.Player{player},
		NumPlayersActive: 1,
		CurrentToActPos:  0,
	}
	round := &Round{
		Table:            table,
		RaisesThisStreet: make(map[string]int),
	}

	result := ApplyPlayerAction(ActionFold, round, "Alice")

	if result != ActionFold {
		t.Errorf("expected ActionFold, got %v", result)
	}

	if !player.HasFolded {
		t.Error("expected player to be folded")
	}
}

func TestApplyPlayerAction_InvalidCheckAutoFolds(t *testing.T) {
	t.Parallel()

	player := &model.Player{IsActive: true, CurrentBet: 0}
	table := &model.Table{
		Players:          []*model.Player{player},
		CurrentBet:       20, // outstanding bet → check invalid
		NumPlayersActive: 1,
		CurrentToActPos:  0,
	}
	round := &Round{
		Table:            table,
		RaisesThisStreet: make(map[string]int),
	}

	result := ApplyPlayerAction(ActionCheck, round, "Alice")

	if result != ActionFold {
		t.Errorf("expected auto-fold (ActionFold), got %v", result)
	}
}

func TestApplyPlayerAction_RaiseLimitEnforced(t *testing.T) {
	t.Parallel()

	player := &model.Player{IsActive: true, CurrentBet: 20, Chips: 1000}
	table := &model.Table{
		Players:          []*model.Player{player},
		CurrentBet:       20,
		BigBlindAmount:   20,
		NumPlayersActive: 1,
		CurrentToActPos:  0,
	}
	round := &Round{
		Table:            table,
		RaisesThisStreet: map[string]int{"Alice": maxRaisesPerStreet},
	}

	result := ApplyPlayerAction(ActionRaise, round, "Alice")

	if result != ActionFold {
		t.Errorf("expected auto-fold when raise limit reached, got %v", result)
	}
}

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

func TestDetermineWinner_LastStanding(t *testing.T) {
	t.Parallel()

	alice := &model.Player{IsActive: true, HasFolded: false}
	bob := &model.Player{IsActive: false, HasFolded: true}
	table := &model.Table{Players: []*model.Player{alice, bob}, Pot: 200}

	round := &Round{
		Table:       table,
		PlayerOrder: []string{"Alice", "Bob"},
	}

	result := DetermineWinner(round)

	if !result.IsLastStanding {
		t.Error("expected IsLastStanding = true")
	}

	if result.WinnerName != "Alice" {
		t.Errorf("expected winner Alice, got %s", result.WinnerName)
	}

	if result.Pot != 200 {
		t.Errorf("expected pot 200, got %d", result.Pot)
	}
}

func TestDetermineWinner_Showdown(t *testing.T) {
	t.Parallel()

	alice := &model.Player{IsActive: true, HasFolded: false}
	bob := &model.Player{IsActive: true, HasFolded: false}
	table := &model.Table{Players: []*model.Player{alice, bob}, Pot: 300}

	round := &Round{
		Table:       table,
		PlayerOrder: []string{"Alice", "Bob"},
	}

	result := DetermineWinner(round)

	if result.IsLastStanding {
		t.Error("expected IsLastStanding = false for multi-player showdown")
	}

	if len(result.Survivors) != 2 {
		t.Errorf("expected 2 survivors, got %d", len(result.Survivors))
	}
}
