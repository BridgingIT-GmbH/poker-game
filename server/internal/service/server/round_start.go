package server

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

func (c *serverCore) tryStartRound() {
	if len(c.players) < minPlayersToStart {
		slog.Info("Not enough players to start round")

		return
	}

	if c.roundState != RoundStateIdle {
		return
	}

	c.roundNumber++

	round := game.NewRound(c.roundNumber)

	session, err := NewRoundSession()
	if err != nil {
		slog.Error("Failed to create round session", "error", err)

		return
	}

	c.currentRound = round
	c.currentSession = session
	c.roundState = RoundStateWaitingForPlayers

	for name, player := range c.players {
		token, err := generateToken()
		if err != nil {
			slog.Error("Failed to generate join token", "player", name, "error", err)

			return
		}

		session.PendingTokens[name] = token
		message := fmt.Sprintf("%s;%s\n", protocol.CmdRoundStarting, token)
		c.sendToPeer(message, player.Peer)
	}

	slog.Info("Round starting", "round", round.RoundNumber)

	c.setJoinTimeout(round, session)
}

func (c *serverCore) setJoinTimeout(round *game.Round, session *RoundSession) {
	if c.joinTimer != nil {
		c.joinTimer.Stop()
		c.joinTimer = nil
	}

	c.joinTimer = time.AfterFunc(joinTimeout, func() {
		// Callback runs in its own goroutine; post back into the actor.
		c.events <- joinTimeoutFired{roundNumber: round.RoundNumber, roundID: session.RoundID()}
	})
}

func (c *serverCore) onJoinTimeout(e joinTimeoutFired) {
	if c.currentRound == nil || c.currentSession == nil {
		return
	}

	if c.roundState != RoundStateWaitingForPlayers {
		return
	}

	if c.currentRound.RoundNumber != e.roundNumber || c.currentSession.RoundID() != e.roundID {
		return
	}

	if len(c.currentRound.JoinedPlayers) < minPlayersToStart {
		slog.Warn("Not enough players joined, aborting round", "joined", len(c.currentRound.JoinedPlayers))
		c.roundState = RoundStateIdle
		c.currentRound = nil
		c.currentSession = nil
		c.joinTimer = nil

		c.tryStartRound()

		return
	}

	c.startRound()
}

func (c *serverCore) startRound() {
	if c.currentRound == nil {
		return
	}

	playerNames := make([]string, 0, len(c.currentRound.JoinedPlayers))
	for name := range c.currentRound.JoinedPlayers {
		playerNames = append(playerNames, name)
	}

	message := fmt.Sprintf(
		"%s;%d;%s\n",
		protocol.CmdRoundStarted,
		c.currentRound.RoundNumber,
		strings.Join(playerNames, ","),
	)

	c.broadcastToPlayers(message, c.currentRound.JoinedPlayers)
	slog.Info("Round started", "round", c.currentRound.RoundNumber, "players", playerNames)

	c.roundState = RoundStateInProgress
	c.joinTimer = nil

	c.dealCards()
}

// resendRoundStartingIfNeeded re-sends the ROUND STARTING token to a
// reconnecting player whose peer address has just been updated.
// It is a no-op when no round is waiting for players or the player has no
// pending join token (e.g. they already joined or were never invited).
func (c *serverCore) resendRoundStartingIfNeeded(name string, peer transport.PeerID) {
	if c.currentSession == nil || c.roundState != RoundStateWaitingForPlayers {
		return
	}

	token, ok := c.currentSession.PendingTokens[name]
	if !ok {
		return
	}

	msg := fmt.Sprintf("%s;%s\n", protocol.CmdRoundStarting, token)
	c.sendToPeer(msg, peer)
	slog.Info("Re-sent ROUND STARTING to reconnected player", "name", name)
}
