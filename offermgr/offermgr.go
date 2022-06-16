package offermgr

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
)

// Logger
var log = logging.Logger("offermgr")

// OfferManager is the interface for a manager that tracks the offer nonce.
type OfferManager interface {
	// GetNonce is used to get the current nonce.
	//
	// @input - context.
	//
	// @output - nonce, error.
	GetNonce(ctx context.Context) (uint64, error)

	// SetNonce is used to set the current nonce.
	//
	// @input - context, nonce.
	//
	// @output - error.
	SetNonce(ctx context.Context, nonce uint64) error
}
