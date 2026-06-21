package server

import (
	"log/slog"
	"strconv"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

func (c *serverCore) registerPlayer(name string, peer transport.PeerID) {
	if len(name) < 1 {
		slog.Warn("Client rejected because name is empty", "peer", string(peer))
		c.sendToPeer(string(protocol.CmdRejected+";Name is empty\n"), peer)

		return
	}

	if existing, exists := c.players[name]; exists {
		// Allow reconnect: update the peer so future messages reach the new address.
		// Any round currently in progress will auto-fold the player via the action
		// timer; they will be included normally in the next round.
		slog.Info("Client re-registered (peer updated)",
			"name", name, "old_peer", string(existing.Peer), "new_peer", string(peer))
		existing.Peer = peer
		c.sendToPeer(string(protocol.CmdRegistered+"\n"), peer)
		c.resendRoundStartingIfNeeded(name, peer)

		return
	}

	c.players[name] = &model.Client{Name: name, Peer: peer}
	c.sendToPeer(string(protocol.CmdRegistered+"\n"), peer)

	slog.Info("Client registered", "name", name, "peer", string(peer))
	slog.Info(strconv.Itoa(len(c.players)) + " players currently registered")

	if len(c.players) >= minPlayersToStart && c.roundState == RoundStateIdle {
		c.tryStartRound()
	}
}
