package server

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

// handleShowdown calls game.DetermineWinner, broadcasts all survivors' hole
// cards via a SCORE message and ends the round.
func (c *serverCore) handleShowdown(round *game.Round) {
	result, err := game.DetermineWinner(round)
	if err != nil {
		// TODO error handling
		return
	}

	parts := make([]string, 0, len(result.Survivors))
	for _, s := range result.Survivors {
		playerRoundScore := int64(0)
		if slices.Contains(result.Winners, s) {
			playerRoundScore = result.Pot
		}

		parts = append(parts, fmt.Sprintf("%s:%d:%s,%s", s.Name, playerRoundScore, s.HoleCards[0], s.HoleCards[1]))
	}

	msg := fmt.Sprintf("%s;%s\n", protocol.CmdScore, strings.Join(parts, ";"))
	c.broadcastToPlayers(msg, round.JoinedPlayers)
	slog.Info("Showdown", "round", round.RoundNumber, "winner", result.Winners[0].Name, "pot", result.Pot)

	c.sendToSpectators(msg)

	c.endRound()
}

// handleLastPlayerStanding is called when all but one player has folded.
// It calls game.DetermineWinner and broadcasts the winner and pot via SCORE.
func (c *serverCore) handleLastPlayerStanding(round *game.Round) {
	result, err := game.DetermineWinner(round)
	if err != nil {
		// TODO error handling
		return
	}

	msg := fmt.Sprintf("%s;%s:%d\n", protocol.CmdScore, result.Winners[0].Name, result.Pot)
	c.broadcastToPlayers(msg, round.JoinedPlayers)
	slog.Info("Last player standing wins", "winners", result.Winners, "pot", result.Pot)

	c.sendToSpectators(msg)

	c.endRound()
}

// endRound stops any running action timer, resets the server state to idle
// and attempts to start the next round.
func (c *serverCore) endRound() {
	if c.actionTimer != nil {
		c.actionTimer.Stop()
		c.actionTimer = nil
	}

	c.roundState = RoundStateIdle
	c.currentRound = nil
	c.currentSession = nil
	c.tryStartRound()
}
