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

// FoldReason is a machine-readable code explaining why a player was
// automatically folded by the server. It is empty for voluntary folds and
// for valid non-fold actions.
type FoldReason string

const (
	FoldReasonNone              FoldReason = ""
	FoldReasonTimeout           FoldReason = "TIMEOUT"
	FoldReasonCannotCheck       FoldReason = "CANNOT_CHECK"
	FoldReasonCannotAffordCall  FoldReason = "CANNOT_AFFORD_CALL"
	FoldReasonInsufficientChips FoldReason = "INSUFFICIENT_CHIPS"
	FoldReasonRaiseLimit        FoldReason = "RAISE_LIMIT"
	FoldReasonInvalidAction     FoldReason = "INVALID_ACTION"
)

// foldReasonFromError maps an action validation error to its FoldReason.
func foldReasonFromError(err error) FoldReason {
	switch {
	case errors.Is(err, ErrCannotCheck):
		return FoldReasonCannotCheck
	case errors.Is(err, ErrCannotAffordCall):
		return FoldReasonCannotAffordCall
	case errors.Is(err, ErrInsufficientChips):
		return FoldReasonInsufficientChips
	case errors.Is(err, ErrRaiseLimitReached):
		return FoldReasonRaiseLimit
	default:
		return FoldReasonInvalidAction
	}
}

// ApplyPlayerAction dispatches the action command, enforces constraints and
// returns the action that was actually applied.
// Illegal or invalid actions result in an automatic fold: ActionFold is
// returned together with a non-empty FoldReason.
func ApplyPlayerAction(action Action, round *Round, playerName string) (Action, FoldReason) {
	table := round.Table
	player := table.Players[table.CurrentToActPos]

	var err error

	switch action {
	case ActionFold:
		ApplyFold(table, player)

		return ActionFold, FoldReasonNone
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
		reason := foldReasonFromError(err)
		slog.Warn("Invalid action, auto-folding", "player", playerName, "action", action,
			"reason", reason, "error", err)
		ApplyFold(table, player)

		return ActionFold, reason
	}

	return action, FoldReasonNone
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
