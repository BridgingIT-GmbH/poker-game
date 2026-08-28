package bot

import (
	"fmt"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

// ─── helper: stub strategy ────────────────────────────────────────────────────

// stubStrategy records the state it was called with and returns a fixed action.
type stubStrategy struct {
	action     Action
	calledWith GameState
	called     bool
}

func (s *stubStrategy) Decide(state GameState) Action {
	s.called = true
	s.calledWith = state

	return s.action
}

// ─── helpers: parseCards / payloadOr ──────────────────────────────────────────

func TestParseCards(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"empty string", "", nil},
		{"single card", "As", []string{"As"}},
		{"multiple cards", "As,Kh,Td", []string{"As", "Kh", "Td"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := parseCards(test.raw)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("parseCards(%q) = %v, want %v", test.raw, got, test.want)
			}
		})
	}
}

func TestPayloadOr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		parts    []string
		fallback string
		want     string
	}{
		{"no parts", []string{}, "fallback", "fallback"},
		{"single part", []string{"CMD"}, "fallback", "fallback"},
		{"two parts", []string{"CMD", "payload"}, "fallback", "payload"},
		{"more parts", []string{"CMD", "payload", "extra"}, "fallback", "payload"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := payloadOr(test.parts, test.fallback)
			if got != test.want {
				t.Fatalf("payloadOr(%v, %q) = %q, want %q", test.parts, test.fallback, got, test.want)
			}
		})
	}
}

// ─── handleMessage: state updates without network I/O ────────────────────────

func TestHandleMessage_RoundStarted(t *testing.T) {
	t.Parallel()

	b := &Bot{}
	msg := fmt.Sprintf("%s;1;Alice,Bob,Carol", protocol.CmdRoundStarted)
	b.handleMessage(msg)

	wantPlayers := []string{"Alice", "Bob", "Carol"}
	if !reflect.DeepEqual(b.state.Players, wantPlayers) {
		t.Fatalf("Players = %v, want %v", b.state.Players, wantPlayers)
	}

	if b.state.FoldedPlayers == nil || len(b.state.FoldedPlayers) != 0 {
		t.Fatalf("FoldedPlayers = %v, want empty map", b.state.FoldedPlayers)
	}
}

func TestHandleMessage_Cards(t *testing.T) {
	t.Parallel()

	b := &Bot{}
	b.handleMessage(fmt.Sprintf("%s;As,Kh", protocol.CmdCards))

	want := []string{"As", "Kh"}
	if !reflect.DeepEqual(b.state.HoleCards, want) {
		t.Fatalf("HoleCards = %v, want %v", b.state.HoleCards, want)
	}
}

func TestHandleMessage_Board(t *testing.T) {
	t.Parallel()

	b := &Bot{}
	b.handleMessage(fmt.Sprintf("%s;Td,7c,2h", protocol.CmdBoard))

	want := []string{"Td", "7c", "2h"}
	if !reflect.DeepEqual(b.state.Board, want) {
		t.Fatalf("Board = %v, want %v", b.state.Board, want)
	}
}

func TestHandleMessage_Score(t *testing.T) {
	t.Parallel()

	b := &Bot{}
	b.handleMessage(fmt.Sprintf("%s;Alice wins with a Flush", protocol.CmdScore))

	want := "Alice wins with a Flush"
	if b.state.LastScore != want {
		t.Fatalf("LastScore = %q, want %q", b.state.LastScore, want)
	}
}

func TestHandleMessage_Fold(t *testing.T) {
	t.Parallel()

	b := &Bot{}
	b.state.Players = []string{"Alice", "Bob"}
	b.state.FoldedPlayers = make(map[string]bool)

	b.handleMessage(fmt.Sprintf("%s;Bob", protocol.CmdFold))

	if !b.state.FoldedPlayers["Bob"] {
		t.Fatal("expected Bob to be marked as folded")
	}

	want := []string{"Alice"}
	if got := b.state.ActivePlayers(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ActivePlayers() = %v, want %v", got, want)
	}
}

func TestHandleMessage_Fold_InitializesNilMap(t *testing.T) {
	t.Parallel()

	b := &Bot{} // FoldedPlayers is nil
	b.handleMessage(fmt.Sprintf("%s;Alice", protocol.CmdFold))

	if !b.state.FoldedPlayers["Alice"] {
		t.Fatal("expected Alice to be marked as folded even with initially nil map")
	}
}

func TestHandleMessage_UnknownCommand_NoPanic(t *testing.T) {
	t.Parallel()

	b := &Bot{}
	b.handleMessage("SOME UNKNOWN COMMAND;payload")
	// No assertions needed - just ensure it doesn't panic and state stays zero.
	if b.state.LastScore != "" {
		t.Fatal("state should remain unchanged for unknown commands")
	}
}

// ─── handleMessage: commands that trigger network sends ──────────────────────

// newLoopbackPair creates a connected UDP pair for testing bot sends.
// It returns the bot-side connection (to be used by the Bot) and the
// listener-side connection (to read what the Bot sent).
func newLoopbackPair(t *testing.T) (botConn *net.UDPConn, serverConn *net.UDPConn) {
	t.Helper()

	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	t.Cleanup(func() { server.Close() })

	client, err := net.DialUDP("udp", nil, server.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	t.Cleanup(func() { client.Close() })

	return client, server
}

func readOneMessage(t *testing.T, conn *net.UDPConn) string {
	t.Helper()

	buf := make([]byte, bufferSize)

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}

	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	return strings.TrimRight(string(buf[:n]), "\n")
}

func TestHandleMessage_RoundStarting_SendsJoin(t *testing.T) {
	t.Parallel()

	botConn, serverConn := newLoopbackPair(t)
	b := &Bot{conn: botConn, name: "Alice"}

	b.handleMessage(fmt.Sprintf("%s;join-token-123", protocol.CmdRoundStarting))

	got := readOneMessage(t, serverConn)
	want := fmt.Sprintf("%s;join-token-123", protocol.CmdJoin)

	if got != want {
		t.Fatalf("sent message = %q, want %q", got, want)
	}

	// Round starting must also reset the state.
	if b.state.HoleCards != nil || b.state.Board != nil {
		t.Fatal("expected state to be reset on ROUND STARTING")
	}
}

func TestHandleMessage_YourTurn_SendsStrategyDecision(t *testing.T) {
	t.Parallel()

	botConn, serverConn := newLoopbackPair(t)
	strategy := &stubStrategy{action: ActionRaise}
	b := &Bot{conn: botConn, name: "Alice", strategy: strategy}
	b.state.HoleCards = []string{"As", "Kh"}

	b.handleMessage(fmt.Sprintf("%s;action-token-456", protocol.CmdYourTurn))

	if !strategy.called {
		t.Fatal("expected strategy.Decide to be called")
	}

	if !reflect.DeepEqual(strategy.calledWith.HoleCards, []string{"As", "Kh"}) {
		t.Fatalf("strategy called with unexpected state: %+v", strategy.calledWith)
	}

	got := readOneMessage(t, serverConn)
	want := fmt.Sprintf("%s;action-token-456", ActionRaise)

	if got != want {
		t.Fatalf("sent message = %q, want %q", got, want)
	}
}

func TestHandleMessage_YourTurn_EmptyToken_DoesNotCallStrategy(t *testing.T) {
	t.Parallel()

	strategy := &stubStrategy{action: ActionCall}
	b := &Bot{strategy: strategy}

	b.handleMessage(string(protocol.CmdYourTurn))

	if strategy.called {
		t.Fatal("expected strategy.Decide NOT to be called when token is empty")
	}
}

