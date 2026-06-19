package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

func TestTryStartRound_NotEnoughPlayers(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	// Only one player registered – below the minimum.
	core.players["Alice"] = &Player{Name: "Alice", Peer: "peer-alice"}

	core.tryStartRound()

	if core.currentRound != nil {
		t.Fatal("expected no round to be created with only one player")
	}

	if core.roundState != RoundStateIdle {
		t.Fatalf("expected state Idle, got %v", core.roundState)
	}

	if len(outbound) != 0 {
		t.Fatal("expected no outbound messages")
	}
}

func TestTryStartRound_AlreadyInProgress(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	core.players["Alice"] = &Player{Name: "Alice", Peer: "peer-alice"}
	core.players["Bob"] = &Player{Name: "Bob", Peer: "peer-bob"}
	core.roundState = RoundStateInProgress // not Idle

	core.tryStartRound()

	if core.currentRound != nil {
		t.Fatal("expected no new round while one is already in progress")
	}

	if len(outbound) != 0 {
		t.Fatal("expected no outbound messages")
	}
}

func TestTryStartRound_Success(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	peerAlice := transport.PeerID("peer-alice")
	peerBob := transport.PeerID("peer-bob")
	core.players["Alice"] = &Player{Name: "Alice", Peer: peerAlice}
	core.players["Bob"] = &Player{Name: "Bob", Peer: peerBob}

	core.tryStartRound()

	if core.currentRound == nil {
		t.Fatal("expected a round to be created")
	}

	if core.roundState != RoundStateWaitingForPlayers {
		t.Fatalf("expected state WaitingForPlayers, got %v", core.roundState)
	}

	if core.roundNumber != 1 {
		t.Fatalf("expected roundNumber 1, got %d", core.roundNumber)
	}

	if core.joinTimer == nil {
		t.Fatal("expected join timer to be set")
	}

	// Both players must receive a ROUND STARTING broadcast.
	expectedMsg := fmt.Sprintf("%s;%s\n", protocol.CmdRoundStarting, core.currentRound.Token)

	got := map[transport.PeerID]string{}

	for range core.players {
		pkg := readOutbound(t, outbound)
		got[pkg.Peer] = string(pkg.Data)
	}

	if got[peerAlice] != expectedMsg {
		t.Fatalf("Alice: expected %q, got %q", expectedMsg, got[peerAlice])
	}

	if got[peerBob] != expectedMsg {
		t.Fatalf("Bob: expected %q, got %q", expectedMsg, got[peerBob])
	}

	// Stop the timer so it doesn't interfere with other tests.
	core.joinTimer.Stop()
}

func TestOnJoinTimeout_StaleRoundNumber(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	round, _ := NewRound(1)
	core.currentRound = round
	core.roundState = RoundStateWaitingForPlayers

	// Fire event with a different round number – must be ignored.
	core.onJoinTimeout(joinTimeoutFired{roundNumber: 99, token: round.Token})

	if core.roundState != RoundStateWaitingForPlayers {
		t.Fatalf("state should be unchanged, got %v", core.roundState)
	}

	if len(outbound) != 0 {
		t.Fatal("expected no outbound messages for stale event")
	}
}

func TestOnJoinTimeout_StaleToken(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	round, _ := NewRound(1)
	core.currentRound = round
	core.roundState = RoundStateWaitingForPlayers

	// Fire event with a wrong token – must be ignored.
	core.onJoinTimeout(joinTimeoutFired{roundNumber: round.RoundNumber, token: "wrong-token"})

	if core.roundState != RoundStateWaitingForPlayers {
		t.Fatalf("state should be unchanged, got %v", core.roundState)
	}

	if len(outbound) != 0 {
		t.Fatal("expected no outbound messages for stale event")
	}
}

func TestOnJoinTimeout_NotEnoughPlayersJoined(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	// One registered player – not enough to trigger a new round after the abort.
	core.players["Alice"] = &Player{Name: "Alice", Peer: "peer-alice"}

	round, _ := NewRound(1)
	core.currentRound = round
	core.roundState = RoundStateWaitingForPlayers
	// Zero players joined – below minPlayersToStart.

	core.onJoinTimeout(joinTimeoutFired{roundNumber: round.RoundNumber, token: round.Token})

	if core.currentRound != nil {
		t.Fatal("round should have been aborted")
	}

	if core.roundState != RoundStateIdle {
		t.Fatalf("expected state Idle after abort, got %v", core.roundState)
	}

	if core.joinTimer != nil {
		t.Fatal("joinTimer should be nil after abort")
	}
}

