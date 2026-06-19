package server

import (
	"fmt"
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

func TestHandleLastPlayerStanding_ScoreFormat(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)
	round.Table.Pot = 300

	applyFold(round.Table, round.Table.Players[0]) // Alice folds → Bob wins

	core.handleLastPlayerStanding(round)

	pkgs := drainN(t, outbound, 2) // broadcast to both joined players

	expected := fmt.Sprintf("%s;Bob:300\n", protocol.CmdScore)
	for _, pkg := range pkgs {
		if string(pkg.Data) != expected {
			t.Errorf("expected %q, got %q", expected, string(pkg.Data))
		}
	}
}

func TestHandleShowdown_BroadcastsScore(t *testing.T) {
	t.Parallel()

	core, round, outbound := twoPlayerRoundInProgress(t)

	core.handleShowdown(round)

	pkgs := drainN(t, outbound, 2) // SCORE to both players

	for _, pkg := range pkgs {
		data := string(pkg.Data)
		if data[:len(string(protocol.CmdScore))] != string(protocol.CmdScore) {
			t.Errorf("expected message to start with SCORE, got %q", data)
		}
	}
}
