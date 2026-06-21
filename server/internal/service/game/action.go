package game

import (
	"errors"
	"log/slog"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
)

// Action represents a player's poker action.
// Defined here so the game package has no dependency on pkg/protocol.
type Action string

const (
	ActionFold  Action = "FOLD"
	ActionCheck Action = "CHECK"
	ActionCall  Action = "CALL"
	ActionRaise Action = "RAISE"
)

var (
	ErrCannotCheck       = errors.New("cannot check: there is an outstanding bet to call")
	ErrInsufficientChips = errors.New("insufficient chips to raise")
	ErrCannotAffordCall  = errors.New("cannot call: insufficient chips (all-in not supported)")
	ErrRaiseLimitReached = errors.New("raise limit reached: only 1 raise per player per betting round")
	ErrUnexpectedAction  = errors.New("unexpected action")
)

const (
	maxRaisesPerStreet = 1 // one raise per player per betting round (pre-flop/flop/turn/river)
)

// ApplyPlayerAction dispatches the action command, enforces constraints and
// returns the action that was actually applied.
// Illegal or invalid actions result in an automatic fold (ActionFold is returned).
func ApplyPlayerAction(action Action, round *Round, playerName string) Action {
	table := round.Table
	player := table.Players[table.CurrentToActPos]

	var err error

	switch action {
	case ActionFold:
		ApplyFold(table, player)

		return ActionFold
	case ActionCheck:
		err = ApplyCheck(table, player)
	case ActionCall:
		err = ApplyCall(table, player)
	case ActionRaise:
		if round.RaisesThisStreet[playerName] >= maxRaisesPerStreet {
			err = ErrRaiseLimitReached
		} else {
			err = ApplyRaise(table, player)
			if err == nil {
				round.RaisesThisStreet[playerName]++
			}
		}
	default:
		err = ErrUnexpectedAction
	}

	if err != nil {
		slog.Warn("Invalid action, auto-folding", "player", playerName, "action", action, "error", err)
		ApplyFold(table, player)

		return ActionFold
	}

	return action
}

// ── individual action helpers ─────────────────────────────────────────────────

// ApplyFold marks the player as folded and decrements the active player count.
func ApplyFold(table *model.Table, player *model.Player) {
	player.HasFolded = true
	player.IsActive = false
	player.HasActed = true
	table.NumPlayersActive--
}

// ApplyCheck verifies that no outstanding bet must be called.
func ApplyCheck(table *model.Table, player *model.Player) error {
	if table.CurrentBet > player.CurrentBet {
		return ErrCannotCheck
	}

	player.HasActed = true

	return nil
}

// ApplyCall matches the current bet.
// All-in is not supported: if the player cannot afford the full call, an error
// is returned and the caller will auto-fold.
func ApplyCall(table *model.Table, player *model.Player) error {
	amount := table.CurrentBet - player.CurrentBet
	if amount > player.Chips {
		return ErrCannotAffordCall
	}

	player.Chips -= amount
	player.CurrentBet += amount
	player.TotalBetThisHand += amount
	table.Pot += amount
	player.HasActed = true

	return nil
}

// ApplyRaise raises the current bet by exactly one big blind.
// All-in is not supported: if the player cannot afford the raise, an error is returned.
// All other active players must act again after a raise.
func ApplyRaise(table *model.Table, player *model.Player) error {
	newBet := table.CurrentBet + table.BigBlindAmount
	amount := newBet - player.CurrentBet

	if amount > player.Chips {
		return ErrInsufficientChips
	}

	player.Chips -= amount
	player.TotalBetThisHand += amount
	table.Pot += amount
	player.CurrentBet = newBet
	table.CurrentBet = newBet
	player.HasActed = true

	// Everyone else must respond to the raise.
	for _, p := range table.Players {
		if p != player && p.IsActive && !p.HasFolded && !p.IsAllIn {
			p.HasActed = false
		}
	}

	return nil
}
