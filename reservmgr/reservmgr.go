package reservmgr

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
)

const (
	DefaultPolicyID = "default"
)

// Logger
var log = logging.Logger("reservmgr")

// ReservationManager is the interface for a manager that is used to manage all reservation policy.
// If a policy has a maximum of 0, it means the policy is to reject.
type ReservationManager interface {
	// SetDefaultPolicy sets the default reservation policy.
	//
	// @input - context, currency id, if unlimited reservation, maximum single reservation amount.
	//
	// @output - error.
	SetDefaultPolicy(ctx context.Context, currencyID byte, unlimited bool, max *big.Int) error

	// GetDefaultPolicy gets the default reservation policy.
	//
	// @input - context, currency id.
	//
	// @output - if unlimited, max single reservation amount, error.
	GetDefaultPolicy(ctx context.Context, currencyID byte) (bool, *big.Int, error)

	// RemoveDefaultPolicy removes the default reservation policy.
	//
	// @input - context, currency id.
	//
	// @output - error.
	RemoveDefaultPolicy(ctx context.Context, currencyID byte) error

	// SetPaychPolicy sets the reservation policy for a given paych.
	//
	// @input - context, currency id, channel address, if unlimited reservation, maximum single reservation amount.
	//
	// @output - error.
	SetPaychPolicy(ctx context.Context, currencyID byte, chAddr string, unlimited bool, max *big.Int) error

	// GetPaychPolicy gets the reservation policy for a given paych.
	//
	// @input - context, currency id, channel address.
	//
	// @output - if unlimited reservation, maximum single reservation amount, error.
	GetPaychPolicy(ctx context.Context, currencyID byte, chAddr string) (bool, *big.Int, error)

	// RemovePaychPolicy removes the reservation policy for a given paych.
	//
	// @input - context, currency id, channel address.
	//
	// @output - error.
	RemovePaychPolicy(ctx context.Context, currencyID byte, chAddr string) error

	// SetPeerPolicy sets the reservation policy for a given peer.
	//
	// @input - context, currency id, channel address, peer address, if unlimited reservation, maximum single reservation amount.
	//
	// @output - error.
	SetPeerPolicy(ctx context.Context, currencyID byte, chAddr string, peerAddr string, unlimited bool, max *big.Int) error

	// GetPeerPolicy gets the reservation policy for a given peer.
	//
	// @input - context, currency id, channel address, peer address.
	//
	// @output - if unlimited reservation, maximum single reservation amount, error.
	GetPeerPolicy(ctx context.Context, currencyID byte, chAddr string, peerAddr string) (bool, *big.Int, error)

	// RemovePeerPolicy removes the reservation policy for a given peer.
	//
	// @input - context, currency id, channel address, peer address.
	//
	// @output - error.
	RemovePeerPolicy(ctx context.Context, currencyID byte, chAddr string, peerAddr string) error

	// ListCurrencyIDs lists all currency ids.
	//
	// @input - context.
	//
	// @output - currency id chan out, error chan out.
	ListCurrencyIDs(ctx context.Context) (<-chan byte, <-chan error)

	// ListPaychs lists all paychs.
	//
	// @input - context, currency id.
	//
	// @output - paych chan out, error chan out.
	ListPaychs(ctx context.Context, currencyID byte) (<-chan string, <-chan error)

	// ListPeers lists all peers.
	//
	// @input - context, currency id, paych address.
	//
	// @output - peer chan out, error chan out.
	ListPeers(ctx context.Context, currencyID byte, chAddr string) (<-chan string, <-chan error)
}
