package spectator

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

//go:embed web/*.html
var webFS embed.FS

const (
	indexTemplate       = "index.html"
	gameStateTemplate   = "game_state.html"
	leaderboardTemplate = "leaderboard.html"
	gameStateEvent      = "game-state"
	leaderboardEvent    = "leaderboard"
)

const keepAliveInterval = 15 * time.Second

func parseTemplates() (*template.Template, error) {
	templates, err := template.ParseFS(webFS, "web/*.html")
	if err != nil {
		return nil, fmt.Errorf("parsing web templates: %w", err)
	}

	return templates, nil
}

func makeIndexHandler(templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if err := templates.ExecuteTemplate(w, indexTemplate, nil); err != nil {
			slog.Error("Rendering index failed", "error", err)
		}
	}
}

func makeSSEHandler(hub *Hub, leaderboard *Leaderboard, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set the mandatory SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)

			return
		}

		// Send an initial comment so the browser confirms the stream is alive.
		if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
			return
		}

		flusher.Flush()
		slog.Info("SSE client connected", "remote", r.RemoteAddr)

		streamEvents(w, r, flusher, templates, hub, leaderboard)
	}
}

// streamEvents pushes game state and leaderboard updates to a single SSE client
// until it disconnects or writing fails.
func streamEvents(
	w http.ResponseWriter,
	r *http.Request,
	flusher http.Flusher,
	templates *template.Template,
	hub *Hub,
	leaderboard *Leaderboard,
) {
	updates := hub.Subscribe()
	defer hub.Unsubscribe(updates)

	rankings := leaderboard.Subscribe()
	defer leaderboard.Unsubscribe(rankings)

	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()

	for {
		select {
		case state, ok := <-updates:
			if !ok {
				return
			}

			if err := writeEvent(w, flusher, templates, gameStateEvent, gameStateTemplate, state); err != nil {
				slog.Error("Rendering game state failed", "error", err)

				return
			}

		case ranking, ok := <-rankings:
			if !ok {
				return
			}

			if err := writeEvent(w, flusher, templates, leaderboardEvent, leaderboardTemplate, ranking); err != nil {
				slog.Error("Rendering leaderboard failed", "error", err)

				return
			}

		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return
			}

			flusher.Flush()

		case <-r.Context().Done():
			slog.Info("SSE client disconnected", "remote", r.RemoteAddr)

			return
		}
	}
}

func writeEvent(
	w http.ResponseWriter,
	flusher http.Flusher,
	templates *template.Template,
	eventName string,
	templateName string,
	data any,
) error {
	var body bytes.Buffer
	if err := templates.ExecuteTemplate(&body, templateName, data); err != nil {
		return fmt.Errorf("executing template %s: %w", templateName, err)
	}

	var event strings.Builder
	event.WriteString("event: " + eventName + "\n")

	for line := range strings.SplitSeq(body.String(), "\n") {
		event.WriteString("data: " + line + "\n")
	}

	event.WriteString("\n")

	if _, err := w.Write([]byte(event.String())); err != nil {
		return fmt.Errorf("writing %s event: %w", eventName, err)
	}

	flusher.Flush()

	return nil
}
