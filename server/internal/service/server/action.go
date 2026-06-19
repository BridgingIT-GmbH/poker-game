package server

import (
	"fmt"
	"log/slog"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

// handleAction is the entry point for FOLD/CHECK/CALL/RAISE commands.
// It validates the token, applies the action to the game model, broadcasts
// the result to all players and advances the turn.
func (c *serverCore) handleAction(cmd protocol.CommandPlayer, token string, peer transport.PeerID) {
	round := c.currentRound
	if round == nil || c.roundState != RoundStateInProgress {
		c.sendToPeer(string(protocol.CmdRejected+";No round in progress\n"), peer)

		return
	}

	playerName := c.playerNameByPeer(peer)
	if playerName == "" {
		c.sendToPeer(string(protocol.CmdRejected+";Not registered\n"), peer)

		return
	}

	// consumeActionToken validates the token AND that it is this player's turn.
	if !round.consumeActionToken(playerName, token) {
		c.sendToPeer(string(protocol.CmdRejected+";Invalid token or not your turn\n"), peer)
		slog.Warn("Action rejected: invalid token or wrong turn", "player", playerName, "cmd", cmd)

		return
	}

	// A valid token means it's their turn → stop the action timer.
	if c.actionTimer != nil {
		c.actionTimer.Stop()
		c.actionTimer = nil
	}

	broadcastCmd := applyPlayerAction(cmd, round, playerName)
	c.broadcastToPlayers(fmt.Sprintf("%s;%s\n", broadcastCmd, playerName), round.JoinedPlayers)
	slog.Info("Player action", "player", playerName, "action", broadcastCmd, "pot", round.Table.Pot)

	c.advanceTurn(round)
}

// onActionTimeout is called when a player fails to respond within actionTimeout.
// The player is automatically folded.
func (c *serverCore) onActionTimeout(e actionTimeoutFired) {
	if c.currentRound == nil {
		return
	}

	if c.currentRound.RoundNumber != e.roundNumber || c.currentRound.roundID != e.roundID {
		return // stale event from a previous round
	}

	if c.roundState != RoundStateInProgress {
		return
	}

	// Confirm it's still this player's turn.
	currentName := c.currentRound.PlayerOrder[c.currentRound.Table.CurrentToActPos]
	if currentName != e.playerName {
		return
	}

	slog.Warn("Action timeout, auto-folding player", "player", e.playerName)

	// Remove the pending token so the player can't act after the fold.
	delete(c.currentRound.PendingTokens, e.playerName)

	table := c.currentRound.Table
	applyFold(table, table.Players[table.CurrentToActPos])

	c.broadcastToPlayers(
		fmt.Sprintf("%s;%s\n", protocol.CmdFold, e.playerName),
		c.currentRound.JoinedPlayers,
	)

	c.advanceTurn(c.currentRound)
}
