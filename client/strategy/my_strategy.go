package strategy

import "github.com/BridgingIT-GmbH/poker-game/client/internal/bot"

// MyStrategy is your poker bot's brain.
//
// The bot will call Decide() every time it is your turn to act.
// You have access to:
//
//	state.HoleCards        – your two private cards, e.g. ["As", "Kh"]
//	state.Board            – community cards so far, e.g. ["Td", "7c", "2h"]
//	state.Players          – all players who joined this round
//	state.FoldedPlayers    – map of players who have folded
//	state.ActivePlayers()  – Players minus FoldedPlayers
//	state.LastScore        – result string of the previous round
//
// Return one of:
//
//	bot.ActionCall   – match the current bet
//	bot.ActionCheck  – pass (only valid when no bet is due)
//	bot.ActionFold   – discard your hand
//	bot.ActionRaise  – increase the bet
type MyStrategy struct{}

// Decide returns the action your bot takes on its turn.
//
// TODO: Replace the body below with your own strategy logic.
//
// Ideas to get started:
//   - Fold if you have no face cards
//   - Raise if you have a pair in your hole cards
//   - Check the board for potential flushes or straights
func (s MyStrategy) Decide(state bot.GameState) bot.Action {
	return bot.ActionCall
}
