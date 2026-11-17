package spectator

import (
	"maps"
	"strings"
	"sync"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/service/game"
)

// PlayerView is a player as shown in the player list, including the table
// positions (D, SB, BB) they hold in the current round.
type PlayerView struct {
	Name  string
	Roles []string
	// Cards holds the player's hole cards once they are revealed at showdown.
	Cards []string
}

type GameState struct {
	Players []PlayerView
	Board   []string
	// Actions maps a player name to their most recent action.
	Actions map[string]string
	// ToAct is the name of the player whose turn it is, or "" if nobody is to act.
	ToAct string
	Score string
}

// HiddenBoardCards returns one entry per board card that has not been revealed
// yet. Outside of a round (no players) there are no hidden cards.
func (s GameState) HiddenBoardCards() []struct{} {
	if len(s.Players) == 0 || len(s.Board) >= boardSize {
		return nil
	}

	return make([]struct{}, boardSize-len(s.Board))
}

// Hub keeps the current game state and notifies connected SSE clients whenever
// the state changes.
const (
	boardSize            = 5
	subscriberBufferSize = 8
	minMessageParts      = 2
	roundStartedParts    = 3
	minBlindPlayers      = 2
	scoreEntryParts      = 3
	holeCardCount        = 2
)

type Hub struct {
	mu      sync.RWMutex
	state   GameState
	clients map[chan GameState]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[chan GameState]struct{})}
}

// Subscribe registers a new client and immediately delivers the current state.
func (h *Hub) Subscribe() chan GameState {
	ch := make(chan GameState, subscriberBufferSize)

	h.mu.Lock()
	h.clients[ch] = struct{}{}
	snapshot := h.snapshotLocked()
	h.mu.Unlock()

	ch <- snapshot

	return ch
}

func (h *Hub) Unsubscribe(ch chan GameState) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[ch]; ok {
		delete(h.clients, ch)
		close(ch)
	}
}

// Apply updates the game state from a single protocol message and broadcasts
// the new state to all subscribers. It returns true when the message changed
// the state (i.e. it belongs to a round: ROUND STARTED .. SCORE).
func (h *Hub) Apply(message string) bool {
	parts := strings.Split(strings.TrimSpace(message), ";")
	if len(parts) < minMessageParts {
		return false
	}

	h.mu.Lock()

	switch parts[0] {
	case "ROUND STARTED":
		if len(parts) < roundStartedParts {
			h.mu.Unlock()

			return false
		}
		// A new round resets the whole view.
		h.state = GameState{Players: buildPlayerViews(splitList(parts[2]))}
	case "BOARD":
		h.state.Board = splitList(parts[1])
		// Revealing board cards starts a new betting round, so previous
		// actions are no longer relevant, except folds, which last the
		// whole round.
		maps.DeleteFunc(h.state.Actions, func(_, action string) bool {
			return action != "FOLD"
		})
	case "FOLD", "CHECK", "CALL", "RAISE":
		if h.state.Actions == nil {
			h.state.Actions = make(map[string]string)
		}

		h.state.Actions[parts[1]] = parts[0]
		// The player has acted; the server announces the next one via TURN.
		h.state.ToAct = ""
	case "TURN":
		h.state.ToAct = parts[1]
	case "SCORE":
		h.state.Score = strings.Join(parts[1:], " ")
		h.state.ToAct = ""
		h.revealHoleCards(parts[1:])
	default:
		h.mu.Unlock()

		return false
	}

	snapshot := h.snapshotLocked()
	h.mu.Unlock()

	h.broadcast(snapshot)

	return true
}

func (h *Hub) broadcast(snapshot GameState) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.clients {
		select {
		case ch <- snapshot:
		default:
			// Drop updates for slow clients to keep playback responsive.
		}
	}
}

// revealHoleCards stores the hole cards from SCORE entries of the form
// "player:score:card1,card2". Entries without cards (e.g. when everybody else
// folded) leave the player's cards hidden.
func (h *Hub) revealHoleCards(entries []string) {
	for _, entry := range entries {
		fields := strings.Split(entry, ":")
		if len(fields) < scoreEntryParts {
			continue
		}

		cards := splitList(fields[2])
		if len(cards) != holeCardCount {
			continue
		}

		for i := range h.state.Players {
			if h.state.Players[i].Name == fields[0] {
				h.state.Players[i].Cards = cards
			}
		}
	}
}

func (h *Hub) snapshotLocked() GameState {
	players := make([]PlayerView, len(h.state.Players))
	for i, p := range h.state.Players {
		players[i] = PlayerView{
			Name:  p.Name,
			Roles: append([]string(nil), p.Roles...),
			Cards: append([]string(nil), p.Cards...),
		}
	}

	return GameState{
		Players: players,
		Board:   append([]string(nil), h.state.Board...),
		Actions: maps.Clone(h.state.Actions),
		ToAct:   h.state.ToAct,
		Score:   h.state.Score,
	}
}

// buildPlayerViews assigns dealer and blind positions using the same rules the
// server uses when building the table from the ROUND STARTED player order.
func buildPlayerViews(names []string) []PlayerView {
	views := make([]PlayerView, len(names))
	for i, name := range names {
		views[i].Name = name
	}

	if len(names) < minBlindPlayers {
		return views
	}

	table := game.BuildTable(game.BuildOrderedPlayers(names, 0), 0, 0)
	views[table.DealerPosition].Roles = append(views[table.DealerPosition].Roles, "D")
	views[table.SmallBlindPos].Roles = append(views[table.SmallBlindPos].Roles, "SB")
	views[table.BigBlindPos].Roles = append(views[table.BigBlindPos].Roles, "BB")

	return views
}
