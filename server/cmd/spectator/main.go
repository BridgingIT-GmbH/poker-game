package main

import (
	"log/slog"
	"net"
	"os"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/spectator"
)

func main() {
	network := "udp"
	serverAddr := "localhost:8083"
	httpAddr := ":9000"

	addr, err := net.ResolveUDPAddr(network, serverAddr)
	exitOnError(err)

	conn, err := net.DialUDP(network, nil, addr)
	exitOnError(err)

	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("Closing connection failed", "error", err)
		}
	}()

	slog.Info("Connected to server", "address", serverAddr)

	spec := spectator.NewSpectator(conn, httpAddr)
	if err := spec.Start(); err != nil {
		exitOnError(err)
	}
}

func exitOnError(err error) {
	if err != nil {
		slog.Error("Fatal error", "error", err)
		os.Exit(1)
	}
}
