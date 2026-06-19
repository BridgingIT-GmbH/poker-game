package server

import (
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
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

func TestConsumeToken(t *testing.T) {
	t.Parallel()

	round, _ := NewRound(1)
	round.PendingTokens["Alice"] = "abc123"

	// Wrong token → rejected, token stays.
	if round.consumeToken("Alice", "wrong") {
		t.Error("consumeToken should return false for wrong token")
	}

	if _, still := round.PendingTokens["Alice"]; !still {
		t.Error("token should not be deleted after failed validation")
	}

	// Correct token → accepted, token removed.
	if !round.consumeToken("Alice", "abc123") {
		t.Error("consumeToken should return true for correct token")
	}

	if _, exists := round.PendingTokens["Alice"]; exists {
		t.Error("token should be deleted after successful consumption")
	}

	// Replay → rejected.
	if round.consumeToken("Alice", "abc123") {
		t.Error("consumeToken should return false on replay (token already consumed)")
	}

	// Unknown player → rejected.
	if round.consumeToken("Bob", "anything") {
		t.Error("consumeToken should return false for unknown player")
	}
}

func TestConsumeActionToken(t *testing.T) {
	t.Parallel()

	round, _ := NewRound(1)
	round.PlayerOrder = []string{"Alice", "Bob"}
	round.PendingTokens["Alice"] = "alice-token"
	round.PendingTokens["Bob"] = "bob-token"

	// Nil table → rejected.
	if round.consumeActionToken("Alice", "alice-token") {
		t.Error("should return false when Table is nil")
	}

	// Set up table with Alice at CurrentToActPos=0.
	round.Table = &model.Table{CurrentToActPos: 0}

	// Bob has a valid token but it's Alice's turn → rejected.
	if round.consumeActionToken("Bob", "bob-token") {
		t.Error("should return false when it is not Bob's turn")
	}

	// Bob's token must not have been consumed.
	if _, ok := round.PendingTokens["Bob"]; !ok {
		t.Error("Bob's token should still be present after failed consumeActionToken")
	}

	// Alice with wrong token → rejected.
	if round.consumeActionToken("Alice", "wrong") {
		t.Error("should return false for wrong token")
	}

	// Alice with correct token and it's her turn → accepted.
	if !round.consumeActionToken("Alice", "alice-token") {
		t.Error("should return true when token and turn are both valid")
	}

	// Replay → rejected (token consumed).
	if round.consumeActionToken("Alice", "alice-token") {
		t.Error("should return false on replay")
	}
}

func TestConsumeActionToken_EmptyPlayerOrder(t *testing.T) {
	t.Parallel()

	round, _ := NewRound(1)
	round.Table = &model.Table{CurrentToActPos: 0}
	round.PendingTokens["Alice"] = "tok"

	if round.consumeActionToken("Alice", "tok") {
		t.Error("should return false when PlayerOrder is empty")
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
