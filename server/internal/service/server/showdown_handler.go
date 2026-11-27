package server

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

// handleShowdown calls game.DetermineWinner, broadcasts player scores and
// survivors' hole cards via a SCORE message, and ends the round.
func (c *serverCore) handleShowdown(round *game.Round) {
	result, err := game.DetermineWinner(round)
	if err != nil {
		// TODO error handling
		return
	}

	payouts := calculatePayouts(result)
	msg := c.buildScoreMessage(round, payouts, true)
	c.broadcastToPlayers(msg, round.JoinedPlayers)
	slog.Info("Showdown", "round", round.RoundNumber, "winners", winnerNames(result.Winners), "pot", result.Pot)

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

	payouts := calculatePayouts(result)

	msg := c.buildScoreMessage(round, payouts, false)
	c.broadcastToPlayers(msg, round.JoinedPlayers)
	slog.Info("Last player standing wins", "winners", result.Winners, "pot", result.Pot)

	c.sendToSpectators(msg)

	c.endRound()
}

func calculatePayouts(result *game.ShowdownResult) map[*model.Player]float64 {
	payouts := make(map[*model.Player]float64, len(result.Winners))
	if len(result.Winners) > 0 {
		share := float64(result.Pot) / float64(len(result.Winners))
		for _, winner := range result.Winners {
			payouts[winner] = share
		}
	}

	return payouts
}

func (c *serverCore) buildScoreMessage(round *game.Round, payouts map[*model.Player]float64,
	revealActiveCards bool) string {
	parts := make([]string, 0, len(round.JoinedPlayers))
	for _, name := range round.PlayerOrder {
		player, ok := round.JoinedPlayers[name]
		if !ok {
			continue
		}

		if player == nil {
			slog.Error("Cannot include player in score message", "player", name)

			continue
		}

		score := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", float64(player.Chips)+payouts[player]), "0"), ".")

		part := fmt.Sprintf("%s:%s", name, score)
		if revealActiveCards && player.IsActive && !player.HasFolded {
			part += fmt.Sprintf(":%s,%s", player.HoleCards[0], player.HoleCards[1])
		}

		parts = append(parts, part)
	}

	return fmt.Sprintf("%s;%s\n", protocol.CmdScore, strings.Join(parts, ";"))
}

func winnerNames(winners []*model.Player) []string {
	names := make([]string, 0, len(winners))
	for _, winner := range winners {
		if winner != nil {
			names = append(names, winner.Name)
		}
	}

	return names
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
