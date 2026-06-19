package server

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

// eventsChanBufferSize is the mailbox capacity of the serverCore actor.
// Producers are the UDP ReadLoop (inbound packets) and time.AfterFunc callbacks
// (join timeout). A larger buffer prevents producers from blocking under burst
// load while the actor is busy processing a previous event.
// 0 or 1 would risk deadlock; 128 gives ample headroom for realistic traffic.
const (
	minPlayersToStart    = 2
	joinTimeout          = 5 * time.Second
	eventsChanBufferSize = 128
)

type coreEvent any

type inboundEvent struct {
	pkg transport.InboundPackage
}

type joinTimeoutFired struct {
	roundNumber int
	token       string
}

// serverCore contains the poker protocol/game logic.
// It is transport-agnostic and only communicates via packages and PeerID.
//
// State is owned exclusively by the actor goroutine in run().
type serverCore struct {
	players      map[string]*Player
	roundNumber  int
	currentRound *Round
	roundState   RoundState

	events   chan coreEvent
	outbound chan<- transport.OutboundPackage

	joinTimer *time.Timer
}

func newServerCore(outbound chan<- transport.OutboundPackage) *serverCore {
	return &serverCore{
		players:     make(map[string]*Player),
		roundState:  RoundStateIdle,
		roundNumber: 0,
		events:      make(chan coreEvent, eventsChanBufferSize),
		outbound:    outbound,
	}
}

func (c *serverCore) run() {
	for ev := range c.events {
		switch e := ev.(type) {
		case inboundEvent:
			c.handlePackage(e.pkg)
		case joinTimeoutFired:
			c.onJoinTimeout(e)
		default:
			slog.Warn("Unknown core event", "type", fmt.Sprintf("%T", ev))
		}
	}
}

func (c *serverCore) postPackage(pkg transport.InboundPackage) {
	c.events <- inboundEvent{pkg: pkg}
}

func (c *serverCore) handlePackage(pkg transport.InboundPackage) {
	c.handleMessage(pkg.Data, pkg.Peer)
}

func (c *serverCore) sendToPeer(message string, peer transport.PeerID) {
	c.outbound <- transport.OutboundPackage{Peer: peer, Data: []byte(message)}
}

func (c *serverCore) handleMessage(data []byte, peer transport.PeerID) {
	slog.Debug("Received message: " + string(data) + " from " + string(peer))

	message, err := protocol.ParseMessage(data)
	if err != nil {
		c.sendToPeer(string(protocol.CmdRejected+";invalid message format\n"), peer)

		return
	}

	command := protocol.CommandPlayer(message[0])
	switch command {
	case protocol.CmdRegister:
		c.registerPlayer(message[1], peer)
	case protocol.CmdJoin:
		c.handleJoin(message[1], peer)
	default:
		c.sendToPeer(string(protocol.CmdRejected+";Unknown command\n"), peer)
	}
}

func (c *serverCore) broadcastToPlayers(message string, players map[string]*Player) {
	for _, player := range players {
		c.sendToPeer(message, player.Peer)
	}
}
