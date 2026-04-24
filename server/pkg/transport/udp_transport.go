package transport

import (
	"fmt"
	"log/slog"
	"net"
)

// UDPTransport encapsulates all UDP-specific I/O.
//
// It uses PeerID as the textual UDP address (addr.String(), e.g. "127.0.0.1:1234" or "[::1]:1234").
type UDPTransport struct {
	conn *net.UDPConn
}

func NewUDPTransport(conn *net.UDPConn) *UDPTransport {
	return &UDPTransport{conn: conn}
}

// ReadLoop blocks reading UDP packages and calls onPackage for each received package.
// It returns if reading fails.
func (t *UDPTransport) ReadLoop(onPackage func(InboundPackage)) error {
	bufferSize := 1024
	buffer := make([]byte, bufferSize)

	for {
		n, addr, err := t.conn.ReadFromUDP(buffer)
		if err != nil {
			return fmt.Errorf("failed to read from UDP: %w", err)
		}

		peer := t.peerID(addr)

		data := append([]byte(nil), buffer[:n]...) // copy; buffer is reused
		onPackage(InboundPackage{Peer: peer, Data: data})
	}
}

// WriteLoop blocks reading outbound packages and sending them via UDP.
// It terminates when outbound is closed.
func (t *UDPTransport) WriteLoop(outbound <-chan OutboundPackage) {
	for pkg := range outbound {
		addr, err := net.ResolveUDPAddr("udp", string(pkg.Peer))
		if err != nil {
			slog.Warn("Dropping outbound package: invalid peer address", "peer", string(pkg.Peer), "error", err)

			continue
		}

		_, err = t.conn.WriteToUDP(pkg.Data, addr)
		if err != nil {
			slog.Error("Failed to send message", "error", err)
		}
	}
}

func (t *UDPTransport) peerID(addr *net.UDPAddr) PeerID {
	return PeerID(addr.String())
}
