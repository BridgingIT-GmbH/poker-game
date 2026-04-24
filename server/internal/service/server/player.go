package server

import "github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"

type Player struct {
	Name string
	Peer transport.PeerID
}
