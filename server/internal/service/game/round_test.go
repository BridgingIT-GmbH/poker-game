package game

import (
	"testing"
)

func TestNewGameRound(t *testing.T) {
	t.Parallel()

	round := NewRound(1)

	if round == nil {
		t.Fatal("NewRound() returned nil")
	}

	if round.RoundNumber != 1 {
		t.Errorf("expected RoundNumber 1, got %d", round.RoundNumber)
	}

	if round.JoinedPlayers == nil {
		t.Error("JoinedPlayers should be initialised")
	}

	if round.RaisesThisStreet == nil {
		t.Error("RaisesThisStreet should be initialised")
	}
}
