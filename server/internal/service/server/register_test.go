package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

func TestRegisterPlayer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		playerName     string
		existingPlayer *model.Client
		expectedMsg    string
		shouldRegister bool
	}{
		{"valid player", "Adam", nil, string(protocol.CmdRegistered + "\n"), true},
		{"empty name", "", nil, string(protocol.CmdRejected + ";Name is empty\n"), false},
		{
			"reconnect same name",
			"Bob",
			&model.Client{Name: "Bob", Peer: "old-peer"},
			string(protocol.CmdRegistered + "\n"),
			true,
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

func TestRegisterPlayer_InvalidName_Rejected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		playerName string
		reason     string
	}{
		{"too long", "abcdefghijklm", "Name is too long (max 12 characters)"},
		{"comma", "Al,ice", "Name contains forbidden characters (; , :)"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			outbound := make(chan transport.OutboundPackage, 8)
			core := newServerCore(outbound)
			peer := transport.PeerID("peer-1")

			core.registerPlayer(test.playerName, peer)

			pkg := readOutbound(t, outbound)
			assertOutbound(t, pkg, peer, string(protocol.CmdRejected)+";"+test.reason+"\n")

			if _, exists := core.players[test.playerName]; exists {
				t.Fatal("player with invalid name should not be registered")
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "Alice", false},
		{"exactly 12", "abcdefghijkl", false},
		{"umlauts count as one", "Jürgenjürgen", false},
		{"empty", "", true},
		{"too long", "abcdefghijklm", true},
		{"comma", "Al,ice", true},
		{"colon", "Al:ice", true},
		{"semicolon", "Al;ice", true},
		{"newline", "Al\nice", true},
		{"tab", "Al\tice", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := validateName(test.input) != ""; got != test.wantErr {
				t.Fatalf("validateName(%q) error = %v, want %v", test.input, got, test.wantErr)
			}
		})
	}
}

// ── reconnect: peer update ────────────────────────────────────────────────────

func TestRegisterPlayer_Reconnect_UpdatesPeer(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)
	core.players["Alice"] = &model.Client{Name: "Alice", Peer: "old-peer"}

	newPeer := transport.PeerID("new-peer")
	core.registerPlayer("Alice", newPeer)

	if core.players["Alice"].Peer != newPeer {
		t.Fatalf("expected peer to be updated to %q, got %q", newPeer, core.players["Alice"].Peer)
	}
}

func TestRegisterPlayer_Reconnect_SendsRegisteredToNewPeer(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)
	core.players["Alice"] = &model.Client{Name: "Alice", Peer: "old-peer"}

	newPeer := transport.PeerID("new-peer")
	core.registerPlayer("Alice", newPeer)

	pkg := readOutbound(t, outbound)
	assertOutbound(t, pkg, newPeer, string(protocol.CmdRegistered+"\n"))
}

// ── reconnect: no round in progress ──────────────────────────────────────────

func TestRegisterPlayer_Reconnect_NoRound_NoExtraMessages(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)
	core.players["Alice"] = &model.Client{Name: "Alice", Peer: "old-peer"}
	// roundState is Idle (default) → resendRoundStartingIfNeeded is a no-op

	core.registerPlayer("Alice", "new-peer")

	readOutbound(t, outbound) // consume REGISTERED

	if len(outbound) != 0 {
		t.Fatal("expected no extra messages when no round is waiting")
	}
}

// ── reconnect during WaitingForPlayers ───────────────────────────────────────

func TestRegisterPlayer_Reconnect_WaitingForPlayers_ResendToken(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	oldPeer := transport.PeerID("old-peer")
	newPeer := transport.PeerID("new-peer")
	core.players["Alice"] = &model.Client{Name: "Alice", Peer: oldPeer}

	// Simulate a round waiting for Alice to join.
	session, _ := NewRoundSession()
	session.PendingTokens["Alice"] = "alice-join-token"
	core.currentSession = session
	core.roundState = RoundStateWaitingForPlayers

	core.registerPlayer("Alice", newPeer)

	readOutbound(t, outbound) // consume REGISTERED

	// Should receive a re-sent ROUND STARTING with the same token.
	pkg := readOutbound(t, outbound)

	if pkg.Peer != newPeer {
		t.Fatalf("expected ROUND STARTING sent to new peer %q, got %q", newPeer, pkg.Peer)
	}

	expected := fmt.Sprintf("%s;alice-join-token\n", protocol.CmdRoundStarting)
	if string(pkg.Data) != expected {
		t.Fatalf("expected %q, got %q", expected, string(pkg.Data))
	}
}

