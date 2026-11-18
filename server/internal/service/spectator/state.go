package spectator

import (
	"maps"
	"slices"
	"strconv"
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
	Cards      []string
	LastAction string
	Winner     bool
}

type GameState struct {
	Round   string
	Players []PlayerView
	Board   []string
	ToAct   string
	Pot     int64
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
	minScoreEntryParts   = 2
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
		h.state = GameState{
			Round:   parts[1],
			Players: buildPlayerViews(splitList(parts[2])),
		}
	case "BOARD":
		h.state.Board = splitList(parts[1])
		// Revealing board cards starts a new betting round, so previous
		// actions are no longer relevant, except folds, which last the
		// whole round.
		for i := range h.state.Players {
			if h.state.Players[i].LastAction != "FOLD" {
				h.state.Players[i].LastAction = ""
			}
		}
	case "FOLD", "CHECK", "CALL", "RAISE":
		if player := h.findPlayer(parts[1]); player != nil {
			player.LastAction = parts[0]
		}
		// The player has acted; the server announces the next one via TURN.
		h.state.ToAct = ""
	case "TURN":
		h.state.ToAct = parts[1]
	case "POT":
		pot, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			h.mu.Unlock()

			return false
		}

		h.state.Pot = pot
	case "SCORE":
		h.state.ToAct = ""
		// At the end of the round only the winners are labelled.
		for i := range h.state.Players {
			h.state.Players[i].LastAction = ""
		}

		h.applyScore(parts[1:])
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

// applyScore processes SCORE entries of the form "player:score:card1,card2"
// (showdown) or "player:score" (everybody else folded). All players with the
// highest score are marked as winners; revealed hole cards are stored.
func (h *Hub) applyScore(entries []string) {
	scores := make(map[*PlayerView]int64, len(entries))

	for _, entry := range entries {
		fields := strings.Split(entry, ":")
		if len(fields) < minScoreEntryParts {
			continue
		}

		player := h.findPlayer(fields[0])
		if player == nil {
			continue
		}

		if score, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
			scores[player] = score
		}

		if len(fields) < scoreEntryParts {
			continue
		}

		if cards := splitList(fields[2]); len(cards) == holeCardCount {
			player.Cards = cards
		}
	}

	if len(scores) == 0 {
		return
	}

	best := slices.Max(slices.Collect(maps.Values(scores)))
	for player, score := range scores {
		player.Winner = score == best
	}
}

func (h *Hub) findPlayer(name string) *PlayerView {
	for i := range h.state.Players {
		if h.state.Players[i].Name == name {
			return &h.state.Players[i]
		}
	}

	return nil
}

func (h *Hub) snapshotLocked() GameState {
	players := make([]PlayerView, len(h.state.Players))
	for i, p := range h.state.Players {
		players[i] = PlayerView{
			Name:       p.Name,
			Roles:      append([]string(nil), p.Roles...),
			Cards:      append([]string(nil), p.Cards...),
			LastAction: p.LastAction,
			Winner:     p.Winner,
		}
	}

	return GameState{
		Round:   h.state.Round,
		Players: players,
		Board:   append([]string(nil), h.state.Board...),
		ToAct:   h.state.ToAct,
		Pot:     h.state.Pot,
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
