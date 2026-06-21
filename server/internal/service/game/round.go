package game

import "github.com/BridgingIT-GmbH/poker-game/server/internal/model"

// Round holds the pure game state of a single poker hand.
// It is intentionally free of any transport or protocol concerns:
// no PeerIDs, no pending tokens, no round-ID for staleness checks.
// Those server-specific fields live in server/session.go.
type Round struct {
	RoundNumber      int
	PlayerOrder      []string                 // seat order, randomly assigned before dealing
	JoinedPlayers    map[string]*model.Player // playerName → game-layer player (chips, cards, flags)
	Table            *model.Table             // populated after cards are dealt
	RaisesThisStreet map[string]int           // playerName → raises made in current betting round (max 1)
}

// NewRound creates an empty Round for the given hand number.
func NewRound(roundNumber int) *Round {
	return &Round{
		RoundNumber:      roundNumber,
		JoinedPlayers:    make(map[string]*model.Player),
		RaisesThisStreet: make(map[string]int),
	}
}
