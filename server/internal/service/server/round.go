package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
)

const (
	tokenByteSize = 16
)

type RoundState int

const (
	RoundStateIdle RoundState = iota
	RoundStateWaitingForPlayers
	RoundStateInProgress
)

type Round struct {
	roundID          string // internal identifier; used only for join-timeout staleness checks
	JoinedPlayers    map[string]*Player
	RoundNumber      int
	State            RoundState
	PlayerOrder      []string          // randomly ordered; fixed for the hand
	Table            *model.Table      // populated after cards are dealt
	PendingTokens    map[string]string // playerName → token the server expects in the response
	RaisesThisStreet map[string]int    // playerName → raises made in current betting round (max 1)
}

func NewRound(roundNumber int) (*Round, error) {
	id, err := generateToken()
	if err != nil {
		return nil, err
	}

	return &Round{
		roundID:          id,
		JoinedPlayers:    make(map[string]*Player),
		RoundNumber:      roundNumber,
		State:            RoundStateWaitingForPlayers,
		PendingTokens:    make(map[string]string),
		RaisesThisStreet: make(map[string]int),
	}, nil
}

func generateToken() (string, error) {
	tokenBytes := make([]byte, tokenByteSize)

	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	return hex.EncodeToString(tokenBytes), nil
}

// consumeToken checks whether token matches the pending token for playerName.
// On success it removes the token from PendingTokens so it cannot be reused.
func (r *Round) consumeToken(playerName, token string) bool {
	expected, ok := r.PendingTokens[playerName]
	if !ok || expected != token {
		return false
	}

	delete(r.PendingTokens, playerName)

	return true
}

// consumeActionToken is like consumeToken but additionally verifies that
// playerName is the player currently expected to act (CurrentToActPos).
// Use this for game actions (FOLD/CHECK/CALL/RAISE); use consumeToken for
// phase transitions such as JOIN where there is no seat-based turn order.
func (r *Round) consumeActionToken(playerName, token string) bool {
	if r.Table == nil || len(r.PlayerOrder) == 0 {
		return false
	}

	if r.PlayerOrder[r.Table.CurrentToActPos] != playerName {
		return false
	}

	return r.consumeToken(playerName, token)
}
