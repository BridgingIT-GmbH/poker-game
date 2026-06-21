package server

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
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
	actionTimeout        = 30 * time.Second // TODO this is a high value for development. Reduce it in production.
	eventsChanBufferSize = 128
)

type coreEvent any

type inboundEvent struct {
	pkg transport.InboundPackage
}

type joinTimeoutFired struct {
	roundNumber int
	roundID     string
}

type actionTimeoutFired struct {
	roundNumber int
	roundID     string
	playerName  string
}

// serverCore contains the poker protocol/game logic.
// It is transport-agnostic and only communicates via packages and PeerID.
//
// State is owned exclusively by the actor goroutine in run().
type serverCore struct {
	players        map[string]*model.Client
	roundNumber    int
	currentRound   *game.Round   // pure game state for the ongoing hand
	currentSession *RoundSession // server-protocol state (tokens, round-ID) for the ongoing hand
	roundState     RoundState

	events   chan coreEvent
	outbound chan<- transport.OutboundPackage

	joinTimer   *time.Timer
	actionTimer *time.Timer
}

func newServerCore(outbound chan<- transport.OutboundPackage) *serverCore {
	return &serverCore{
		players:     make(map[string]*model.Client),
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
		case actionTimeoutFired:
			c.onActionTimeout(e)
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
	case protocol.CmdFold, protocol.CmdCheck, protocol.CmdCall, protocol.CmdRaise:
		c.handleAction(command, message[1], peer)
	default:
		c.sendToPeer(string(protocol.CmdRejected+";Unknown command\n"), peer)
	}
}

func (c *serverCore) broadcastToPlayers(message string, joinedPlayers map[string]*model.Player) {
	for name := range joinedPlayers {
		if p, ok := c.players[name]; ok {
			c.sendToPeer(message, p.Peer)
		}
	}
}

// playerNameByPeer returns the registered name for a peer, or "" if unknown.
func (c *serverCore) playerNameByPeer(peer transport.PeerID) string {
	for name, p := range c.players {
		if p.Peer == peer {
			return name
		}
	}

	return ""
}
