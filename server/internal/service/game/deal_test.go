package game

import (
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
)

func TestBuildOrderedPlayers_Length(t *testing.T) {
	t.Parallel()

	names := []string{"Alice", "Bob", "Charlie"}
	orderedNames, players := BuildOrderedPlayers(names, 1000)

	if len(orderedNames) != 3 {
		t.Errorf("expected 3 ordered names, got %d", len(orderedNames))
	}

	if len(players) != 3 {
		t.Errorf("expected 3 players, got %d", len(players))
	}
}

func TestBuildOrderedPlayers_ChipsAndActive(t *testing.T) {
	t.Parallel()

	names := []string{"Alice", "Bob"}
	_, players := BuildOrderedPlayers(names, 500)

	for _, p := range players {
		if p.Chips != 500 {
			t.Errorf("expected Chips=500, got %d", p.Chips)
		}

		if !p.IsActive {
			t.Errorf("expected player %s to be active", p.Name)
		}
	}
}

func TestBuildOrderedPlayers_ContainsAllNames(t *testing.T) {
	t.Parallel()

	input := []string{"Alice", "Bob", "Charlie"}
	orderedNames, players := BuildOrderedPlayers(input, 1000)

	// Build a set of the input names for lookup.
	nameSet := make(map[string]struct{}, len(input))
	for _, n := range input {
		nameSet[n] = struct{}{}
	}

	for _, n := range orderedNames {
		if _, ok := nameSet[n]; !ok {
			t.Errorf("unexpected name in ordered list: %s", n)
		}
	}

	for _, p := range players {
		if _, ok := nameSet[p.Name]; !ok {
			t.Errorf("unexpected player name: %s", p.Name)
		}
	}
}

func TestBuildOrderedPlayers_DoesNotMutateInput(t *testing.T) {
	t.Parallel()

	original := []string{"Alice", "Bob", "Charlie"}
	input := []string{"Alice", "Bob", "Charlie"}

	BuildOrderedPlayers(input, 1000)

	for i, n := range input {
		if n != original[i] {
			t.Errorf("input slice was mutated at index %d: got %s, want %s", i, n, original[i])
		}
	}
}

func TestBuildTable_Positions_TwoPlayers(t *testing.T) {
	t.Parallel()

	players := []*model.Player{
		model.NewPlayer(0, "Alice", 1000),
		model.NewPlayer(1, "Bob", 1000),
	}

	// Round 1: dealer=0, sb=1, bb=0 (heads-up wraps), first-to-act=1
	table := BuildTable(players, 1, 10, 20)

	if table.DealerPosition != 0 {
		t.Errorf("expected DealerPosition=0, got %d", table.DealerPosition)
	}

	if table.SmallBlindPos != 1 {
		t.Errorf("expected SmallBlindPos=1, got %d", table.SmallBlindPos)
	}

	// bigBlindOffset=2, so (0+2)%2 = 0
	if table.BigBlindPos != 0 {
		t.Errorf("expected BigBlindPos=0, got %d", table.BigBlindPos)
	}

	// (BigBlindPos+1)%2 = 1
	if table.CurrentToActPos != 1 {
		t.Errorf("expected CurrentToActPos=1, got %d", table.CurrentToActPos)
	}
}

func TestBuildTable_Positions_ThreePlayers(t *testing.T) {
	t.Parallel()

	players := []*model.Player{
		model.NewPlayer(0, "Alice", 1000),
		model.NewPlayer(1, "Bob", 1000),
		model.NewPlayer(2, "Charlie", 1000),
	}

	// Round 1: dealer=(1-1)%3=0, sb=1, bb=2, first-to-act=0
	table := BuildTable(players, 1, 10, 20)

	if table.DealerPosition != 0 {
		t.Errorf("expected DealerPosition=0, got %d", table.DealerPosition)
	}

	if table.SmallBlindPos != 1 {
		t.Errorf("expected SmallBlindPos=1, got %d", table.SmallBlindPos)
	}

	if table.BigBlindPos != 2 {
		t.Errorf("expected BigBlindPos=2, got %d", table.BigBlindPos)
	}

	if table.CurrentToActPos != 0 {
		t.Errorf("expected CurrentToActPos=0, got %d", table.CurrentToActPos)
	}
}

