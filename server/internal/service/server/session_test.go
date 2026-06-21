package server

import (
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
)

func TestNewRoundSession(t *testing.T) {
	t.Parallel()

	session, err := NewRoundSession()
	if err != nil {
		t.Fatalf("NewRoundSession() returned error: %v", err)
	}

	if session == nil {
		t.Fatal("NewRoundSession() returned nil")
	}

	if session.RoundID() == "" {
		t.Error("RoundID should not be empty")
	}

	if len(session.RoundID()) != 32 {
		t.Errorf("expected RoundID length 32, got %d", len(session.RoundID()))
	}

	if session.PendingTokens == nil {
		t.Error("PendingTokens should be initialised")
	}

	if len(session.PendingTokens) != 0 {
		t.Errorf("expected empty PendingTokens, got %d", len(session.PendingTokens))
	}
}

func TestConsumeToken(t *testing.T) {
	t.Parallel()

	session, _ := NewRoundSession()
	session.PendingTokens["Alice"] = "abc123"

	// Wrong token → rejected, token stays.
	if session.ConsumeToken("Alice", "wrong") {
		t.Error("ConsumeToken should return false for wrong token")
	}

	if _, still := session.PendingTokens["Alice"]; !still {
		t.Error("token should not be deleted after failed validation")
	}

	// Correct token → accepted, token removed.
	if !session.ConsumeToken("Alice", "abc123") {
		t.Error("ConsumeToken should return true for correct token")
	}

	if _, exists := session.PendingTokens["Alice"]; exists {
		t.Error("token should be deleted after successful consumption")
	}

	// Replay → rejected.
	if session.ConsumeToken("Alice", "abc123") {
		t.Error("ConsumeToken should return false on replay (token already consumed)")
	}

	// Unknown player → rejected.
	if session.ConsumeToken("Bob", "anything") {
		t.Error("ConsumeToken should return false for unknown player")
	}
}

func TestConsumeActionToken(t *testing.T) {
	t.Parallel()

	session, _ := NewRoundSession()
	session.PendingTokens["Alice"] = "alice-token"
	session.PendingTokens["Bob"] = "bob-token"

	round := game.NewRound(1)
	round.PlayerOrder = []string{"Alice", "Bob"}

	// Nil table → rejected.
	if session.ConsumeActionToken("Alice", "alice-token", round) {
		t.Error("should return false when Table is nil")
	}

	// Set up table with Alice at CurrentToActPos=0.
	round.Table = &model.Table{CurrentToActPos: 0}

	// Bob has a valid token but it's Alice's turn → rejected.
	if session.ConsumeActionToken("Bob", "bob-token", round) {
		t.Error("should return false when it is not Bob's turn")
	}

	// Bob's token must not have been consumed.
	if _, ok := session.PendingTokens["Bob"]; !ok {
		t.Error("Bob's token should still be present after failed ConsumeActionToken")
	}

	// Alice with wrong token → rejected.
	if session.ConsumeActionToken("Alice", "wrong", round) {
		t.Error("should return false for wrong token")
	}

	// Alice with correct token and it's her turn → accepted.
	if !session.ConsumeActionToken("Alice", "alice-token", round) {
		t.Error("should return true when token and turn are both valid")
	}

	// Replay → rejected (token consumed).
	if session.ConsumeActionToken("Alice", "alice-token", round) {
		t.Error("should return false on replay")
	}
}

func TestConsumeActionToken_EmptyPlayerOrder(t *testing.T) {
	t.Parallel()

	session, _ := NewRoundSession()
	session.PendingTokens["Alice"] = "tok"

	round := game.NewRound(1)
	round.Table = &model.Table{CurrentToActPos: 0}
	// PlayerOrder is empty

	if session.ConsumeActionToken("Alice", "tok", round) {
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
		t.Errorf("expected token length 32, got %d", len(token1))
	}
}
