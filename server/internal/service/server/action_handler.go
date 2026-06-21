package server

import (
	"fmt"
	"log/slog"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

// handleAction is the entry point for FOLD/CHECK/CALL/RAISE commands.
// It validates the token via RoundSession, delegates the game logic to the
// game package and broadcasts the result to all players in the round.
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

	if !c.currentSession.ConsumeActionToken(playerName, token, round) {
		c.sendToPeer(string(protocol.CmdRejected+";Invalid token or not your turn\n"), peer)
		slog.Warn("Action rejected: invalid token or wrong turn", "player", playerName, "cmd", cmd)

		return
	}

	// A valid token means it's their turn → stop the action timer.
	if c.actionTimer != nil {
		c.actionTimer.Stop()
		c.actionTimer = nil
	}

	result := game.ApplyPlayerAction(game.Action(string(cmd)), round, playerName)
	broadcastCmd := protocol.CommandPlayer(string(result))

	c.broadcastToPlayers(fmt.Sprintf("%s;%s\n", broadcastCmd, playerName), round.JoinedPlayers)
	slog.Info("Client action", "player", playerName, "action", broadcastCmd, "pot", round.Table.Pot)

	c.advanceTurn(round)
}

// onActionTimeout is called when a player fails to respond within actionTimeout.
// The player is automatically folded via the game package.
func (c *serverCore) onActionTimeout(e actionTimeoutFired) {
	if c.currentRound == nil || c.currentSession == nil {
		return
	}

	if c.currentRound.RoundNumber != e.roundNumber || c.currentSession.RoundID() != e.roundID {
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
	delete(c.currentSession.PendingTokens, e.playerName)

	table := c.currentRound.Table
	game.ApplyFold(table, table.Players[table.CurrentToActPos])

	c.broadcastToPlayers(
		fmt.Sprintf("%s;%s\n", protocol.CmdFold, e.playerName),
		c.currentRound.JoinedPlayers,
	)

	c.advanceTurn(c.currentRound)
}
