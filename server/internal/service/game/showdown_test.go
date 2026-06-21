package game

import (
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
)

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
