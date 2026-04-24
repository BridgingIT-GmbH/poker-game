package main

import (
	"log/slog"
	"net"
	"os"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/server"
)

func main() {
	network := "udp"
	port := "8083"

	addr, err := net.ResolveUDPAddr(network, "localhost:"+port)
	exitOnError(err)

	conn, err := net.ListenUDP(network, addr)
	exitOnError(err)

	defer func(conn *net.UDPConn) {
		err := conn.Close()
		if err != nil {
			exitOnError(err)
		}
	}(conn)

	slog.Info("Starting server on port " + port)

	srv := server.NewServer(conn)
	if err := srv.Start(); err != nil {
		exitOnError(err)
	}
}

func exitOnError(err error) {
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
