package peermgr

/*
 * Dual-licensed under Apache-2.0 and MIT.
 *
 * You can get a copy of the Apache License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * You can also get a copy of the MIT License at
 *
 * http://opensource.org/licenses/MIT
 *
 * @wcgcyx - https://github.com/wcgcyx
 */

import (
	"context"
	"math/big"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/peer"
)

// Logger
var log = logging.Logger("peermgr")

// PeerManager is an interface for a peer manager that not only tracks each peer by
// maintaining a map of (toAddr -> peer addr), also tracks the history of each peer,
// where history is a collection of record. It can also be used to block/unblock a
// peer.
type PeerManager interface {
	// AddPeer adds a peer. If there is an existing peer presented, overwrite the peer addr.
	//
	// @input - context, currency id, peer wallet address, peer p2p address.
	//
	// @output - error.
	AddPeer(ctx context.Context, currencyID byte, toAddr string, pi peer.AddrInfo) error

	// HasPeer checks if a given peer exists.
	//
	// @input - context, currency id, peer wallet address.
	//
	// @output - if exists, error.
	HasPeer(ctx context.Context, currencyID byte, toAddr string) (bool, error)

	// RemovePeer removes a peer from the manager.
	//
	// @input - context, currency id, peer wallet address.
	//
	// @output - error.
	RemovePeer(ctx context.Context, currencyID byte, toAddr string) error

	// PeerAddr gets the peer addr by given wallet address.
	//
	// @input - context, currency id, peer wallet address.
	//
	// @output - peer p2p address, error.
	PeerAddr(ctx context.Context, currencyID byte, toAddr string) (peer.AddrInfo, error)

	// IsBlocked checks if given peer is currently blocked.
	//
	// @input - context, currency id, peer wallet address.
	//
	// @output - boolean indicating whether it is blocked, error.
	IsBlocked(ctx context.Context, currencyID byte, toAddr string) (bool, error)

	// BlockPeer blocks a peer.
	//
	// @input - context, currency id, peer wallet address.
	//
	// @output - error.
	BlockPeer(ctx context.Context, currencyID byte, toAddr string) error

	// UnblockPeer unblocks a peer.
	//
	// @input - context, currency id, peer wallet address.
	//
	// @output - error.
	UnblockPeer(ctx context.Context, currencyID byte, toAddr string) error

	// ListCurrencyIDs lists all currencies.
	//
	// @input - context.
	//
	// @output - currency chan out, error chan out.
	ListCurrencyIDs(ctx context.Context) (<-chan byte, <-chan error)

	// ListPeers lists all peers stored by the manager.
	//
	// @input - context, currency id.
	//
	// @output - peer wallet address channel out, error channel out.
	ListPeers(ctx context.Context, currencyID byte) (<-chan string, <-chan error)

	// AddToHistory adds a record to a given peer's history.
	//
	// @input - context, currency id, peer wallet address, record.
	//
	// @output - error.
	AddToHistory(ctx context.Context, currencyID byte, toAddr string, rec Record) error

	// RemoveRecord removes a record from a given peer's history.
	//
	// @input - context, currency id, peer wallet address, record id.
	//
	// @output - error.
	RemoveRecord(ctx context.Context, currencyID byte, toAddr string, recID *big.Int) error

	// SetRecID sets the next record id to use for a given peer's history.
	//
	// @input - context, currency id, peer wallet address, record id.
	//
	// @output - error.
	SetRecID(ctx context.Context, currencyID byte, toAddr string, recID *big.Int) error

	// ListHistory lists all recorded history of given peer. The result will be given from
	// latest to oldest.
	//
	// @input - context, currency id, peer wallet address.
	//
	// @output - record id channel out, record channel out, error channel out.
	ListHistory(ctx context.Context, currencyID byte, toAddr string) (<-chan *big.Int, <-chan Record, <-chan error)
}
