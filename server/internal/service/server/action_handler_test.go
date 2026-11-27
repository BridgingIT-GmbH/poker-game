package server

import (
	"strings"
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

// ── Shared test helpers ───────────────────────────────────────────────────────

// twoPlayerRoundInProgress sets up a serverCore with two players whose cards
// have already been dealt and whose YOUR TURN token has been issued to Alice
// (seat 0). Bob sits at seat 1.
func twoPlayerRoundInProgress(t *testing.T) (*serverCore, *game.Round, chan transport.OutboundPackage) {
	t.Helper()

	outbound := make(chan transport.OutboundPackage, 32)
	core := newServerCore(outbound)

	core.players["Alice"] = &model.Client{Name: "Alice", Peer: "peer-alice"}
	core.players["Bob"] = &model.Client{Name: "Bob", Peer: "peer-bob"}

	round := game.NewRound(1)
	session, _ := NewRoundSession()

	pgAlice := model.NewPlayer(0, "Alice", defaultChips)
	pgBob := model.NewPlayer(1, "Bob", defaultChips)
	pgAlice.IsActive = true
	pgBob.IsActive = true

	table := model.NewTable([]*model.Player{pgAlice, pgBob}, defaultSmallBlind, defaultBigBlind)
	table.DealerPosition = 0
	table.SmallBlindPos = 0
	table.BigBlindPos = 1
	table.CurrentToActPos = 0
	table.NumPlayersActive = 2

	game.CollectBlinds(table)

	round.Table = table
	round.PlayerOrder = []string{"Alice", "Bob"}
	round.JoinedPlayers["Alice"] = pgAlice
	round.JoinedPlayers["Bob"] = pgBob

	session.PendingTokens["Alice"] = "alice-token"

	core.currentRound = round
	core.currentSession = session
	core.roundState = RoundStateInProgress

	return core, round, outbound
}

// drainN reads exactly n messages from outbound, failing on timeout.
func drainN(t *testing.T, outbound <-chan transport.OutboundPackage, n int) []transport.OutboundPackage {
	t.Helper()

	pkgs := make([]transport.OutboundPackage, n)
	for i := range pkgs {
		pkgs[i] = readOutbound(t, outbound)
	}

	return pkgs
}

// firstWithPrefix returns true if any package's Data starts with prefix.
func firstWithPrefix(pkgs []transport.OutboundPackage, prefix string) bool {
	for _, pkg := range pkgs {
		if strings.HasPrefix(string(pkg.Data), prefix) {
			return true
		}
	}

	return false
}

// ── handleAction: guards ──────────────────────────────────────────────────────

func TestHandleAction_NoRound(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 8)
	core := newServerCore(outbound)
	core.players["Alice"] = &model.Client{Name: "Alice", Peer: "peer-alice"}

	core.handleAction(protocol.CmdFold, "any", "peer-alice")

	pkg := readOutbound(t, outbound)
	if !strings.Contains(string(pkg.Data), string(protocol.CmdRejected)) {
		t.Errorf("expected REJECTED when no round in progress, got %q", string(pkg.Data))
	}
}

func TestHandleAction_Fold_InvalidToken(t *testing.T) {
	t.Parallel()

	core, _, outbound := twoPlayerRoundInProgress(t)

	core.handleAction(protocol.CmdFold, "wrong-token", "peer-alice")

	pkg := readOutbound(t, outbound)
	if !strings.Contains(string(pkg.Data), string(protocol.CmdRejected)) {
		t.Errorf("expected REJECTED, got %q", string(pkg.Data))
	}
}

func TestHandleAction_Fold_NotYourTurn(t *testing.T) {
	t.Parallel()

	core, _, outbound := twoPlayerRoundInProgress(t)
	core.currentSession.PendingTokens["Bob"] = "bob-token"

	core.handleAction(protocol.CmdFold, "bob-token", "peer-bob")

	pkg := readOutbound(t, outbound)
	if !strings.Contains(string(pkg.Data), string(protocol.CmdRejected)) {
		t.Errorf("expected REJECTED for Bob acting out of turn, got %q", string(pkg.Data))
	}
}

// ── handleAction: FOLD ────────────────────────────────────────────────────────

func TestHandleAction_Fold(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)

	core.handleAction(protocol.CmdFold, "alice-token", "peer-alice")

	pkgs := drainN(t, outbound, 3) // 2× FOLD;Alice + 1× SCORE

	foldCount := 0

	for _, pkg := range pkgs {
		if string(pkg.Data) == "FOLD;Alice\n" {
			foldCount++
		}
	}

	if foldCount != 2 {
		t.Errorf("expected 2 FOLD;Alice broadcasts without reason, got %d", foldCount)
	}

	hasScore := firstWithPrefix(pkgs, string(protocol.CmdScore))
	if !hasScore {
		t.Error("expected SCORE message after last player stands")
	}

	if !round.Table.Players[0].HasFolded {
		t.Error("Alice should be marked as folded")
	}
}

// ── handleAction: CHECK ───────────────────────────────────────────────────────

func TestHandleAction_Check_Valid(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)
	round.Table.Players[0].CurrentBet = round.Table.CurrentBet // no outstanding bet

	core.handleAction(protocol.CmdCheck, "alice-token", "peer-alice")

	pkgs := drainN(t, outbound, 3) // 2× CHECK;Alice + 1× YOUR TURN

	hasCheck := firstWithPrefix(pkgs, "CHECK;Alice")
	if !hasCheck {
		t.Error("expected CHECK;Alice broadcast")
	}

	hasYourTurn := firstWithPrefix(pkgs, string(protocol.CmdYourTurn))
	if !hasYourTurn {
		t.Error("expected YOUR TURN message for Bob")
	}

	if !round.Table.Players[0].HasActed {
		t.Error("Alice should be marked as having acted")
	}
}

