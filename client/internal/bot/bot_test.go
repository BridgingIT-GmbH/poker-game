package bot

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
)

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
			if !equalStringSlices(got, test.want) {
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

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// ─── handleMessage: informational messages don't panic ───────────────────────

func TestHandleMessage_InformationalMessages_NoPanic(t *testing.T) {
	t.Parallel()

	tests := []string{
		fmt.Sprintf("%s;1;Alice,Bob,Carol", protocol.CmdRoundStarted),
		fmt.Sprintf("%s;As,Kh", protocol.CmdCards),
		fmt.Sprintf("%s;Td,7c,2h", protocol.CmdBoard),
		fmt.Sprintf("%s;Alice wins with a Flush", protocol.CmdScore),
		fmt.Sprintf("%s;Bob", protocol.CmdFold),
		fmt.Sprintf("%s;Bob", protocol.CmdCall),
		fmt.Sprintf("%s;Bob", protocol.CmdRaise),
		fmt.Sprintf("%s;Bob", protocol.CmdCheck),
		string(protocol.CmdRegistered),
		fmt.Sprintf("%s;some reason", protocol.CmdRejected),
	}

	for _, msg := range tests {
		msg := msg
		t.Run(msg, func(t *testing.T) {
			t.Parallel()

			b := &Bot{}
			b.handleMessage(msg) // must not panic
		})
	}
}

func TestHandleMessage_UnknownCommand_NoPanic(t *testing.T) {
	t.Parallel()

	b := &Bot{}
	b.handleMessage("SOME UNKNOWN COMMAND;payload") // must not panic
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
}

func TestHandleMessage_YourTurn_SendsCallByDefault(t *testing.T) {
	t.Parallel()

	botConn, serverConn := newLoopbackPair(t)
	b := &Bot{conn: botConn, name: "Alice"}

	b.handleMessage(fmt.Sprintf("%s;action-token-456", protocol.CmdYourTurn))

	got := readOneMessage(t, serverConn)
	want := fmt.Sprintf("%s;action-token-456", protocol.CmdCall)

	if got != want {
		t.Fatalf("sent message = %q, want %q", got, want)
	}
}

func TestHandleMessage_YourTurn_EmptyToken_DoesNotSend(t *testing.T) {
	t.Parallel()

	botConn, serverConn := newLoopbackPair(t)
	b := &Bot{conn: botConn, name: "Alice"}

	b.handleMessage(string(protocol.CmdYourTurn))

	if err := serverConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}

	buf := make([]byte, bufferSize)
	if _, _, err := serverConn.ReadFromUDP(buf); err == nil {
		t.Fatal("expected no message to be sent for empty token")
	}
}
