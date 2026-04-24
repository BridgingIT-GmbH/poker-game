package server

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

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

func (c *serverCore) registerPlayer(name string, peer transport.PeerID) {
	if len(name) < 1 {
		slog.Warn("Player rejected because name is empty", "peer", string(peer))
		c.sendToPeer(string(protocol.CmdRejected+";Name is empty\n"), peer)

		return
	}

	if _, exists := c.players[name]; exists {
		slog.Warn("Player rejected because name is already taken", "name", name, "peer", string(peer))
		c.sendToPeer(string(protocol.CmdRejected+";Name already taken\n"), peer)

		return
	}

	c.players[name] = &Player{Name: name, Peer: peer}
	c.sendToPeer(string(protocol.CmdRegistered+"\n"), peer)

	slog.Info("Player registered", "name", name, "peer", string(peer))
	slog.Info(strconv.Itoa(len(c.players)) + " players currently registered")

	if len(c.players) >= minPlayersToStart && c.roundState == RoundStateIdle {
		c.tryStartRound()
	}
}

func (c *serverCore) tryStartRound() {
	if len(c.players) < minPlayersToStart {
		slog.Info("Not enough players to start round")

		return
	}

	if c.roundState != RoundStateIdle {
		return
	}

	c.roundNumber++

	round, err := NewRound(c.roundNumber)
	if err != nil {
		slog.Error("Failed to create round", "error", err)

		return
	}

	c.currentRound = round
	c.roundState = RoundStateWaitingForPlayers

	message := fmt.Sprintf("%s;%s\n", protocol.CmdRoundStarting, round.Token)
	c.broadcastToPlayers(message, c.players)
	slog.Info("Round starting", "round", round.RoundNumber, "token", round.Token)

	c.setJoinTimeout(round)
}

func (c *serverCore) setJoinTimeout(round *Round) {
	if c.joinTimer != nil {
		c.joinTimer.Stop()
		c.joinTimer = nil
	}

	c.joinTimer = time.AfterFunc(joinTimeout, func() {
		// Callback runs in its own goroutine; post back into the actor.
		c.events <- joinTimeoutFired{roundNumber: round.RoundNumber, token: round.Token}
	})
}

func (c *serverCore) onJoinTimeout(e joinTimeoutFired) {
	if c.currentRound == nil {
		return
	}

	if c.roundState != RoundStateWaitingForPlayers {
		return
	}

	if c.currentRound.RoundNumber != e.roundNumber || c.currentRound.Token != e.token {
		return
	}

	if len(c.currentRound.JoinedPlayers) < minPlayersToStart {
		slog.Warn("Not enough players joined, aborting round", "joined", len(c.currentRound.JoinedPlayers))
		c.roundState = RoundStateIdle
		c.currentRound = nil
		c.joinTimer = nil

		c.tryStartRound()

		return
	}

	c.startRound()
}

func (c *serverCore) startRound() {
	if c.currentRound == nil {
		return
	}

	playerNames := make([]string, 0, len(c.currentRound.JoinedPlayers))
	for name := range c.currentRound.JoinedPlayers {
		playerNames = append(playerNames, name)
	}

	message := fmt.Sprintf(
		"%s;%d;%s\n",
		protocol.CmdRoundStarted,
		c.currentRound.RoundNumber,
		strings.Join(playerNames, ","),
	)

	c.broadcastToPlayers(message, c.currentRound.JoinedPlayers)
	slog.Info("Round started", "round", c.currentRound.RoundNumber, "players", playerNames)

	c.roundState = RoundStateInProgress
	c.joinTimer = nil
}

func (c *serverCore) handleJoin(token string, peer transport.PeerID) {
	expectedToken := ""
	if c.currentRound != nil {
		expectedToken = c.currentRound.Token
	}

	if c.currentRound == nil || c.currentRound.Token != token {
		c.sendToPeer(string(protocol.CmdRejected+";Invalid token\n"), peer)
		slog.Warn(
			"Player tried to join with invalid token",
			"peer",
			string(peer),
			"token",
			token,
			"expectedToken",
			expectedToken,
		)

		return
	}

	if c.roundState != RoundStateWaitingForPlayers {
		c.sendToPeer(string(protocol.CmdRejected+";Round not accepting players\n"), peer)
		slog.Warn("Player tried to join but round not accepting", "peer", string(peer))

		return
	}

	var playerName string

	for name, player := range c.players {
		if player.Peer == peer {
			playerName = name

			break
		}
	}

	if playerName == "" {
		c.sendToPeer(string(protocol.CmdRejected+";Player not registered\n"), peer)
		slog.Warn("Unregistered player tried to join", "peer", string(peer))

		return
	}

	if _, exists := c.currentRound.JoinedPlayers[playerName]; exists {
		slog.Info("Player already joined", "name", playerName)

		return
	}

	c.currentRound.JoinedPlayers[playerName] = c.players[playerName]
	slog.Info("Player joined round", "name", playerName, "round", c.currentRound.RoundNumber)
}

func (c *serverCore) broadcastToPlayers(message string, players map[string]*Player) {
	for _, player := range players {
		c.sendToPeer(message, player.Peer)
	}
}
