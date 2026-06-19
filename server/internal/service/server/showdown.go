package server

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

// handleShowdown broadcasts all surviving players' hole cards.
// Winner determination via hand evaluation is left as a future TODO.
func (c *serverCore) handleShowdown(round *Round) {
	table := round.Table
	parts := make([]string, 0, len(round.PlayerOrder))

	for i, name := range round.PlayerOrder {
		p := table.Players[i]
		if p.IsActive && !p.HasFolded {
			parts = append(parts, fmt.Sprintf("%s:%s,%s", name, p.HoleCards[0], p.HoleCards[1]))
		}
	}

	msg := fmt.Sprintf("%s;%s\n", protocol.CmdScore, strings.Join(parts, ";"))
	c.broadcastToPlayers(msg, round.JoinedPlayers)
	slog.Info("Showdown", "round", round.RoundNumber, "pot", table.Pot)

	c.endRound()
}

// handleLastPlayerStanding is called when all but one player has folded.
func (c *serverCore) handleLastPlayerStanding(round *Round) {
	table := round.Table

	var winnerName string

	for i, name := range round.PlayerOrder {
		p := table.Players[i]
		if p.IsActive && !p.HasFolded {
			winnerName = name

			break
		}
	}

	msg := fmt.Sprintf("%s;%s:%d\n", protocol.CmdScore, winnerName, table.Pot)
	c.broadcastToPlayers(msg, round.JoinedPlayers)
	slog.Info("Last player standing wins", "winner", winnerName, "pot", table.Pot)

	c.endRound()
}

func (c *serverCore) endRound() {
	if c.actionTimer != nil {
		c.actionTimer.Stop()
		c.actionTimer = nil
	}

	c.roundState = RoundStateIdle
	c.currentRound = nil
	c.tryStartRound()
}
