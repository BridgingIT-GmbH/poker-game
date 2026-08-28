package bot

// GameState holds all observable information available to a player strategy.
// It is updated automatically by the bot as messages arrive from the server.
type GameState struct {
	// HoleCards are the bot's two private cards, e.g. ["As", "Kh"].
	// They are set once per round when the CARDS message arrives.
	HoleCards []string

	// Board contains the community cards dealt so far (0–5 cards).
	// It grows with each BOARD message (flop, turn, river).
	Board []string

	// Players holds the names of all players who joined this round.
	// Set when ROUND STARTED is received.
	Players []string

	// FoldedPlayers tracks which players have folded this round.
	// Updated as FOLD broadcasts arrive from the server.
	FoldedPlayers map[string]bool

	// LastScore is the result string of the most recently completed round.
	// It can be used to understand which hand won and why.
	LastScore string
}

// ActivePlayers returns the names of players who have not yet folded.
func (s GameState) ActivePlayers() []string {
	active := make([]string, 0, len(s.Players))

	for _, name := range s.Players {
		if !s.FoldedPlayers[name] {
			active = append(active, name)
		}
	}

	return active
}

// Action represents a player decision returned by a Strategy.
type Action string

const (
	ActionCall  Action = "CALL"
	ActionCheck Action = "CHECK"
	ActionFold  Action = "FOLD"
	ActionRaise Action = "RAISE"
)