func TestRegisterPlayer_Reconnect_WaitingForPlayers_AlreadyJoined_NoResend(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)
	core.players["Alice"] = &model.Client{Name: "Alice", Peer: "old-peer"}

	// Alice has already joined (token consumed) → no pending token.
	session, _ := NewRoundSession()
	core.currentSession = session
	core.roundState = RoundStateWaitingForPlayers

	core.registerPlayer("Alice", "new-peer")

	readOutbound(t, outbound) // consume REGISTERED

	if len(outbound) != 0 {
		t.Fatal("expected no ROUND STARTING re-send when token was already consumed")
	}
}

// ── reconnect during InProgress ───────────────────────────────────────────────

func TestRegisterPlayer_Reconnect_InProgress_NoResend(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)
	core.players["Alice"] = &model.Client{Name: "Alice", Peer: "old-peer"}

	session, _ := NewRoundSession()
	session.PendingTokens["Alice"] = "some-action-token"
	core.currentSession = session
	core.roundState = RoundStateInProgress // not WaitingForPlayers

	core.registerPlayer("Alice", "new-peer")

	readOutbound(t, outbound) // consume REGISTERED

	if len(outbound) != 0 {
		t.Fatal("expected no ROUND STARTING re-send when round is already in progress")
	}
}

// ── resendRoundStartingIfNeeded: direct unit tests ────────────────────────────

func TestResendRoundStartingIfNeeded_NoSession(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)
	// currentSession is nil

	core.resendRoundStartingIfNeeded("Alice", "peer-alice")

	if len(outbound) != 0 {
		t.Fatal("expected no message when session is nil")
	}
}

func TestResendRoundStartingIfNeeded_WrongState(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	session, _ := NewRoundSession()
	session.PendingTokens["Alice"] = "token"
	core.currentSession = session
	core.roundState = RoundStateInProgress // must be WaitingForPlayers

	core.resendRoundStartingIfNeeded("Alice", "peer-alice")

	if len(outbound) != 0 {
		t.Fatal("expected no message when round state is not WaitingForPlayers")
	}
}

func TestResendRoundStartingIfNeeded_NoToken(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	session, _ := NewRoundSession()
	// No token stored for Alice.
	core.currentSession = session
	core.roundState = RoundStateWaitingForPlayers

	core.resendRoundStartingIfNeeded("Alice", "peer-alice")

	if len(outbound) != 0 {
		t.Fatal("expected no message when player has no pending token")
	}
}

func TestResendRoundStartingIfNeeded_SendsToken(t *testing.T) {
	t.Parallel()

	const testToken = "abc123"

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	session, _ := NewRoundSession()
	session.PendingTokens["Alice"] = testToken
	core.currentSession = session
	core.roundState = RoundStateWaitingForPlayers

	peer := transport.PeerID("peer-alice")
	core.resendRoundStartingIfNeeded("Alice", peer)

	pkg := readOutbound(t, outbound)

	if pkg.Peer != peer {
		t.Fatalf("expected message to peer %q, got %q", peer, pkg.Peer)
	}

	expected := fmt.Sprintf("%s;%s\n", protocol.CmdRoundStarting, testToken)
	if string(pkg.Data) != expected {
		t.Fatalf("expected %q, got %q", expected, string(pkg.Data))
	}
}

func TestResendRoundStartingIfNeeded_TokenNotConsumed(t *testing.T) {
	t.Parallel()

	const testToken = "abc123"

	// The token must NOT be consumed by resend – the player still needs it to JOIN.
	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	session, _ := NewRoundSession()
	session.PendingTokens["Alice"] = testToken
	core.currentSession = session
	core.roundState = RoundStateWaitingForPlayers

	core.resendRoundStartingIfNeeded("Alice", "peer-alice")
	readOutbound(t, outbound)

	if _, ok := session.PendingTokens["Alice"]; !ok {
		t.Fatal("token should still be present after resend (not consumed)")
	}

	// Calling again must produce another message (token still valid).
	core.resendRoundStartingIfNeeded("Alice", "peer-alice")

	pkg := readOutbound(t, outbound)

	if !strings.Contains(string(pkg.Data), testToken) {
		t.Fatalf("expected token in second resend, got %q", string(pkg.Data))
	}
}
