package game

import (
	"math/rand"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
)

const (
	bigBlindOffset = 2 // seats between dealer and big blind
)

// BuildOrderedPlayers creates a randomly shuffled seat order from the given
// player names and initialises a model.Player for each seat with initialChips.
// Returns the ordered names slice and the corresponding model.Player slice.
func BuildOrderedPlayers(playerNames []string, initialChips int64) ([]string, []*model.Player) {
	names := make([]string, len(playerNames))
	copy(names, playerNames)

	rand.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })

	players := make([]*model.Player, len(names))
	for i, name := range names {
		p := model.NewPlayer(i, name, initialChips)
		p.IsActive = true
		players[i] = p
	}

	return names, players
}

// BuildTable creates a model.Table with dealer, blind and first-to-act
// positions correctly set for the given round number.
// smallBlind and bigBlind are the blind amounts for this hand.
func BuildTable(players []*model.Player, roundNumber int, smallBlind, bigBlind int64) *model.Table {
	table := model.NewTable(players, smallBlind, bigBlind)
	n := len(players)
	table.DealerPosition = (roundNumber - 1) % n
	table.SmallBlindPos = (table.DealerPosition + 1) % n
	table.BigBlindPos = (table.DealerPosition + bigBlindOffset) % n
	// Pre-flop: first to act is the player left of the big blind.
	table.CurrentToActPos = (table.BigBlindPos + 1) % n
	table.NumPlayersActive = n

	return table
}

// CollectBlinds deducts the small and big blind from the respective players
// and sets the initial pot and current bet on the table.
func CollectBlinds(table *model.Table) {
	sb := table.Players[table.SmallBlindPos]
	bb := table.Players[table.BigBlindPos]

	sbPay := min(table.SmallBlindAmount, sb.Chips)
	bbPay := min(table.BigBlindAmount, bb.Chips)

	sb.Chips -= sbPay
	sb.CurrentBet = sbPay
	sb.TotalBetThisHand += sbPay

	bb.Chips -= bbPay
	bb.CurrentBet = bbPay
	bb.TotalBetThisHand += bbPay

	table.Pot = sbPay + bbPay
	table.CurrentBet = bbPay
}
