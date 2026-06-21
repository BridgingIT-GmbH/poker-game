package server

import (
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

func TestAdvanceBettingRound_PreFlopToFlop(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 32)
	core := newServerCore(outbound)

	core.players["Alice"] = &Player{Name: "Alice", Peer: "peer-alice"}
	core.players["Bob"] = &Player{Name: "Bob", Peer: "peer-bob"}

	round := game.NewRound(1)
	session, _ := NewRoundSession()

	pgAlice := model.NewPlayer(0, "Alice", defaultChips)
	pgBob := model.NewPlayer(1, "Bob", defaultChips)
	pgAlice.IsActive, pgAlice.HasActed, pgAlice.CurrentBet = true, true, defaultBigBlind
	pgBob.IsActive, pgBob.HasActed, pgBob.CurrentBet = true, true, defaultBigBlind

	table := model.NewTable([]*model.Player{pgAlice, pgBob}, defaultSmallBlind, defaultBigBlind)
	table.DealerPosition = 0
	table.CurrentToActPos = 0
	table.CurrentBet = defaultBigBlind
	table.BettingRound = model.PreFlop
	table.NumPlayersActive = 2
	table.Deck.Shuffle()

	round.Table = table
	round.PlayerOrder = []string{"Alice", "Bob"}
	round.JoinedPlayers["Alice"] = pgAlice
	round.JoinedPlayers["Bob"] = pgBob

	core.currentRound = round
	core.currentSession = session
	core.roundState = RoundStateInProgress

	core.advanceBettingRound(round)

	pkgs := drainN(t, outbound, 3) // 2× BOARD + 1× YOUR TURN

	hasBoard := firstWithPrefix(pkgs, string(protocol.CmdBoard))
	if !hasBoard {
		t.Error("expected BOARD message after pre-flop")
	}

	if table.BettingRound != model.Flop {
		t.Errorf("expected Flop, got %v", table.BettingRound)
	}

	if len(table.CommunityCards) != 3 {
		t.Errorf("expected 3 community cards, got %d", len(table.CommunityCards))
	}

	hasYourTurn := firstWithPrefix(pkgs, string(protocol.CmdYourTurn))
	if !hasYourTurn {
		t.Error("expected YOUR TURN after advancing to flop")
	}
}

func TestAdvanceBettingRound_ResetsRaiseCounters(t *testing.T) {
	t.Parallel()

	outbound := make(chan transport.OutboundPackage, 32)
	core := newServerCore(outbound)

	core.players["Alice"] = &Player{Name: "Alice", Peer: "peer-alice"}
	core.players["Bob"] = &Player{Name: "Bob", Peer: "peer-bob"}

	round := game.NewRound(1)
	session, _ := NewRoundSession()
	round.RaisesThisStreet["Alice"] = 1
	round.RaisesThisStreet["Bob"] = 1

	pgAlice := model.NewPlayer(0, "Alice", defaultChips)
	pgBob := model.NewPlayer(1, "Bob", defaultChips)
	pgAlice.IsActive, pgAlice.HasActed, pgAlice.CurrentBet = true, true, defaultBigBlind
	pgBob.IsActive, pgBob.HasActed, pgBob.CurrentBet = true, true, defaultBigBlind

	table := model.NewTable([]*model.Player{pgAlice, pgBob}, defaultSmallBlind, defaultBigBlind)
	table.DealerPosition = 0
	table.CurrentToActPos = 0
	table.CurrentBet = defaultBigBlind
	table.BettingRound = model.PreFlop
	table.NumPlayersActive = 2
	table.Deck.Shuffle()

	round.Table = table
	round.PlayerOrder = []string{"Alice", "Bob"}
	round.JoinedPlayers["Alice"] = pgAlice
	round.JoinedPlayers["Bob"] = pgBob

	core.currentRound = round
	core.currentSession = session
	core.roundState = RoundStateInProgress

	core.advanceBettingRound(round)
	drainN(t, outbound, 3)

	if round.RaisesThisStreet["Alice"] != 0 || round.RaisesThisStreet["Bob"] != 0 {
		t.Error("raise counters should be reset when betting round advances")
	}
}
