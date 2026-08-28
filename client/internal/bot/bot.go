package bot

import (
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

const bufferSize = 4096

// Bot handles all network communication and protocol details.
// It maintains the current game state and delegates every action decision
// to the injected Strategy.
type Bot struct {
	name     string
	conn     *net.UDPConn
	strategy Strategy
	state    GameState
}

// New creates a Bot with the given player name, UDP connection, and strategy.
func New(name string, conn *net.UDPConn, strategy Strategy) *Bot {
	return &Bot{name: name, conn: conn, strategy: strategy}
}

// Connect registers the bot with the server.
func (b *Bot) Connect() error {
	msg := fmt.Sprintf("%s;%s", protocol.CmdRegister, b.name)
	if err := b.send(msg); err != nil {
		return fmt.Errorf("register: %w", err)
	}

	slog.Info("Sent REGISTER", "name", b.name)

	return nil
}

// Run starts the blocking receive loop.
func (b *Bot) Run() error {
	buf := make([]byte, bufferSize)

	for {
		n, err := b.conn.Read(buf)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		raw := strings.TrimRight(string(buf[:n]), "\n")
		slog.Info("← server", "msg", raw)

		b.handleMessage(raw)
	}
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
		b.state = GameState{} // reset state for new round
		b.mustSend(fmt.Sprintf("%s;%s", protocol.CmdJoin, payload))
		slog.Info("Sent JOIN")

	case string(protocol.CmdRoundStarted):
		// format: ROUND STARTED;<roundNumber>;<player1,player2,...>
		playerList := payloadOr(parts[1:], "")
		b.state.Players = strings.Split(playerList, ",")
		b.state.FoldedPlayers = make(map[string]bool)
		slog.Info("Round started", "round", payload, "players", b.state.Players)

	case string(protocol.CmdCards):
		b.state.HoleCards = parseCards(payload)
		slog.Info("Hole cards received", "cards", b.state.HoleCards)

	case string(protocol.CmdBoard):
		b.state.Board = parseCards(payload)
		slog.Info("Board updated", "cards", b.state.Board)

	case string(protocol.CmdScore):
		b.state.LastScore = payload
		slog.Info("Score received", "result", payload)

	case string(protocol.CmdYourTurn):
		b.act(payload)

	// ── server-broadcast player actions ──────────────────────────────────────
	case string(protocol.CmdFold):
		if b.state.FoldedPlayers == nil {
			b.state.FoldedPlayers = make(map[string]bool)
		}

		b.state.FoldedPlayers[payload] = true
		slog.Info("Player folded", "player", payload, "activePlayers", b.state.ActivePlayers())

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

// act asks the strategy for a decision and sends the resulting action.
func (b *Bot) act(token string) {
	if token == "" {
		slog.Warn("YOUR TURN received without token, skipping action")
		return
	}

	action := b.strategy.Decide(b.state)
	slog.Info("Strategy decided", "action", action, "holeCards", b.state.HoleCards, "board", b.state.Board)

	b.mustSend(fmt.Sprintf("%s;%s", action, token))
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (b *Bot) send(msg string) error {
	_, err := b.conn.Write([]byte(msg + "\n"))
	return err
}

func (b *Bot) mustSend(msg string) {
	if err := b.send(msg); err != nil {
		slog.Error("Failed to send message", "msg", msg, "err", err)
	}
}

func payloadOr(parts []string, fallback string) string {
	if len(parts) < 2 {
		return fallback
	}

	return parts[1]
}

// parseCards splits a comma-separated card string into individual cards.
// The server sends cards in the format "As,Kh" (CARDS) and "Td,7c,2h" (BOARD).
func parseCards(raw string) []string {
	if raw == "" {
		return nil
	}

	return strings.Split(raw, ",")
}
