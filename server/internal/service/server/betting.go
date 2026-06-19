package server

import (
	"fmt"
	"strings"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

const (
	flopCardCount      = 3
	turnRiverCardCount = 1
)

// advanceTurn decides what happens after a player has acted:
//  1. If only one player remains → they win.
//  2. If the current betting round is complete → advance to the next street.
//  3. Otherwise → prompt the next active player.
func (c *serverCore) advanceTurn(round *Round) {
	table := round.Table

	if table.CountActivePlayers() <= 1 {
		c.handleLastPlayerStanding(round)

		return
	}

	if isBettingRoundOver(table) {
		c.advanceBettingRound(round)

		return
	}

	// Find next active, non-folded, non-all-in player.
	n := len(table.Players)
	pos := (table.CurrentToActPos + 1) % n

	for range n {
		p := table.Players[pos]
		if p.IsActive && !p.HasFolded && !p.IsAllIn {
			break
		}

		pos = (pos + 1) % n
	}

	table.CurrentToActPos = pos

	c.promptCurrentToAct(round)
}

// isBettingRoundOver returns true when every active, non-folded, non-all-in
// player has acted and matched the current bet.
func isBettingRoundOver(table *model.Table) bool {
	for _, p := range table.Players {
		if !p.IsActive || p.HasFolded || p.IsAllIn {
			continue
		}

		if !p.HasActed || p.CurrentBet < table.CurrentBet {
			return false
		}
	}

	return true
}

// advanceBettingRound deals the next community cards (Flop/Turn/River) or
// triggers the showdown after the River.
func (c *serverCore) advanceBettingRound(round *Round) {
	table := round.Table
	table.ResetForNewBettingRound()

	// Reset per-street raise counters for the new betting round.
	round.RaisesThisStreet = make(map[string]int)

	switch table.BettingRound {
	case model.PreFlop:
		table.CommunityCards = append(table.CommunityCards, table.Deck.Deal(flopCardCount)...)
		table.BettingRound = model.Flop
	case model.Flop:
		table.CommunityCards = append(table.CommunityCards, table.Deck.Deal(turnRiverCardCount)...)
		table.BettingRound = model.Turn
	case model.Turn:
		table.CommunityCards = append(table.CommunityCards, table.Deck.Deal(turnRiverCardCount)...)
		table.BettingRound = model.River
	case model.River:
		table.BettingRound = model.Showdown

		c.handleShowdown(round)

		return
	case model.Showdown:
		// Already at showdown; nothing to advance.
		return
	default:
		return
	}

	c.broadcastBoard(round)

	// Post-flop: first to act starts left of the dealer.
	n := len(table.Players)
	pos := (table.DealerPosition + 1) % n

	for range n {
		p := table.Players[pos]
		if p.IsActive && !p.HasFolded && !p.IsAllIn {
			break
		}

		pos = (pos + 1) % n
	}

	table.CurrentToActPos = pos

	c.promptCurrentToAct(round)
}

func (c *serverCore) broadcastBoard(round *Round) {
	cards := make([]string, len(round.Table.CommunityCards))
	for i, card := range round.Table.CommunityCards {
		cards[i] = card.String()
	}

	msg := fmt.Sprintf("%s;%s\n", protocol.CmdBoard, strings.Join(cards, ","))
	c.broadcastToPlayers(msg, round.JoinedPlayers)
}
