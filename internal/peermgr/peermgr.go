package peermgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
)

// PeerManager is used to keep track of every peer.
// It also maintain a simple local reputation table.
type PeerManager interface {
	// AddPeer adds a peer to the manager.
	// It takes a currency id, peer address and peer id as arguments.
	// It returns error.
	AddPeer(currencyID uint64, toAddr string, pid peer.ID) error

	// Record record a success or failure to a peer.
	// It takes a currency id, peer address, boolean indicates whether succeed as arguments.
	Record(currencyID uint64, toAddr string, succeed bool)

	// GetPeerInfo gets the information of a peer.
	// It takes a currency id, peer address as arguments.
	// It returns the peer id, a boolean indicate whether it is blocked, success count, failure count and error.
	GetPeerInfo(currencyID uint64, toAddr string) (peer.ID, bool, int, int, error)

	// BlockPeer is used to block a peer.
	// It takes a currency id, peer address to block as arguments.
	// It returns error.
	BlockPeer(currencyID uint64, toAddr string) error

	// UnblockPeer is used to unblock a peer.
	// It takes a currency id, peer address to unblock as arguments.
	// It returns error.
	UnblockPeer(currencyID uint64, toAddr string) error

	// ListPeers is used to list all the peers.
	// It takes a context as the argument.
	// It returns a map from currency id to list of peers and error.
	ListPeers(ctx context.Context) (map[uint64][]string, error)
}
