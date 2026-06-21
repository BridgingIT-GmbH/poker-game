package server

import (
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

func TestHandleJoin(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	peer := transport.PeerID("peer-3")
	core.players["TestPlayer"] = &Player{Name: "TestPlayer", Peer: peer}

	round := game.NewRound(1)
	session, _ := NewRoundSession()
	session.PendingTokens["TestPlayer"] = "valid-token"

	core.currentRound = round
	core.currentSession = session
	core.roundState = RoundStateWaitingForPlayers

	core.handleJoin("valid-token", peer)

	_, joined := core.currentRound.JoinedPlayers["TestPlayer"]
	if !joined {
		t.Fatal("player should have joined the round")
	}
}

func TestHandleJoinInvalidToken(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	peer := transport.PeerID("peer-4")
	core.players["TestPlayer"] = &Player{Name: "TestPlayer", Peer: peer}

	round := game.NewRound(1)
	session, _ := NewRoundSession()
	session.PendingTokens["TestPlayer"] = "correct-token"

	core.currentRound = round
	core.currentSession = session
	core.roundState = RoundStateWaitingForPlayers

	core.handleJoin("invalid-token", peer)

	pkg := readOutbound(t, outbound)
	assertOutbound(t, pkg, peer, string(protocol.CmdRejected+";Invalid token\n"))
}

func TestHandleJoinNotRegistered(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	peer := transport.PeerID("peer-5")

	round := game.NewRound(1)
	session, _ := NewRoundSession()

	core.currentRound = round
	core.currentSession = session
	core.roundState = RoundStateWaitingForPlayers

	core.handleJoin("any-token", peer)

	pkg := readOutbound(t, outbound)
	assertOutbound(t, pkg, peer, string(protocol.CmdRejected+";Player not registered\n"))

	_ = model.Player{} // keep model import used
}
