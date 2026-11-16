package bot

import (
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

// Bot handles all network communication and protocol details.
// Players are free to implement their own decision logic directly inside
// the message switch below (see handleMessage / CmdYourTurn case).
type Bot struct {
	name string
	conn *net.UDPConn
}

// New creates a Bot with the given player name and UDP connection.
func New(name string, conn *net.UDPConn) *Bot {
	return &Bot{name: name, conn: conn}
}

// ─── Message dispatcher ───────────────────────────────────────────────────────

// handleMessage routes an incoming server message to the appropriate handler.
// It handles both server→client commands and server-broadcast player actions
// (FOLD/CALL/RAISE/CHECK with the acting player's name as payload).
func (b *Bot) handleMessage(raw string) {
	// Split into at most 3 parts to handle ROUND STARTED;roundNum;player1,player2
	parts := strings.SplitN(raw, ";", 3)
	cmd := parts[0]
	payload := payloadOr(parts, "")

	switch cmd {
	// ── server → client ──────────────────────────────────────────────────────
	case string(protocol.CmdRegistered):
		slog.Info("Registered successfully")

	case string(protocol.CmdRejected):
		slog.Warn("Rejected by server", "reason", payload)

	case string(protocol.CmdRoundStarting):
		b.mustSend(fmt.Sprintf("%s;%s", protocol.CmdJoin, payload))
		slog.Info("Sent JOIN")

	case string(protocol.CmdRoundStarted):
		// format: ROUND STARTED;<roundNumber>;<player1,player2,...>
		playerList := payloadOr(parts[1:], "")
		players := strings.Split(playerList, ",")
		slog.Info("Round started", "round", payload, "players", players)

	case string(protocol.CmdCards):
		slog.Info("Hole cards received", "cards", parseCards(payload))

	case string(protocol.CmdBoard):
		slog.Info("Board updated", "cards", parseCards(payload))

	case string(protocol.CmdScore):
		slog.Info("Score received", "result", payload)

	case string(protocol.CmdYourTurn):
		b.act(payload)

	// ── server-broadcast player actions ──────────────────────────────────────
	case string(protocol.CmdFold):
		slog.Info("Player folded", "player", payload)

	case string(protocol.CmdCall):
		slog.Info("Player called", "player", payload)

	case string(protocol.CmdRaise):
		slog.Info("Player raised", "player", payload)

	case string(protocol.CmdCheck):
		slog.Info("Player checked", "player", payload)

	default:
		slog.Debug("Unknown message", "raw", raw)
	}
}

// act decides which action the bot takes on its turn and sends it to the
// server.
//
// ╔════════════════════════════════════════════════════════════════════╗
// ║  HIER GEHT'S LOS – baue hier deine eigene Poker-Strategie ein!     ║
// ╚════════════════════════════════════════════════════════════════════╝
//
// Verfügbare Aktionen (protocol-Paket):
//
//	protocol.CmdFold   – Hand aufgeben
//	protocol.CmdCheck  – passen (nur gültig, wenn kein Einsatz aussteht)
//	protocol.CmdCall   – aktuellen Einsatz mitgehen (Standard)
//	protocol.CmdRaise  – Einsatz erhöhen
//
// Tipp: In handleMessage() oben siehst du alle Nachrichten, die der Server
// schickt (CARDS, BOARD, FOLD, ...). Wenn du dir Informationen zwischen
// Zügen merken möchtest, ergänze einfach eigene Felder im Bot-Struct oben
// und aktualisiere sie in den passenden Cases.
func (b *Bot) act(token string) {
	if token == "" {
		slog.Warn("YOUR TURN received without token, skipping action")
		return
	}

	action := protocol.CmdCall // Standard: immer CALL

	// TODO: Ersetze die Zeile oben durch deine eigene Entscheidungslogik,

	slog.Info("Decided action", "action", action)

	b.mustSend(fmt.Sprintf("%s;%s", action, token))
}
