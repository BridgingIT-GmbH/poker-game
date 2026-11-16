package server

import (
	"strings"
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

const twoPlayerDealMessages = 3

func drainOutbound(t *testing.T, outbound <-chan transport.OutboundPackage) []transport.OutboundPackage {
	t.Helper()

	pkgs := make([]transport.OutboundPackage, twoPlayerDealMessages)
	for i := range pkgs {
		pkgs[i] = readOutbound(t, outbound)
	}

	return pkgs
}

const (
	testPeerAlice = transport.PeerID("peer-alice")
	testPeerBob   = transport.PeerID("peer-bob")
)

func testPlayerNames() []string {
	return []string{"Alice", "Bob"}
}

// setupDealTest creates a core and round ready for dealCards() with two players.
func setupDealTest(
	t *testing.T,
	roundNumber int,
) (*serverCore, *game.Round, chan transport.OutboundPackage) {
	t.Helper()

	outbound := make(chan transport.OutboundPackage, 16)
	core := newServerCore(outbound)

	core.players["Alice"] = &model.Client{Name: "Alice", Peer: testPeerAlice}
	core.players["Bob"] = &model.Client{Name: "Bob", Peer: testPeerBob}

	round := game.NewRound(roundNumber)
	session, _ := NewRoundSession()
	round.JoinedPlayers["Alice"] = &model.Player{Name: "Alice"}
	round.JoinedPlayers["Bob"] = &model.Player{Name: "Bob"}

	core.currentRound = round
	core.currentSession = session
	core.roundState = RoundStateInProgress

	return core, round, outbound
}

func TestDealCards_NilRound(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)
	core.dealCards(nil)

	if len(outbound) != 0 {
		t.Fatal("expected no outbound messages when currentRound is nil")
	}
}

func TestDealCards_EachPlayerReceivesCards(t *testing.T) {
	t.Parallel()

	core, _, outbound := setupDealTest(t, 1)

	core.dealCards(testPlayerNames())

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

	if _, ok := cardsReceived[testPeerAlice]; !ok {
		t.Error("Alice did not receive a CARDS message")
	}

	if _, ok := cardsReceived[testPeerBob]; !ok {
		t.Error("Bob did not receive a CARDS message")
	}

	if len(yourTurnReceived) != 1 {
		t.Fatalf("expected exactly 1 YOUR TURN message, got %d", len(yourTurnReceived))
	}
}

func TestDealCards_CardsFormat(t *testing.T) {
	t.Parallel()

	core, _, outbound := setupDealTest(t, 1)
	core.dealCards(testPlayerNames())

	pkgs := drainOutbound(t, outbound)

	for _, pkg := range pkgs {
		data := string(pkg.Data)
		if !strings.HasPrefix(data, string(protocol.CmdCards)+";") {
			continue
		}

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

	core, _, outbound := setupDealTest(t, 1)
	core.dealCards(testPlayerNames())

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

	core, round, outbound := setupDealTest(t, 2)
	core.dealCards(testPlayerNames())
	drainOutbound(t, outbound)

	if round.Table == nil {
		t.Fatal("round.Table must be populated after dealCards()")
	}

	if len(round.PlayerOrder) != 2 {
		t.Fatalf("expected PlayerOrder length 2, got %d", len(round.PlayerOrder))
	}

	orderSet := map[string]bool{}
	for _, name := range round.PlayerOrder {
		orderSet[name] = true
	}

	if !orderSet["Alice"] || !orderSet["Bob"] {
		t.Fatalf("PlayerOrder missing a player: %v", round.PlayerOrder)
	}

	if round.Table.DealerPosition != 0 {
		t.Fatalf("expected DealerPosition 0, got %d", round.Table.DealerPosition)
	}

	const expectedRemaining = 48
	if len(round.Table.Deck.Cards) != expectedRemaining {
		t.Fatalf("expected %d remaining cards, got %d", expectedRemaining, len(round.Table.Deck.Cards))
	}
}

func TestDealCards_YourTurnContainsToken(t *testing.T) {
	t.Parallel()

	core, round, outbound := setupDealTest(t, 1)
	core.dealCards(testPlayerNames())

	pkgs := drainOutbound(t, outbound)

	var yourTurnPeer transport.PeerID

	var yourTurnToken string

	for _, pkg := range pkgs {
		data := string(pkg.Data)
		if strings.HasPrefix(data, string(protocol.CmdYourTurn)+";") {
			yourTurnPeer = pkg.Peer
			yourTurnToken = strings.TrimSuffix(strings.TrimPrefix(data, string(protocol.CmdYourTurn)+";"), "\n")
		}
	}

	if yourTurnPeer == "" {
		t.Fatal("no YOUR TURN message found in outbound packages")
	}

	// Find the player name for that peer via core.players.
	var actingPlayer string

	for name, p := range core.players {
		if p.Peer == yourTurnPeer {
			actingPlayer = name
		}
	}

	// The stored pending token in the session must match the token sent.
	expectedToken, ok := core.currentSession.PendingTokens[actingPlayer]
	if !ok {
		t.Fatalf("no pending token stored for acting player %q", actingPlayer)
	}

	if yourTurnToken != expectedToken {
		t.Fatalf("YOUR TURN token %q does not match PendingTokens[%q] = %q",
			yourTurnToken, actingPlayer, expectedToken)
	}

	_ = round
}

func TestDealCards_SpectatorsReceiveTurn(t *testing.T) {
	t.Parallel()

	const testPeerSpectator = transport.PeerID("peer-spectator")

	core, round, outbound := setupDealTest(t, 1)
	core.spectators["spectator_1"] = &model.Client{Name: "spectator_1", Peer: testPeerSpectator}

	core.dealCards(testPlayerNames())

	var turnMessages []string

	for range twoPlayerDealMessages + 1 {
		pkg := readOutbound(t, outbound)
		if pkg.Peer == testPeerSpectator {
			turnMessages = append(turnMessages, string(pkg.Data))
		}
	}

	toAct := round.PlayerOrder[round.Table.CurrentToActPos]
	want := string(protocol.CmdTurn) + ";" + toAct + "\n"

	if len(turnMessages) != 1 || turnMessages[0] != want {
		t.Fatalf("expected spectator to receive exactly %q, got %q", want, turnMessages)
	}
}
