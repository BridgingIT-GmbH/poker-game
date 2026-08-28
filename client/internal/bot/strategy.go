package bot

// Strategy is the decision-making interface that participants implement.
// The bot calls Decide every time the server sends a YOUR TURN message.
type Strategy interface {
	Decide(state GameState) Action
}

