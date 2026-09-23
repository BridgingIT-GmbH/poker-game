package main

import (
	"log/slog"
	"net"
	"os"

	"github.com/BridgingIT-GmbH/poker-game/client/internal/bot"
)

func main() {
	addr, name := parseArgs()

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	exitOnError("resolve address", err)

	conn, err := net.DialUDP("udp", nil, udpAddr)
	exitOnError("dial UDP", err)

	defer conn.Close()

	slog.Info("Connecting to server", "addr", addr, "player", name)

	b := bot.New(name, conn)

	exitOnError("register", b.Connect())
	exitOnError("run", b.Run())
}

// parseArgs returns the server address and player name from CLI arguments.
// Usage: client <host:port> [name]
func parseArgs() (addr, name string) {
	if len(os.Args) < 2 {
		slog.Error("Usage: client <host:port> [name]")
		os.Exit(1)
	}

	addr = os.Args[1]
	name = "CallBot"

	if len(os.Args) >= 3 {
		name = os.Args[2]
	}

	return addr, name
}

func exitOnError(context string, err error) {
	if err != nil {
		slog.Error(context, "err", err)
		os.Exit(1)
	}
}
