package bot

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

// bufferSize is the size of the UDP receive buffer.
const bufferSize = 4096

// ─── Network plumbing ──────────────────────────────────────────────────────────
//
// The functions in this file take care of the low-level network and protocol
// details (connecting, receiving, sending, parsing). Players don't need to
// touch anything here — see bot.go for the parts relevant to your strategy.

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

