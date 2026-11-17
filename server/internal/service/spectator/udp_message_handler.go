package spectator

import (
	"log/slog"
	"net"
	"strings"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

// UdpMessageHandler reads UDP packets from a connection and buffers each one in
// a queue for later, delayed playback.
type UdpMessageHandler struct {
	conn *net.UDPConn
	// queue is bidirectional because a full queue drops the oldest round.
	queue     chan []string
	roundData []string
}

const (
	udpBufferSize   = 1024
	commandSplitMax = 2
)

func NewUdpMessageHandler(conn *net.UDPConn, queue chan []string) *UdpMessageHandler {
	return &UdpMessageHandler{
		conn:      conn,
		queue:     queue,
		roundData: nil,
	}
}

// ListenForMessages reads UDP packets and buffers each one in the queue for
// later, delayed playback.
func (h *UdpMessageHandler) ListenForMessages() {
	buffer := make([]byte, udpBufferSize)
	for {
		n, _, err := h.conn.ReadFromUDP(buffer)
		if err != nil {
			slog.Error("Receiving message failed", "error", err)

			continue
		}

		message := string(buffer[:n])
		slog.Debug("Received message", "message", message)

		parts := strings.SplitN(message, ";", commandSplitMax)
		cmd := parts[0]

		switch cmd {
		case string(protocol.CmdRoundStarted):
			if h.roundData != nil {
				h.enqueueRound(h.roundData)
			}

			h.roundData = []string{message}
		default:
			if h.roundData != nil {
				h.roundData = append(h.roundData, message)
			}
		}
	}
}

// enqueueRound puts a completed round into the queue. If the queue is full the
// oldest round is dropped so reading new packets never blocks.
//
// This relies on the queue having capacity 1: it is either empty (the send is
// ready) or full (the receive is ready), so select never has to choose between
// both and never blocks. With a larger capacity select could pick the receive
// at random and drop a round for no reason.
func (h *UdpMessageHandler) enqueueRound(round []string) {
	for {
		select {
		case h.queue <- round:
			slog.Debug("Round enqueued", "first", firstMessage(round))

			return
		case dropped := <-h.queue:
			slog.Debug("Queue full, round dropped", "first", firstMessage(dropped))
		}
	}
}

// firstMessage returns the first message of a round for logging purposes.
func firstMessage(round []string) string {
	if len(round) == 0 {
		return "<empty round>"
	}

	return round[0]
}
