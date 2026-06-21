package server

import (
	"fmt"
	"strings"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

// advanceTurn decides what happens after a player has acted:
//  1. If only one player remains → they win (last player standing).
//  2. If the current betting round is complete → advance to the next street.
//  3. Otherwise → prompt the next active player.
func (c *serverCore) advanceTurn(round *game.Round) {
	table := round.Table

	if table.CountActivePlayers() <= 1 {
		c.handleLastPlayerStanding(round)

		return
	}

	if game.IsBettingRoundOver(table) {
		c.advanceBettingRound(round)

		return
	}

	// Find next active, non-folded, non-all-in player.
	n := len(table.Players)
	table.CurrentToActPos = game.NextActivePlayerPos(table, (table.CurrentToActPos+1)%n)

	c.promptCurrentToAct(round)
}

// advanceBettingRound delegates street advancement to the game package
// (deals community cards, resets betting state) and broadcasts the new board
// or triggers the showdown.
func (c *serverCore) advanceBettingRound(round *game.Round) {
	reachedShowdown := game.NextStreet(round)

	if reachedShowdown {
		c.handleShowdown(round)

		return
	}

	c.broadcastBoard(round)
	c.promptCurrentToAct(round)
}

// broadcastBoard serialises the current community cards and broadcasts a
// BOARD message to all players in the round.
func (c *serverCore) broadcastBoard(round *game.Round) {
	cards := make([]string, len(round.Table.CommunityCards))
	for i, card := range round.Table.CommunityCards {
		cards[i] = card.String()
	}

	msg := fmt.Sprintf("%s;%s\n", protocol.CmdBoard, strings.Join(cards, ","))
	c.broadcastToPlayers(msg, round.JoinedPlayers)
}
