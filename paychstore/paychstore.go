package paychstore

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

	logging "github.com/ipfs/go-log"
	"github.com/wcgcyx/fcr/paychstate"
)

// Logger
var log1 = logging.Logger("active-out")
var log2 = logging.Logger("active-in")
var log3 = logging.Logger("inactive-out")
var log4 = logging.Logger("inactive-in")

// PaychStore is the interface for a storage storing all payment channels.
type PaychStore interface {
	// Upsert is used to update or insert a channel state.
	//
	// @input - channel state.
	Upsert(state paychstate.State)

	// Remove is used to remove a channel state.
	//
	// @input - currency id, peer address, channel address.
	Remove(currencyID byte, peerAddr string, chAddr string)

	// Read is used to read a channel state.
	//
	// @input - context, currency id, peer address, channel address.
	//
	// @output - channel state, error.
	Read(ctx context.Context, currencyID byte, peerAddr string, chAddr string) (paychstate.State, error)

	// ListCurrencyIDs is used to list all currencies.
	//
	// @input - context.
	//
	// @output - currency id chan out, error chan out.
	ListCurrencyIDs(ctx context.Context) (<-chan byte, <-chan error)

	// ListPeers is used to list all peers.
	//
	// @input - context, currency id.
	//
	// @output - peer address chan out, error chan out.
	ListPeers(ctx context.Context, currencyID byte) (<-chan string, <-chan error)

	// List channels by peer.
	//
	// @input - context, currency id, peer address.
	//
	// @output - channel address chan out, error chan out.
	ListPaychsByPeer(ctx context.Context, currencyID byte, peerAddr string) (<-chan string, <-chan error)
}
