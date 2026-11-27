package game

import (
	"errors"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
)

// ShowdownResult holds the outcome of a finished hand.
type ShowdownResult struct {
	Winners []*model.Player
	Pot     int64
	// Survivors contains all players still active at showdown, with their hole cards.
	Survivors []*model.Player
	BestHands map[*model.Player]string
}

// SurvivorInfo describes a player who reached the showdown.
type SurvivorInfo struct {
	Name      string
	HoleCards [2]model.Card
}

var errNoPlayersLeft = errors.New("no active, non-folded players left")

// DetermineWinner evaluates the end-of-hand state and returns a ShowdownResult.
//
// If only one active player remains (all others folded) that player wins the
// entire pot without a card comparison.
//
// If multiple players remain the result includes all survivors so the caller
// can broadcast their hole cards. Full hand-rank evaluation is a future TODO:
// for now the first survivor in seat order is returned as the provisional winner.
func DetermineWinner(round *Round) (*ShowdownResult, error) {
	table := round.Table
	bestHandsMap := make(map[*model.Player]string)

	// Collect survivors (active and not folded).
	survivors := make([]*model.Player, 0)

	for _, player := range table.Players {
		if player != nil && player.IsActive && !player.HasFolded {
			survivors = append(survivors, player)
		}
	}

	// No survivors → error.
	if len(survivors) == 0 {
		return nil, errNoPlayersLeft
	}

	// If only one player remains, they win the pot without card comparison.
	if len(survivors) == 1 {
		return &ShowdownResult{
			Winners:   []*model.Player{survivors[0]},
			Pot:       table.Pot,
			Survivors: survivors,
			BestHands: bestHandsMap,
		}, nil
	}

	// Multiple survivors → perform full showdown evaluation.
	winners, bestHands, err := EvaluateShowdown(table)
	if err != nil {
		return nil, err
	}

	for player, handValue := range bestHands {
		switch handValue.Value.Category {
		case HighCard:
			bestHandsMap[player] = "HIGH_CARD"
		case OnePair:
			bestHandsMap[player] = "ONE_PAIR"
		case TwoPair:
			bestHandsMap[player] = "TWO_PAIR"
		case ThreeOfAKind:
			bestHandsMap[player] = "THREE_OF_A_KIND"
		case Straight:
			bestHandsMap[player] = "STRAIGHT"
		case Flush:
			bestHandsMap[player] = "FLUSH"
		case FullHouse:
			bestHandsMap[player] = "FULL_HOUSE"
		case FourOfAKind:
			bestHandsMap[player] = "FOUR_OF_A_KIND"
		case StraightFlush:
			bestHandsMap[player] = "STRAIGHT_FLUSH"
		}
	}

	return &ShowdownResult{
		Winners:   winners,
		Pot:       table.Pot,
		Survivors: survivors,
		BestHands: bestHandsMap,
	}, nil
}
