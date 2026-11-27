package server

import (
	"log/slog"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

// maxNameLength is the maximum number of characters (runes) in a player name.
const maxNameLength = 12

// forbiddenNameChars break the protocol format:
//
//	;  message separator
//	,  player list in ROUND STARTED
//	:  SCORE format (name:score:cards)
const forbiddenNameChars = ";,:"

// validateName returns a rejection reason, or "" if the name is valid.
func validateName(name string) string {
	switch {
	case name == "":
		return "Name is empty"
	case utf8.RuneCountInString(name) > maxNameLength:
		return "Name is too long (max " + strconv.Itoa(maxNameLength) + " characters)"
	case strings.ContainsAny(name, forbiddenNameChars):
		return "Name contains forbidden characters (; , :)"
	case strings.IndexFunc(name, unicode.IsControl) >= 0:
		return "Name contains control characters"
	}

	return ""
}

func (c *serverCore) registerPlayer(name string, peer transport.PeerID) {
	if reason := validateName(name); reason != "" {
		slog.Warn("Client rejected", "name", name, "reason", reason, "peer", string(peer))
		c.sendToPeer(string(protocol.CmdRejected)+";"+reason+"\n", peer)

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

	c.sendToSpectators(string(protocol.CmdRegistered) + ";" + name + "\n")

	if len(c.players) >= minPlayersToStart && c.roundState == RoundStateIdle {
		c.tryStartRound()
	}
}

func (c *serverCore) registerSpectator(peer transport.PeerID) {
	name := "spectator_" + strconv.Itoa(len(c.spectators)+1)
	c.spectators[name] = &model.Client{Name: name, Peer: peer}
	c.sendToPeer(string(protocol.CmdRegistered+"\n"), peer)

	slog.Info("Spectator registered", "name", name, "peer", string(peer))
	slog.Info(strconv.Itoa(len(c.spectators)) + " spectators currently registered")
}
