package server

import (
	"fmt"
	"net"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

// outboundChanBufferSize buffers broadcast bursts from the actor goroutine to
// the UDP WriteLoop. A broadcast sends one packet per player, so the buffer
// must absorb at least one full broadcast without blocking the actor.
// 32 would suffice for typical table sizes; 128 gives comfortable headroom.
const outboundChanBufferSize = 128

// Server wires transport (UDP) and protocol/game logic (serverCore).
//
// Transport responsibilities:
//   - Read/Write UDP package
//   - Translate between net.UDPAddr and PeerID
//
// Core responsibilities:
//   - Parse protocol messages
//   - Maintain poker game state (players/rounds)
//   - Emit responses as OutboundPackage
//
// The public API is: NewServer(*net.UDPConn) and Start().
type Server struct {
	transport *transport.UDPTransport
	core      *serverCore

	outbound chan transport.OutboundPackage
}

func NewServer(conn *net.UDPConn) *Server {
	outbound := make(chan transport.OutboundPackage, outboundChanBufferSize)

	udpTransport := transport.NewUDPTransport(conn)
	core := newServerCore(outbound)

	return &Server{
		transport: udpTransport,
		core:      core,
		outbound:  outbound,
	}
}

func (srv *Server) Start() error {
	go srv.core.run()
	go srv.transport.WriteLoop(srv.outbound)

	// Block on UDP reads; each package is forwarded into the core's actor loop.
	err := srv.transport.ReadLoop(func(pkg transport.InboundPackage) {
		srv.core.postPackage(pkg)
	})
	if err != nil {
		return fmt.Errorf("udp read loop: %w", err)
	}

	return nil
}
