package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

// twoPlayerDealMessages is the number of outbound messages produced by
// dealCards() for a two-player round: 2× CARDS + 1× YOUR TURN.
const twoPlayerDealMessages = 3

// drainOutbound reads twoPlayerDealMessages packages from the channel,
// failing on timeout.
func drainOutbound(t *testing.T, outbound <-chan transport.OutboundPackage) []transport.OutboundPackage {
	t.Helper()

	pkgs := make([]transport.OutboundPackage, twoPlayerDealMessages)
	for i := range pkgs {
		pkgs[i] = readOutbound(t, outbound)
	}

	return pkgs
}

func TestDealCards_NilRound(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)
	// currentRound is nil – must not panic.
	core.dealCards()

	if len(outbound) != 0 {
		t.Fatal("expected no outbound messages when currentRound is nil")
	}
}

func TestDealCards_EachPlayerReceivesCards(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 16)
	core := newServerCore(outbound)

	peerAlice := transport.PeerID("peer-alice")
	peerBob := transport.PeerID("peer-bob")

	round, _ := NewRound(1)
	round.JoinedPlayers["Alice"] = &Player{Name: "Alice", Peer: peerAlice}
	round.JoinedPlayers["Bob"] = &Player{Name: "Bob", Peer: peerBob}
	core.currentRound = round
	core.roundState = RoundStateInProgress

	core.dealCards()

	// Expect: 2× CARDS + 1× YOUR TURN  (3 messages total)
	pkgs := drainOutbound(t, outbound)

	cardsReceived := map[transport.PeerID]string{}
	yourTurnReceived := map[transport.PeerID]string{}

	for _, pkg := range pkgs {
		data := string(pkg.Data)
		switch {
		case strings.HasPrefix(data, string(protocol.CmdCards)+";"):
			cardsReceived[pkg.Peer] = data
		case strings.HasPrefix(data, string(protocol.CmdYourTurn)+";"):
			yourTurnReceived[pkg.Peer] = data
		default:
			t.Fatalf("unexpected message from peer %q: %q", pkg.Peer, data)
		}
	}

	// Both players must have received a CARDS message.
	if _, ok := cardsReceived[peerAlice]; !ok {
		t.Error("Alice did not receive a CARDS message")
	}

	if _, ok := cardsReceived[peerBob]; !ok {
		t.Error("Bob did not receive a CARDS message")
	}

	// Exactly one YOUR TURN must have been sent.
	if len(yourTurnReceived) != 1 {
		t.Fatalf("expected exactly 1 YOUR TURN message, got %d", len(yourTurnReceived))
	}
}

func TestDealCards_CardsFormat(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 16)
	core := newServerCore(outbound)

	round, _ := NewRound(1)
	round.JoinedPlayers["Alice"] = &Player{Name: "Alice", Peer: "peer-alice"}
	round.JoinedPlayers["Bob"] = &Player{Name: "Bob", Peer: "peer-bob"}
	core.currentRound = round

	core.dealCards()

	pkgs := drainOutbound(t, outbound)

	for _, pkg := range pkgs {
		data := string(pkg.Data)
		if !strings.HasPrefix(data, string(protocol.CmdCards)+";") {
			continue
		}
		// Format: "CARDS;Xs,Yh\n"  – two comma-separated cards, one newline.
		trimmed := strings.TrimSuffix(data, "\n")

		parts := strings.SplitN(trimmed, ";", 2)
		if len(parts) != 2 {
			t.Fatalf("malformed CARDS message: %q", data)
		}

		cards := strings.Split(parts[1], ",")
		if len(cards) != 2 {
			t.Fatalf("expected 2 cards in CARDS message, got %q", parts[1])
		}

		for _, card := range cards {
			if len(card) != 2 {
				t.Fatalf("expected card notation of length 2, got %q", card)
			}
		}
	}
}

func TestDealCards_UniqueCards(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 16)
	core := newServerCore(outbound)

	round, _ := NewRound(1)
	round.JoinedPlayers["Alice"] = &Player{Name: "Alice", Peer: "peer-alice"}
	round.JoinedPlayers["Bob"] = &Player{Name: "Bob", Peer: "peer-bob"}
	core.currentRound = round

	core.dealCards()

	pkgs := drainOutbound(t, outbound)

	seen := map[string]bool{}

	for _, pkg := range pkgs {
		data := string(pkg.Data)
		if !strings.HasPrefix(data, string(protocol.CmdCards)+";") {
			continue
		}

		trimmed := strings.TrimSuffix(data, "\n")

		parts := strings.SplitN(trimmed, ";", 2)
		for _, card := range strings.Split(parts[1], ",") {
			if seen[card] {
				t.Fatalf("card %q dealt more than once", card)
			}

			seen[card] = true
		}
	}
}

func TestDealCards_TablePopulated(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 16)
	core := newServerCore(outbound)

	round, _ := NewRound(2) // round 2 → dealer rotates to seat 1
	round.JoinedPlayers["Alice"] = &Player{Name: "Alice", Peer: "peer-alice"}
	round.JoinedPlayers["Bob"] = &Player{Name: "Bob", Peer: "peer-bob"}
	core.currentRound = round

	core.dealCards()

	drainOutbound(t, outbound)

	if round.Table == nil {
		t.Fatal("round.Table must be populated after dealCards()")
	}

	if len(round.PlayerOrder) != 2 {
		t.Fatalf("expected PlayerOrder length 2, got %d", len(round.PlayerOrder))
	}

	// PlayerOrder must contain both players (order is random).
	orderSet := map[string]bool{}
	for _, name := range round.PlayerOrder {
		orderSet[name] = true
	}

	if !orderSet["Alice"] || !orderSet["Bob"] {
		t.Fatalf("PlayerOrder missing a player: %v", round.PlayerOrder)
	}

	// Dealer position for round 2 with 2 players: (2-1) % 2 = 1 → Bob.
	if round.Table.DealerPosition != 1 {
		t.Fatalf("expected DealerPosition 1, got %d", round.Table.DealerPosition)
	}

	// Remaining deck must have 52 - 4 dealt cards = 48 cards.
	const expectedRemaining = 48
	if len(round.Table.Deck.Cards) != expectedRemaining {
		t.Fatalf("expected %d remaining cards, got %d", expectedRemaining, len(round.Table.Deck.Cards))
	}
}

func TestDealCards_YourTurnContainsToken(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 16)
	core := newServerCore(outbound)

	round, _ := NewRound(1)
	round.JoinedPlayers["Alice"] = &Player{Name: "Alice", Peer: "peer-alice"}
	round.JoinedPlayers["Bob"] = &Player{Name: "Bob", Peer: "peer-bob"}
	core.currentRound = round

	core.dealCards()

	pkgs := drainOutbound(t, outbound)

	expectedYourTurn := fmt.Sprintf("%s;%s\n", protocol.CmdYourTurn, round.Token)
	found := false

	for _, pkg := range pkgs {
		if string(pkg.Data) == expectedYourTurn {
			found = true

			break
		}
	}

	if !found {
		t.Fatalf("expected YOUR TURN message %q not found in outbound packages", expectedYourTurn)
	}
}
