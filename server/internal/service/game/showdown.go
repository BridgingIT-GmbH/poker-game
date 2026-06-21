package game

import "github.com/BridgingIT-GmbH/poker-game/server/internal/model"

// ShowdownResult holds the outcome of a finished hand.
type ShowdownResult struct {
	WinnerName     string
	Pot            int64
	IsLastStanding bool // true when all other players folded (no hand comparison needed)
	// Survivors contains all players still active at showdown, with their hole cards.
	// Only populated when IsLastStanding is false.
	Survivors []SurvivorInfo
}

// SurvivorInfo describes a player who reached the showdown.
type SurvivorInfo struct {
	Name      string
	HoleCards [2]model.Card
}

// DetermineWinner evaluates the end-of-hand state and returns a ShowdownResult.
//
// If only one active player remains (all others folded) that player wins the
// entire pot without a card comparison.
//
// If multiple players remain the result includes all survivors so the caller
// can broadcast their hole cards. Full hand-rank evaluation is a future TODO:
// for now the first survivor in seat order is returned as the provisional winner.
func DetermineWinner(round *Round) ShowdownResult {
	table := round.Table

	var survivors []SurvivorInfo

	for i, name := range round.PlayerOrder {
		p := table.Players[i]
		if p.IsActive && !p.HasFolded {
			survivors = append(survivors, SurvivorInfo{
				Name:      name,
				HoleCards: p.HoleCards,
			})
		}
	}

	// Single survivor – wins by last-player-standing (all others folded).
	if len(survivors) == 1 {
		return ShowdownResult{
			WinnerName:     survivors[0].Name,
			Pot:            table.Pot,
			IsLastStanding: true,
		}
	}

	// Multiple survivors – full showdown.
	// TODO: replace provisional first-seat winner with proper hand evaluation
	// via model.EvaluateHand once that is implemented.
	winner := survivors[0].Name

	return ShowdownResult{
		WinnerName:     winner,
		Pot:            table.Pot,
		IsLastStanding: false,
		Survivors:      survivors,
	}
}