func TestHandleAction_Check_InvalidAutoFold(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)
	round.Table.Players[0].CurrentBet = 0 // outstanding bet → check invalid → auto-fold

	core.handleAction(protocol.CmdCheck, "alice-token", "peer-alice")

	pkgs := drainN(t, outbound, 3) // 2× FOLD;Alice + 1× SCORE

	hasFold := firstWithPrefix(pkgs, "FOLD;Alice;CANNOT_CHECK")
	if !hasFold {
		t.Error("expected auto-fold broadcast when check is invalid")
	}
}

// ── handleAction: CALL ────────────────────────────────────────────────────────

func TestHandleAction_Call(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)
	table := round.Table

	initialChips := table.Players[0].Chips
	callAmount := table.CurrentBet - table.Players[0].CurrentBet

	core.handleAction(protocol.CmdCall, "alice-token", "peer-alice")

	pkgs := drainN(t, outbound, 3) // 2× CALL;Alice + 1× YOUR TURN

	hasCall := firstWithPrefix(pkgs, "CALL;Alice")
	if !hasCall {
		t.Error("expected CALL;Alice broadcast")
	}

	if table.Players[0].Chips != initialChips-callAmount {
		t.Errorf("expected chips %d, got %d", initialChips-callAmount, table.Players[0].Chips)
	}

	if table.Pot != defaultSmallBlind+defaultBigBlind+callAmount {
		t.Errorf("unexpected pot value: %d", table.Pot)
	}
}

func TestHandleAction_Call_InsufficientChips_AutoFold(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)
	round.Table.Players[0].Chips = 1
	round.Table.Players[0].CurrentBet = 0
	round.Table.CurrentBet = 20

	core.handleAction(protocol.CmdCall, "alice-token", "peer-alice")

	pkgs := drainN(t, outbound, 3) // 2× FOLD;Alice + 1× SCORE

	hasFold := firstWithPrefix(pkgs, "FOLD;Alice;CANNOT_AFFORD_CALL")
	if !hasFold {
		t.Error("expected auto-fold when player cannot afford the call")
	}
}

// ── handleAction: RAISE ───────────────────────────────────────────────────────

func TestHandleAction_Raise(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)
	table := round.Table

	initialPot := table.Pot
	expectedPay := table.CurrentBet + table.BigBlindAmount - table.Players[0].CurrentBet

	core.handleAction(protocol.CmdRaise, "alice-token", "peer-alice")

	pkgs := drainN(t, outbound, 3) // 2× RAISE;Alice + 1× YOUR TURN

	hasRaise := firstWithPrefix(pkgs, "RAISE;Alice")
	if !hasRaise {
		t.Error("expected RAISE;Alice broadcast")
	}

	if table.Pot != initialPot+expectedPay {
		t.Errorf("expected pot %d, got %d", initialPot+expectedPay, table.Pot)
	}

	if table.CurrentBet != defaultBigBlind+defaultBigBlind {
		t.Errorf("expected new current bet %d, got %d", defaultBigBlind+defaultBigBlind, table.CurrentBet)
	}

	if table.Players[1].HasActed {
		t.Error("Bob should need to act again after Alice's raise")
	}
}

func TestHandleAction_Raise_LimitEnforced(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)
	round.RaisesThisStreet["Alice"] = 1 // max raises per street = 1

	core.handleAction(protocol.CmdRaise, "alice-token", "peer-alice")

	pkgs := drainN(t, outbound, 3) // 2× FOLD;Alice + 1× SCORE

	hasFold := firstWithPrefix(pkgs, "FOLD;Alice;RAISE_LIMIT")
	if !hasFold {
		t.Error("expected auto-fold when raise limit is reached")
	}
}

func TestHandleAction_Raise_CounterIncremented(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)

	core.handleAction(protocol.CmdRaise, "alice-token", "peer-alice")
	drainN(t, outbound, 3)

	if round.RaisesThisStreet["Alice"] != 1 {
		t.Errorf("expected RaisesThisStreet[Alice]=1, got %d", round.RaisesThisStreet["Alice"])
	}
}

// ── onActionTimeout ───────────────────────────────────────────────────────────

func TestOnActionTimeout_AutoFold(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)

	core.onActionTimeout(actionTimeoutFired{
		roundNumber: round.RoundNumber,
		roundID:     core.currentSession.RoundID(),
		playerName:  "Alice",
	})

	pkgs := drainN(t, outbound, 3) // 2× FOLD;Alice + 1× SCORE

	hasFold := firstWithPrefix(pkgs, "FOLD;Alice;TIMEOUT")
	if !hasFold {
		t.Error("expected FOLD broadcast on action timeout")
	}

	if !round.Table.Players[0].HasFolded {
		t.Error("Alice should be folded after timeout")
	}
}

func TestOnActionTimeout_StaleRound(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)

	core.onActionTimeout(actionTimeoutFired{
		roundNumber: 99,
		roundID:     core.currentSession.RoundID(),
		playerName:  "Alice",
	})

	_ = round // used above

	if len(outbound) != 0 {
		t.Error("expected no outbound messages for stale timeout event")
	}
}

func TestOnActionTimeout_WrongPlayer(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)

	core.onActionTimeout(actionTimeoutFired{
		roundNumber: round.RoundNumber,
		roundID:     core.currentSession.RoundID(),
		playerName:  "Bob", // it's Alice's turn
	})

	if len(outbound) != 0 {
		t.Error("expected no outbound messages when timeout is for the wrong player")
	}
}
