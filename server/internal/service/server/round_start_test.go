package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

func TestTryStartRound_NotEnoughPlayers(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)
	core.players["Alice"] = &model.Client{Name: "Alice", Peer: "peer-alice"}

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

	core.players["Alice"] = &model.Client{Name: "Alice", Peer: "peer-alice"}
	core.players["Bob"] = &model.Client{Name: "Bob", Peer: "peer-bob"}
	core.roundState = RoundStateInProgress

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
	core.players["Alice"] = &model.Client{Name: "Alice", Peer: peerAlice}
	core.players["Bob"] = &model.Client{Name: "Bob", Peer: peerBob}

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

	got := map[transport.PeerID]string{}

	for range core.players {
		pkg := readOutbound(t, outbound)
		got[pkg.Peer] = string(pkg.Data)
	}

	aliceToken, ok := core.currentSession.PendingTokens["Alice"]
	if !ok {
		t.Fatal("no pending token for Alice")
	}

	expectedAlice := fmt.Sprintf("%s;%s\n", protocol.CmdRoundStarting, aliceToken)
	if got[peerAlice] != expectedAlice {
		t.Fatalf("Alice: expected %q, got %q", expectedAlice, got[peerAlice])
	}

	bobToken, ok := core.currentSession.PendingTokens["Bob"]
	if !ok {
		t.Fatal("no pending token for Bob")
	}

	expectedBob := fmt.Sprintf("%s;%s\n", protocol.CmdRoundStarting, bobToken)
	if got[peerBob] != expectedBob {
		t.Fatalf("Bob: expected %q, got %q", expectedBob, got[peerBob])
	}

	if aliceToken == bobToken {
		t.Fatal("expected unique tokens for each player, got identical tokens")
	}

	core.joinTimer.Stop()
}

func TestOnJoinTimeout_StaleRoundNumber(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)

	round := game.NewRound(1)
	session, _ := NewRoundSession()
	core.currentRound = round
	core.currentSession = session
	core.roundState = RoundStateWaitingForPlayers

	core.onJoinTimeout(joinTimeoutFired{roundNumber: 99, roundID: session.RoundID()})

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

	round := game.NewRound(1)
	session, _ := NewRoundSession()
	core.currentRound = round
	core.currentSession = session
	core.roundState = RoundStateWaitingForPlayers

	core.onJoinTimeout(joinTimeoutFired{roundNumber: round.RoundNumber, roundID: "wrong-id"})

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
	core.players["Alice"] = &model.Client{Name: "Alice", Peer: "peer-alice"}

	round := game.NewRound(1)
	session, _ := NewRoundSession()
	core.currentRound = round
	core.currentSession = session
	core.roundState = RoundStateWaitingForPlayers

	core.onJoinTimeout(joinTimeoutFired{roundNumber: round.RoundNumber, roundID: session.RoundID()})

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

	outbound := make(chan transport.OutboundPackage, 16)
	core := newServerCore(outbound)

	core.players["Alice"] = &model.Client{Name: "Alice", Peer: transport.PeerID("peer-alice")}
	core.players["Bob"] = &model.Client{Name: "Bob", Peer: transport.PeerID("peer-bob")}

	round := game.NewRound(1)
	session, _ := NewRoundSession()
	round.JoinedPlayers["Alice"] = &model.Player{Name: "Alice"}
	round.JoinedPlayers["Bob"] = &model.Player{Name: "Bob"}

	core.currentRound = round
	core.currentSession = session
	core.roundState = RoundStateWaitingForPlayers

	core.onJoinTimeout(joinTimeoutFired{roundNumber: round.RoundNumber, roundID: session.RoundID()})

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
	core.startRound()

	if len(outbound) != 0 {
		t.Fatal("expected no outbound messages when currentRound is nil")
	}
}

func TestStartRound_Success(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 16)
	core := newServerCore(outbound)

	peerAlice := transport.PeerID("peer-alice")
	peerBob := transport.PeerID("peer-bob")

	core.players["Alice"] = &model.Client{Name: "Alice", Peer: peerAlice}
	core.players["Bob"] = &model.Client{Name: "Bob", Peer: peerBob}

	round := game.NewRound(1)
	session, _ := NewRoundSession()
	round.JoinedPlayers["Alice"] = &model.Player{Name: "Alice"}
	round.JoinedPlayers["Bob"] = &model.Player{Name: "Bob"}

	core.currentRound = round
	core.currentSession = session
	core.roundState = RoundStateWaitingForPlayers

	core.startRound()

	if core.roundState != RoundStateInProgress {
		t.Fatalf("expected state InProgress, got %v", core.roundState)
	}

	if core.joinTimer != nil {
		t.Fatal("joinTimer should be nil after startRound")
	}

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
