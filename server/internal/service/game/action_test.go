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

	result, reason := ApplyPlayerAction(ActionFold, round, "Alice")

	if result != ActionFold {
		t.Errorf("expected ActionFold, got %v", result)
	}

	if reason != FoldReasonNone {
		t.Errorf("expected no fold reason for voluntary fold, got %q", reason)
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

	result, reason := ApplyPlayerAction(ActionCheck, round, "Alice")

	if result != ActionFold {
		t.Errorf("expected auto-fold (ActionFold), got %v", result)
	}

	if reason != FoldReasonCannotCheck {
		t.Errorf("expected reason %q, got %q", FoldReasonCannotCheck, reason)
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

	result, reason := ApplyPlayerAction(ActionRaise, round, "Alice")

	if result != ActionFold {
		t.Errorf("expected auto-fold when raise limit reached, got %v", result)
	}

	if reason != FoldReasonRaiseLimit {
		t.Errorf("expected reason %q, got %q", FoldReasonRaiseLimit, reason)
	}
}

func TestApplyPlayerAction_FoldReasons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action Action
		player *model.Player
		table  *model.Table
		want   FoldReason
	}{
		{
			name:   "call without enough chips",
			action: ActionCall,
			player: &model.Player{IsActive: true, Chips: 5},
			table:  &model.Table{CurrentBet: 20, BigBlindAmount: 20},
			want:   FoldReasonCannotAffordCall,
		},
		{
			name:   "raise without enough chips",
			action: ActionRaise,
			player: &model.Player{IsActive: true, CurrentBet: 10, Chips: 5},
			table:  &model.Table{CurrentBet: 20, BigBlindAmount: 20},
			want:   FoldReasonInsufficientChips,
		},
		{
			name:   "unknown action",
			action: Action("DANCE"),
			player: &model.Player{IsActive: true},
			table:  &model.Table{},
			want:   FoldReasonInvalidAction,
		},
		{
			name:   "valid check",
			action: ActionCheck,
			player: &model.Player{IsActive: true, CurrentBet: 20},
			table:  &model.Table{CurrentBet: 20},
			want:   FoldReasonNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.table.Players = []*model.Player{tt.player}
			tt.table.NumPlayersActive = 1
			round := &Round{Table: tt.table, RaisesThisStreet: make(map[string]int)}

			_, reason := ApplyPlayerAction(tt.action, round, "Alice")
			if reason != tt.want {
				t.Errorf("expected reason %q, got %q", tt.want, reason)
			}
		})
	}
}
