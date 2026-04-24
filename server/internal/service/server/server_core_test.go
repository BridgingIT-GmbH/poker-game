package server

import (
	"testing"
	"time"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

func readOutbound(t *testing.T, outbound <-chan transport.OutboundPackage) transport.OutboundPackage {
	t.Helper()

	select {
	case pkg := <-outbound:
		return pkg
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for outbound package")

		return transport.OutboundPackage{}
	}
}

func assertOutbound(t *testing.T, pkg transport.OutboundPackage, expectedPeer transport.PeerID, expectedMsg string) {
	t.Helper()

	if pkg.Peer != expectedPeer {
		t.Fatalf("expected peer %q, got %q", expectedPeer, pkg.Peer)
	}

	if string(pkg.Data) != expectedMsg {
		t.Fatalf("expected message %q, got %q", expectedMsg, string(pkg.Data))
	}
}

func TestRegisterPlayer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		playerName     string
		existingPlayer *Player
		expectedMsg    string
		shouldRegister bool
	}{
		{"valid player", "Adam", nil, string(protocol.CmdRegistered + "\n"), true},
		{"empty name", "", nil, string(protocol.CmdRejected + ";Name is empty\n"), false},
		{
			"duplicate name",
			"Bob",
			&Player{Name: "Bob", Peer: "known"},
			string(protocol.CmdRejected + ";Name already taken\n"),
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			outbound := make(chan transport.OutboundPackage, 8)
			core := newServerCore(outbound)

			if test.existingPlayer != nil {
				core.players[test.existingPlayer.Name] = test.existingPlayer
			}

			peer := transport.PeerID("peer-1")
			core.registerPlayer(test.playerName, peer)

			pkg := readOutbound(t, outbound)
			assertOutbound(t, pkg, peer, test.expectedMsg)

			_, exists := core.players[test.playerName]
			if test.shouldRegister && !exists {
				t.Fatal("player should be registered but was not")
			}

			if !test.shouldRegister &&
				exists &&
				test.playerName != "" &&
				(test.existingPlayer == nil || test.playerName != test.existingPlayer.Name) {
				t.Fatal("player should not be registered but was")
			}
		})
	}
}

func TestHandleMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		message     []byte
		expectedMsg string
	}{
		{"valid register", []byte("REGISTER;TestPlayer"), string(protocol.CmdRegistered + "\n")},
		{"invalid format", []byte("REGISTER"), string(protocol.CmdRejected + ";invalid message format\n")},
		{"unknown command", []byte("UNKNOWN;Player"), string(protocol.CmdRejected + ";Unknown command\n")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			outbound := make(chan transport.OutboundPackage, 8)
			core := newServerCore(outbound)

			peer := transport.PeerID("peer-2")
			core.handleMessage(test.message, peer)

			pkg := readOutbound(t, outbound)
			assertOutbound(t, pkg, peer, test.expectedMsg)
		})
	}
}

func TestHandleJoin(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	peer := transport.PeerID("peer-3")
	core.players["TestPlayer"] = &Player{Name: "TestPlayer", Peer: peer}

	round, _ := NewRound(1)
	core.currentRound = round
	core.roundState = RoundStateWaitingForPlayers

	core.handleJoin(round.Token, peer)

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

	round, _ := NewRound(1)
	core.currentRound = round
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

	round, _ := NewRound(1)
	core.currentRound = round
	core.roundState = RoundStateWaitingForPlayers

	core.handleJoin(round.Token, peer)

	pkg := readOutbound(t, outbound)
	assertOutbound(t, pkg, peer, string(protocol.CmdRejected+";Player not registered\n"))
}

func TestBroadcastToPlayers(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	players := map[string]*Player{
		"Player1": {Name: "Player1", Peer: "peer-1"},
		"Player2": {Name: "Player2", Peer: "peer-2"},
	}

	msg := "BROADCAST_TEST\n"
	core.broadcastToPlayers(msg, players)

	pkg1 := readOutbound(t, outbound)
	pkg2 := readOutbound(t, outbound)

	// order is not guaranteed
	got := map[transport.PeerID]string{pkg1.Peer: string(pkg1.Data), pkg2.Peer: string(pkg2.Data)}
	if got["peer-1"] != msg {
		t.Fatalf("peer-1: expected %q, got %q", msg, got["peer-1"])
	}

	if got["peer-2"] != msg {
		t.Fatalf("peer-2: expected %q, got %q", msg, got["peer-2"])
	}
}
