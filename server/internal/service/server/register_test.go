package server

import (
	"testing"

	"github.com/BridgingIT-GmbH/poker-game/server/internal/model"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/protocol"
	"github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"
)

func TestRegisterPlayer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		playerName     string
		existingPlayer *model.Client
		expectedMsg    string
		shouldRegister bool
	}{
		{"valid player", "Adam", nil, string(protocol.CmdRegistered + "\n"), true},
		{"empty name", "", nil, string(protocol.CmdRejected + ";Name is empty\n"), false},
		{
			"duplicate name",
			"Bob",
			&model.Client{Name: "Bob", Peer: "known"},
			string(protocol.CmdRejected + ";Name already taken\n"),
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			outbound := make(chan transport.OutboundPackage, 8)
			core := newServerCore(outbound)

			if test.existingPlayer != nil {
				core.players[test.existingPlayer.Name] = test.existingPlayer
			}

			peer := transport.PeerID("peer-1")
			core.registerPlayer(test.playerName, peer)

			pkg := readOutbound(t, outbound)
			assertOutbound(t, pkg, peer, test.expectedMsg)

			_, exists := core.players[test.playerName]
			if test.shouldRegister && !exists {
				t.Fatal("player should be registered but was not")
			}

			if !test.shouldRegister &&
				exists &&
				test.playerName != "" &&
				(test.existingPlayer == nil || test.playerName != test.existingPlayer.Name) {
				t.Fatal("player should not be registered but was")
			}
		})
	}
}
