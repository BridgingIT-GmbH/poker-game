package spectator

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// LeaderboardEntry is a single row of the overall ranking.
type LeaderboardEntry struct {
	Rank   int
	Name   string
	Points int64
}

// Leaderboard accumulates the points of all SCORE messages and notifies
// connected SSE clients whenever the ranking changes. Unlike the Hub it is
// updated immediately when a SCORE message arrives, not during delayed playback.
type Leaderboard struct {
	mu      sync.RWMutex
	totals  map[string]int64
	clients map[chan []LeaderboardEntry]struct{}
}

func NewLeaderboard() *Leaderboard {
	return &Leaderboard{
		totals:  make(map[string]int64),
		clients: make(map[chan []LeaderboardEntry]struct{}),
	}
}

// Subscribe registers a new client and immediately delivers the current ranking.
func (l *Leaderboard) Subscribe() chan []LeaderboardEntry {
	ch := make(chan []LeaderboardEntry, subscriberBufferSize)

	l.mu.Lock()
	l.clients[ch] = struct{}{}
	snapshot := l.snapshotLocked()
	l.mu.Unlock()

	ch <- snapshot

	return ch
}

func (l *Leaderboard) Unsubscribe(ch chan []LeaderboardEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.clients[ch]; ok {
		delete(l.clients, ch)
		close(ch)
	}
}

// ApplyScore adds the points of a SCORE message ("SCORE;player:points[:cards];...")
// to the overall ranking. It returns true when the message was a SCORE message.
func (l *Leaderboard) ApplyScore(message string) bool {
	parts := strings.Split(strings.TrimSpace(message), ";")
	if len(parts) < minMessageParts || parts[0] != "SCORE" {
		return false
	}

	l.mu.Lock()

	for _, entry := range parts[1:] {
		fields := strings.Split(entry, ":")
		if len(fields) < minScoreEntryParts || fields[0] == "" {
			continue
		}

		points, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}

		l.totals[fields[0]] += points
	}

	snapshot := l.snapshotLocked()
	l.mu.Unlock()

	l.broadcast(snapshot)

	return true
}

// Snapshot returns the current ranking.
func (l *Leaderboard) Snapshot() []LeaderboardEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.snapshotLocked()
}

func (l *Leaderboard) broadcast(snapshot []LeaderboardEntry) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for ch := range l.clients {
		select {
		case ch <- snapshot:
		default:
			// Drop updates for slow clients; they get the next full ranking.
		}
	}
}

// snapshotLocked returns the ranking sorted by points (descending) and name.
// Players with equal points share the same rank (e.g. 1., 1., 3.).
func (l *Leaderboard) snapshotLocked() []LeaderboardEntry {
	entries := make([]LeaderboardEntry, 0, len(l.totals))
	for name, points := range l.totals {
		entries = append(entries, LeaderboardEntry{Name: name, Points: points})
	}

	slices.SortFunc(entries, func(a, b LeaderboardEntry) int {
		return cmp.Or(cmp.Compare(b.Points, a.Points), cmp.Compare(a.Name, b.Name))
	})

	for i := range entries {
		if i > 0 && entries[i].Points == entries[i-1].Points {
			entries[i].Rank = entries[i-1].Rank
		} else {
			entries[i].Rank = i + 1
		}
	}

	return entries
}
