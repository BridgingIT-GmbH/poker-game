package server

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
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

	round, err := NewRound(c.roundNumber)
	if err != nil {
		slog.Error("Failed to create round", "error", err)

		return
	}

	c.currentRound = round
	c.roundState = RoundStateWaitingForPlayers

	for name, player := range c.players {
		token, err := generateToken()
		if err != nil {
			slog.Error("Failed to generate join token", "player", name, "error", err)

			return
		}

		round.PendingTokens[name] = token
		message := fmt.Sprintf("%s;%s\n", protocol.CmdRoundStarting, token)
		c.sendToPeer(message, player.Peer)
	}

	slog.Info("Round starting", "round", round.RoundNumber)

	c.setJoinTimeout(round)
}

func (c *serverCore) setJoinTimeout(round *Round) {
	if c.joinTimer != nil {
		c.joinTimer.Stop()
		c.joinTimer = nil
	}

	c.joinTimer = time.AfterFunc(joinTimeout, func() {
		// Callback runs in its own goroutine; post back into the actor.
		c.events <- joinTimeoutFired{roundNumber: round.RoundNumber, roundID: round.roundID}
	})
}

func (c *serverCore) onJoinTimeout(e joinTimeoutFired) {
	if c.currentRound == nil {
		return
	}

	if c.roundState != RoundStateWaitingForPlayers {
		return
	}

	if c.currentRound.RoundNumber != e.roundNumber || c.currentRound.roundID != e.roundID {
		return
	}

	if len(c.currentRound.JoinedPlayers) < minPlayersToStart {
		slog.Warn("Not enough players joined, aborting round", "joined", len(c.currentRound.JoinedPlayers))
		c.roundState = RoundStateIdle
		c.currentRound = nil
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
