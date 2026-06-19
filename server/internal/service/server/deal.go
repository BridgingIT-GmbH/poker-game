package server

import (
	"fmt"
	"math/rand"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

const (
	defaultSmallBlind int64 = 10
	defaultBigBlind   int64 = 20
	defaultChips      int64 = 1000
	holeCardCount           = 2
	bigBlindOffset          = 2 // seats between dealer and big blind
)

// dealCards initialises the poker table for the current round, shuffles the
// deck, deals two hole cards to every joined player and sends each player
// their hand via a CARDS message. Afterwards the player who must act first
// receives a YOUR TURN prompt.
func (c *serverCore) dealCards() {
	round := c.currentRound
	if round == nil {
		return
	}

	names, pgPlayers := buildOrderedPlayers(round)
	table := buildTable(pgPlayers, round.RoundNumber)

	c.shuffleAndDeal(table, names, round)

	round.Table = table
	round.PlayerOrder = names

	c.promptFirstToAct(round)
}

func buildOrderedPlayers(round *Round) ([]string, []*model.Player) {
	names := make([]string, 0, len(round.JoinedPlayers))
	for name := range round.JoinedPlayers {
		names = append(names, name)
	}

	rand.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })

	pgPlayers := make([]*model.Player, len(names))
	for i, name := range names {
		p := model.NewPlayer(i, name, defaultChips)
		p.IsActive = true
		pgPlayers[i] = p
	}

	return names, pgPlayers
}

func buildTable(pgPlayers []*model.Player, roundNumber int) *model.Table {
	table := model.NewTable(pgPlayers, defaultSmallBlind, defaultBigBlind)
	n := len(pgPlayers)
	table.DealerPosition = (roundNumber - 1) % n
	table.SmallBlindPos = (table.DealerPosition + 1) % n
	table.BigBlindPos = (table.DealerPosition + bigBlindOffset) % n
	// Pre-flop: first to act is the player left of the big blind.
	table.CurrentToActPos = (table.BigBlindPos + 1) % n
	table.NumPlayersActive = n

	return table
}

func (c *serverCore) shuffleAndDeal(table *model.Table, names []string, round *Round) {
	table.Deck.Shuffle()

	for i, pgPlayer := range table.Players {
		cards := table.Deck.Deal(holeCardCount)
		pgPlayer.HoleCards = [2]model.Card{cards[0], cards[1]}

		serverPlayer := round.JoinedPlayers[names[i]]
		msg := fmt.Sprintf("%s;%s,%s\n", protocol.CmdCards, cards[0], cards[1])
		c.sendToPeer(msg, serverPlayer.Peer)
	}
}

func (c *serverCore) promptFirstToAct(round *Round) {
	firstToActName := round.PlayerOrder[round.Table.CurrentToActPos]
	firstToActPeer := round.JoinedPlayers[firstToActName].Peer
	msg := fmt.Sprintf("%s;%s\n", protocol.CmdYourTurn, round.Token)
	c.sendToPeer(msg, firstToActPeer)
}
