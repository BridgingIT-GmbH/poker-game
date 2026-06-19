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
	Token         string
	JoinedPlayers map[string]*Player
	RoundNumber   int
	State         RoundState
	PlayerOrder   []string     // alphabetically sorted; fixed for the hand
	Table         *model.Table // populated after cards are dealt
}

func NewRound(roundNumber int) (*Round, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	return &Round{
		Token:         token,
		JoinedPlayers: make(map[string]*Player),
		RoundNumber:   roundNumber,
		State:         RoundStateWaitingForPlayers,
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
