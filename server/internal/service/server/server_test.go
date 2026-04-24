package server

import (
	"net"
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

func TestNewServer(t *testing.T) {
	t.Parallel()

	conn := &net.UDPConn{}
	srv := NewServer(conn)

	if srv == nil {
		t.Fatal("NewServer() returned nil")
	}

	if srv.transport == nil {
		t.Fatal("NewServer() transport not initialized")
	}

	_ = (*transport.UDPTransport)(nil)

	if srv.core == nil {
		t.Fatal("NewServer() core not initialized")
	}

	if srv.outbound == nil {
		t.Fatal("NewServer() outbound channel not initialized")
	}
}