func TestBuildTable_NumPlayersActive(t *testing.T) {
	t.Parallel()

	players := []*model.Player{
		model.NewPlayer(0, "Alice", 1000),
		model.NewPlayer(1, "Bob", 1000),
		model.NewPlayer(2, "Charlie", 1000),
	}

	table := BuildTable(players, 1, 10, 20)

	if table.NumPlayersActive != 3 {
		t.Errorf("expected NumPlayersActive=3, got %d", table.NumPlayersActive)
	}
}

func TestBuildTable_DealerRotatesEachRound(t *testing.T) {
	t.Parallel()

	players := []*model.Player{
		model.NewPlayer(0, "Alice", 1000),
		model.NewPlayer(1, "Bob", 1000),
		model.NewPlayer(2, "Charlie", 1000),
	}

	for round := 1; round <= 6; round++ {
		table := BuildTable(players, round, 10, 20)
		expected := (round - 1) % 3

		if table.DealerPosition != expected {
			t.Errorf("round %d: expected DealerPosition=%d, got %d", round, expected, table.DealerPosition)
		}
	}
}

func TestCollectBlinds_DeductsChips(t *testing.T) {
	t.Parallel()

	alice := model.NewPlayer(0, "Alice", 1000)
	bob := model.NewPlayer(1, "Bob", 1000)
	players := []*model.Player{alice, bob}

	table := model.NewTable(players, 10, 20)
	table.SmallBlindPos = 0
	table.BigBlindPos = 1

	CollectBlinds(table)

	if alice.Chips != 990 {
		t.Errorf("expected Alice chips=990 after paying small blind, got %d", alice.Chips)
	}

	if bob.Chips != 980 {
		t.Errorf("expected Bob chips=980 after paying big blind, got %d", bob.Chips)
	}
}

func TestCollectBlinds_SetsCurrentBet(t *testing.T) {
	t.Parallel()

	alice := model.NewPlayer(0, "Alice", 1000)
	bob := model.NewPlayer(1, "Bob", 1000)
	players := []*model.Player{alice, bob}

	table := model.NewTable(players, 10, 20)
	table.SmallBlindPos = 0
	table.BigBlindPos = 1

	CollectBlinds(table)

	if table.CurrentBet != 20 {
		t.Errorf("expected CurrentBet=20 (big blind), got %d", table.CurrentBet)
	}
}

func TestCollectBlinds_SetsPot(t *testing.T) {
	t.Parallel()

	alice := model.NewPlayer(0, "Alice", 1000)
	bob := model.NewPlayer(1, "Bob", 1000)
	players := []*model.Player{alice, bob}

	table := model.NewTable(players, 10, 20)
	table.SmallBlindPos = 0
	table.BigBlindPos = 1

	CollectBlinds(table)

	if table.Pot != 30 {
		t.Errorf("expected Pot=30 (sb+bb), got %d", table.Pot)
	}
}

func TestCollectBlinds_PlayerCurrentBet(t *testing.T) {
	t.Parallel()

	alice := model.NewPlayer(0, "Alice", 1000)
	bob := model.NewPlayer(1, "Bob", 1000)
	players := []*model.Player{alice, bob}

	table := model.NewTable(players, 10, 20)
	table.SmallBlindPos = 0
	table.BigBlindPos = 1

	CollectBlinds(table)

	if alice.CurrentBet != 10 {
		t.Errorf("expected Alice CurrentBet=10, got %d", alice.CurrentBet)
	}

	if bob.CurrentBet != 20 {
		t.Errorf("expected Bob CurrentBet=20, got %d", bob.CurrentBet)
	}
}

func TestCollectBlinds_ShortStack(t *testing.T) {
	t.Parallel()

	// Alice only has 5 chips but small blind is 10 – she pays what she can.
	alice := model.NewPlayer(0, "Alice", 5)
	bob := model.NewPlayer(1, "Bob", 1000)
	players := []*model.Player{alice, bob}

	table := model.NewTable(players, 10, 20)
	table.SmallBlindPos = 0
	table.BigBlindPos = 1

	CollectBlinds(table)

	if alice.Chips != 0 {
		t.Errorf("expected Alice chips=0 after paying all she can, got %d", alice.Chips)
	}

	if alice.CurrentBet != 5 {
		t.Errorf("expected Alice CurrentBet=5, got %d", alice.CurrentBet)
	}

	// Pot = 5 (Alice partial SB) + 20 (Bob BB)
	if table.Pot != 25 {
		t.Errorf("expected Pot=25, got %d", table.Pot)
	}
}
