package server

import (
	"log/slog"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

func (c *serverCore) handleJoin(token string, peer transport.PeerID) {
	expectedToken := ""
	if c.currentRound != nil {
		expectedToken = c.currentRound.Token
	}

	if c.currentRound == nil || c.currentRound.Token != token {
		c.sendToPeer(string(protocol.CmdRejected+";Invalid token\n"), peer)
		slog.Warn(
			"Player tried to join with invalid token",
			"peer",
			string(peer),
			"token",
			token,
			"expectedToken",
			expectedToken,
		)

		return
	}

	if c.roundState != RoundStateWaitingForPlayers {
		c.sendToPeer(string(protocol.CmdRejected+";Round not accepting players\n"), peer)
		slog.Warn("Player tried to join but round not accepting", "peer", string(peer))

		return
	}

	var playerName string

	for name, player := range c.players {
		if player.Peer == peer {
			playerName = name

			break
		}
	}

	if playerName == "" {
		c.sendToPeer(string(protocol.CmdRejected+";Player not registered\n"), peer)
		slog.Warn("Unregistered player tried to join", "peer", string(peer))

		return
	}

	if _, exists := c.currentRound.JoinedPlayers[playerName]; exists {
		slog.Info("Player already joined", "name", playerName)

		return
	}

	c.currentRound.JoinedPlayers[playerName] = c.players[playerName]
	slog.Info("Player joined round", "name", playerName, "round", c.currentRound.RoundNumber)
}
