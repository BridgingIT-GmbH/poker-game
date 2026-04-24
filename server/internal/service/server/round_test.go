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

	if round.Token == "" {
		t.Error("NewRound() token is empty")
	}

	if len(round.Token) != 32 {
		t.Errorf("Expected token length 32, got %d", len(round.Token))
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
