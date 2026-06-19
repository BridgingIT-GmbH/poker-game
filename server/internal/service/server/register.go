package server

import (
	"log/slog"
	"strconv"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

func (c *serverCore) registerPlayer(name string, peer transport.PeerID) {
	if len(name) < 1 {
		slog.Warn("Player rejected because name is empty", "peer", string(peer))
		c.sendToPeer(string(protocol.CmdRejected+";Name is empty\n"), peer)

		return
	}

	if _, exists := c.players[name]; exists {
		slog.Warn("Player rejected because name is already taken", "name", name, "peer", string(peer))
		c.sendToPeer(string(protocol.CmdRejected+";Name already taken\n"), peer)

		return
	}

	c.players[name] = &Player{Name: name, Peer: peer}
	c.sendToPeer(string(protocol.CmdRegistered+"\n"), peer)

	slog.Info("Player registered", "name", name, "peer", string(peer))
	slog.Info(strconv.Itoa(len(c.players)) + " players currently registered")

	if len(c.players) >= minPlayersToStart && c.roundState == RoundStateIdle {
		c.tryStartRound()
	}
}
