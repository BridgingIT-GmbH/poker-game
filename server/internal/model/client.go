package model

import "github.com/BridgingIT-GmbH/poker-game/server/pkg/transport"

type Client struct {
	Name string
	Peer transport.PeerID
}
