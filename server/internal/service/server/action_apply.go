package server

import (
	"errors"
	"log/slog"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

var (
	errCannotCheck       = errors.New("cannot check: there is an outstanding bet to call")
	errInsufficientChips = errors.New("insufficient chips to raise")
	errCannotAffordCall  = errors.New("cannot call: insufficient chips (all-in not supported)")
	errRaiseLimitReached = errors.New("raise limit reached: only 1 raise per player per betting round")
	errUnexpectedCommand = errors.New("unexpected command in action handler")
)

const (
	maxRaisesPerStreet = 1 // one raise per player per betting round (pre-flop/flop/turn/river)
)

// applyPlayerAction dispatches the action command, enforces constraints and
// returns the command that should be broadcast to all players.
// Illegal or invalid actions result in an automatic fold (CmdFold is returned).
func applyPlayerAction(cmd protocol.CommandPlayer, round *Round, playerName string) protocol.CommandPlayer {
	table := round.Table
	player := table.Players[table.CurrentToActPos]

	var err error

	switch cmd {
	case protocol.CmdFold:
		applyFold(table, player)

		return protocol.CmdFold
	case protocol.CmdCheck:
		err = applyCheck(table, player)
	case protocol.CmdCall:
		err = applyCall(table, player)
	case protocol.CmdRaise:
		if round.RaisesThisStreet[playerName] >= maxRaisesPerStreet {
			err = errRaiseLimitReached
		} else {
			err = applyRaise(table, player)
			if err == nil {
				round.RaisesThisStreet[playerName]++
			}
		}
	case protocol.CmdRegister, protocol.CmdJoin:
		// These commands are handled upstream in handleMessage and must never
		// reach the action handler; treat as an unexpected command.
		err = errUnexpectedCommand
	}

	if err != nil {
		slog.Warn("Invalid action, auto-folding", "player", playerName, "cmd", cmd, "error", err)
		applyFold(table, player)

		return protocol.CmdFold
	}

	return cmd
}

// ── individual action helpers ─────────────────────────────────────────────────

func applyFold(table *model.Table, player *model.Player) {
	player.HasFolded = true
	player.IsActive = false
	player.HasActed = true
	table.NumPlayersActive--
}

func applyCheck(table *model.Table, player *model.Player) error {
	if table.CurrentBet > player.CurrentBet {
		return errCannotCheck
	}

	player.HasActed = true

	return nil
}

// applyCall matches the current bet.
// All-in is not supported: if the player cannot afford the full call, an error
// is returned and the caller will auto-fold.
func applyCall(table *model.Table, player *model.Player) error {
	amount := table.CurrentBet - player.CurrentBet
	if amount > player.Chips {
		return errCannotAffordCall
	}

	player.Chips -= amount
	player.CurrentBet += amount
	player.TotalBetThisHand += amount
	table.Pot += amount
	player.HasActed = true

	return nil
}

// applyRaise raises the current bet by exactly one big blind.
// All-in is not supported: if the player cannot afford the raise, an error is returned.
// All other active players must act again after a raise.
func applyRaise(table *model.Table, player *model.Player) error {
	newBet := table.CurrentBet + table.BigBlindAmount
	amount := newBet - player.CurrentBet

	if amount > player.Chips {
		return errInsufficientChips
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
