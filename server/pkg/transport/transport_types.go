package transport

// PeerID identifies a remote client independent of the underlying transport.
// For UDP we currently use addr.String() ("ip:port").
type PeerID string

type InboundPackage struct {
	Peer PeerID
	Data []byte
}

type OutboundPackage struct {
	Peer PeerID
	Data []byte
}