func assertStartedRoundMessages(
	t *testing.T,
	outbound <-chan transport.OutboundPackage,
	n, wantStarted, wantCards, wantYourTurn int,
) {
	t.Helper()

	roundStartedCount, cardsCount, yourTurnCount := 0, 0, 0

	for range n {
		pkg := readOutbound(t, outbound)
		data := string(pkg.Data)

		switch {
		case strings.HasPrefix(data, string(protocol.CmdRoundStarted)+";"):
			roundStartedCount++
		case strings.HasPrefix(data, string(protocol.CmdCards)+";"):
			cardsCount++
		case strings.HasPrefix(data, string(protocol.CmdYourTurn)+";"):
			yourTurnCount++
		default:
			t.Fatalf("unexpected message %q", data)
		}
	}

	if roundStartedCount != wantStarted {
		t.Errorf("expected %d ROUND STARTED messages, got %d", wantStarted, roundStartedCount)
	}

	if cardsCount != wantCards {
		t.Errorf("expected %d CARDS messages, got %d", wantCards, cardsCount)
	}

	if yourTurnCount != wantYourTurn {
		t.Errorf("expected %d YOUR TURN messages, got %d", wantYourTurn, yourTurnCount)
	}
}

func TestOnJoinTimeout_EnoughPlayersJoined(t *testing.T) {
	t.Parallel()

	// 2× ROUND STARTED + 2× CARDS + 1× YOUR TURN = 5 messages
	outbound := make(chan transport.OutboundPackage, 16)
	core := newServerCore(outbound)

	playerAlice := &Player{Name: "Alice", Peer: transport.PeerID("peer-alice")}
	playerBob := &Player{Name: "Bob", Peer: transport.PeerID("peer-bob")}

	round, _ := NewRound(1)
	round.JoinedPlayers["Alice"] = playerAlice
	round.JoinedPlayers["Bob"] = playerBob

	core.currentRound = round
	core.roundState = RoundStateWaitingForPlayers

	core.onJoinTimeout(joinTimeoutFired{roundNumber: round.RoundNumber, token: round.Token})

	if core.roundState != RoundStateInProgress {
		t.Fatalf("expected state InProgress, got %v", core.roundState)
	}

	if core.joinTimer != nil {
		t.Fatal("joinTimer should be nil after round started")
	}

	assertStartedRoundMessages(t, outbound, 5, 2, 2, 1)
}

func TestStartRound_NilRound(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)
	// currentRound is nil – must not panic.
	core.startRound()

	if len(outbound) != 0 {
		t.Fatal("expected no outbound messages when currentRound is nil")
	}
}

func TestStartRound_Success(t *testing.T) {
	t.Parallel()

	// 2× ROUND STARTED + 2× CARDS + 1× YOUR TURN = 5 messages
	outbound := make(chan transport.OutboundPackage, 16)
	core := newServerCore(outbound)

	peerAlice := transport.PeerID("peer-alice")
	peerBob := transport.PeerID("peer-bob")

	round, _ := NewRound(1)
	round.JoinedPlayers["Alice"] = &Player{Name: "Alice", Peer: peerAlice}
	round.JoinedPlayers["Bob"] = &Player{Name: "Bob", Peer: peerBob}
	core.currentRound = round
	core.roundState = RoundStateWaitingForPlayers

	core.startRound()

	if core.roundState != RoundStateInProgress {
		t.Fatalf("expected state InProgress, got %v", core.roundState)
	}

	if core.joinTimer != nil {
		t.Fatal("joinTimer should be nil after startRound")
	}

	// Collect all 5 messages.
	const totalMessages = 5

	pkgs := make([]transport.OutboundPackage, totalMessages)
	for i := range pkgs {
		pkgs[i] = readOutbound(t, outbound)
	}

	roundStartedCount := 0

	for _, pkg := range pkgs {
		data := string(pkg.Data)
		if !strings.HasPrefix(data, string(protocol.CmdRoundStarted)+";") {
			continue
		}

		roundStartedCount++

		if !strings.HasPrefix(data, string(protocol.CmdRoundStarted)+";1;") {
			t.Fatalf("peer %q: unexpected message %q", pkg.Peer, data)
		}

		if !strings.Contains(data, "Alice") || !strings.Contains(data, "Bob") {
			t.Fatalf("peer %q: player names missing in message %q", pkg.Peer, data)
		}
	}

	if roundStartedCount != 2 {
		t.Fatalf("expected 2 ROUND STARTED messages, got %d", roundStartedCount)
	}
}
