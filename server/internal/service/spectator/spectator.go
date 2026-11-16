package spectator

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

// playbackDelay is the pause between two replayed messages, so the game state
// fills up gradually instead of all at once.
const playbackDelay = 2 * time.Second

const (
	readHeaderTimeout = 10 * time.Second
	registerMessage   = "REGISTER_SPECTATOR;abc"
)

// Spectator registers at the game server via UDP, replays the received rounds
// with a delay and serves the game state to browsers via HTTP/SSE.
//
// The public API is: NewSpectator(*net.UDPConn, string) and Start().
type Spectator struct {
	conn     *net.UDPConn
	httpAddr string
	hub      *Hub
	queue    chan []string
}

func NewSpectator(conn *net.UDPConn, httpAddr string) *Spectator {
	return &Spectator{
		conn:     conn,
		httpAddr: httpAddr,
		hub:      NewHub(),
		// Capacity 1 is required by UdpMessageHandler.enqueueRound.
		queue: make(chan []string, 1),
	}
}

// Start registers as spectator, starts UDP reading and playback and blocks
// while serving HTTP.
func (s *Spectator) Start() error {
	templates, err := parseTemplates()
	if err != nil {
		return err
	}

	if _, err := s.conn.Write([]byte(registerMessage)); err != nil {
		return fmt.Errorf("sending register message: %w", err)
	}

	slog.Info("Sent message", "message", registerMessage)

	go NewUdpMessageHandler(s.conn, s.queue).ListenForMessages()
	go playback(s.queue, s.hub)

	mux := http.NewServeMux()
	mux.HandleFunc("/", makeIndexHandler(templates))
	mux.HandleFunc("/events", makeSSEHandler(s.hub, templates))

	server := &http.Server{
		Addr:              s.httpAddr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	slog.Info("HTTP server started", "address", s.httpAddr)

	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("HTTP server stopped: %w", err)
	}

	return nil
}

// playback replays buffered rounds message by message. After a message that
// actually changed the game state it waits playbackDelay before continuing, so
// the state is revealed step by step. There is no pause right before a TURN
// message, so an action or board and the following turn appear together.
func playback(queue <-chan []string, hub *Hub) {
	for round := range queue {
		for i, message := range round {
			changed := hub.Apply(message)
			nextIsTurn := i+1 < len(round) && strings.HasPrefix(round[i+1], "TURN;")

			if changed && !nextIsTurn {
				time.Sleep(playbackDelay)
			}
		}
	}
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	values := strings.Split(value, ",")
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}

	return values
}
