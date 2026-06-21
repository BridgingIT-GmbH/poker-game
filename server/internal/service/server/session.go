package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
)

// RoundState describes which phase the server's round lifecycle is in.
type RoundState int

const (
	RoundStateIdle RoundState = iota
	RoundStateWaitingForPlayers
	RoundStateInProgress
)

const tokenByteSize = 16

// generateToken creates a cryptographically secure random hex token.
func generateToken() (string, error) {
	tokenBytes := make([]byte, tokenByteSize)

	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	return hex.EncodeToString(tokenBytes), nil
}

// RoundSession holds server-protocol state for one hand:
// the opaque round ID used for timeout staleness checks and the
// map of pending one-time tokens the server has issued to players.
//
// It is intentionally separate from game.Round, which owns only
// pure poker game state (chips, cards, seat order).
type RoundSession struct {
	roundID       string            // opaque ID; compared in joinTimeoutFired / actionTimeoutFired
	PendingTokens map[string]string // playerName → token the server expects in the response
}

// NewRoundSession creates a RoundSession with a freshly generated round ID.
func NewRoundSession() (*RoundSession, error) {
	id, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("creating round session: %w", err)
	}

	return &RoundSession{
		roundID:       id,
		PendingTokens: make(map[string]string),
	}, nil
}

// RoundID returns the opaque identifier for this session.
// Used only to detect stale timeout events from a previous round.
func (s *RoundSession) RoundID() string {
	return s.roundID
}

// ConsumeToken checks whether token matches the pending token for playerName.
// On success it removes the token so it cannot be reused.
func (s *RoundSession) ConsumeToken(playerName, token string) bool {
	expected, ok := s.PendingTokens[playerName]
	if !ok || expected != token {
		return false
	}

	delete(s.PendingTokens, playerName)

	return true
}

// ConsumeActionToken is like ConsumeToken but additionally verifies that
// playerName is the player currently expected to act (CurrentToActPos).
// Use this for game actions (FOLD/CHECK/CALL/RAISE); use ConsumeToken for
// phase transitions such as JOIN where there is no seat-based turn order.
func (s *RoundSession) ConsumeActionToken(playerName, token string, round *game.Round) bool {
	if round.Table == nil || len(round.PlayerOrder) == 0 {
		return false
	}

	if round.PlayerOrder[round.Table.CurrentToActPos] != playerName {
		return false
	}

	return s.ConsumeToken(playerName, token)
}
