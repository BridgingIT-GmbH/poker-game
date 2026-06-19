package server

import (
	"testing"
)

func TestNewRound(t *testing.T) {
	t.Parallel()

	roundNumber := 1

	round, err := NewRound(roundNumber)
	if err != nil {
		t.Fatalf("NewRound() returned error: %v", err)
	}

	if round == nil {
		t.Fatal("NewRound() returned nil")
	}

	if round.RoundNumber != roundNumber {
		t.Errorf("Expected round number %d, got %d", roundNumber, round.RoundNumber)
	}

	if round.roundID == "" {
		t.Error("NewRound() roundID is empty")
	}

	if len(round.roundID) != 32 {
		t.Errorf("Expected roundID length 32, got %d", len(round.roundID))
	}

	if round.PendingTokens == nil {
		t.Error("NewRound() PendingTokens not initialized")
	}

	if len(round.PendingTokens) != 0 {
		t.Errorf("Expected empty PendingTokens, got %d", len(round.PendingTokens))
	}

	if round.JoinedPlayers == nil {
		t.Error("NewRound() JoinedPlayers not initialized")
	}

	if len(round.JoinedPlayers) != 0 {
		t.Errorf("Expected empty JoinedPlayers, got %d", len(round.JoinedPlayers))
	}

	if round.State != RoundStateWaitingForPlayers {
		t.Errorf("Expected state WaitingForPlayers, got %v", round.State)
	}
}

func TestGenerateToken(t *testing.T) {
	t.Parallel()

	token1, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken() returned error: %v", err)
	}

	token2, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken() returned error: %v", err)
	}

	if token1 == token2 {
		t.Error("generateToken() returned same token twice")
	}

	if len(token1) != 32 {
		t.Errorf("Expected token length 32, got %d", len(token1))
	}
}
