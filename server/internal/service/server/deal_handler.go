package server

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

const (
	defaultSmallBlind int64 = 10
	defaultBigBlind   int64 = 20
	defaultChips      int64 = 1000
	holeCardCount           = 2
)

// dealCards initialises the poker table for the current round:
// it calls the game package to build the player order, table and collect blinds,
// shuffles the deck, deals hole cards to each player (sent via CARDS message)
// and prompts the first player to act.
func (c *serverCore) dealCards(names []string) {
	round := c.currentRound
	if round == nil {
		return
	}

	players := game.BuildOrderedPlayers(names, defaultChips)
	table := game.BuildTable(players, defaultSmallBlind, defaultBigBlind)
	game.CollectBlinds(table)

	c.shuffleAndDeal(table, names)

	round.Table = table
	round.PlayerOrder = names

	// Sync JoinedPlayers values with the real model.Client objects from the table.
	for i, name := range names {
		round.JoinedPlayers[name] = table.Players[i]
	}

	potMsg := fmt.Sprintf("%s;%d\n", protocol.CmdPot, table.Pot)
	c.sendToSpectators(potMsg)

	c.promptCurrentToAct(round)
}

// shuffleAndDeal shuffles the deck and deals two hole cards to each player,
// sending each player their hand via a CARDS protocol message.
func (c *serverCore) shuffleAndDeal(table *model.Table, names []string) {
	table.Deck.Shuffle()

	for i, pgPlayer := range table.Players {
		cards := table.Deck.Deal(holeCardCount)
		pgPlayer.HoleCards = [2]model.Card{cards[0], cards[1]}

		serverPlayer, ok := c.players[names[i]]
		if !ok {
			slog.Error("Cannot deal cards: player not found in registry", "player", names[i])

			continue
		}

		msg := fmt.Sprintf("%s;%s,%s\n", protocol.CmdCards, cards[0], cards[1])
		c.sendToPeer(msg, serverPlayer.Peer)
	}
}

// promptCurrentToAct generates a one-time action token for the active player,
// sends a YOUR_TURN message, tells spectators whose turn it is and starts the
// action timer.
func (c *serverCore) promptCurrentToAct(round *game.Round) {
	toActName := round.PlayerOrder[round.Table.CurrentToActPos]

	serverPlayer, ok := c.players[toActName]
	if !ok {
		slog.Error("Cannot prompt player: not found in registry", "player", toActName)

		return
	}

	token, err := generateToken()
	if err != nil {
		slog.Error("Failed to generate YOUR TURN token", "player", toActName, "error", err)

		return
	}

	c.currentSession.PendingTokens[toActName] = token
	msg := fmt.Sprintf("%s;%s\n", protocol.CmdYourTurn, token)
	c.sendToPeer(msg, serverPlayer.Peer)

	c.sendToSpectators(fmt.Sprintf("%s;%s\n", protocol.CmdTurn, toActName))

	// Cancel the previous action timer and start a fresh one.
	if c.actionTimer != nil {
		c.actionTimer.Stop()
	}

	c.actionTimer = time.AfterFunc(actionTimeout, func() {
		c.events <- actionTimeoutFired{
			roundNumber: round.RoundNumber,
			roundID:     c.currentSession.RoundID(),
			playerName:  toActName,
		}
	})
}
