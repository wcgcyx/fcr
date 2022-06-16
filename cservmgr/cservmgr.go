package cservmgr

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

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
)

// Logger
var log = logging.Logger("cservmgr")

// PieceServingManager is the interface for a manager that manages all served pieces.
type PieceServingManager interface {
	// Serve is used to serve a piece.
	//
	// @input - context, piece id, currency id, price-per-byte.
	//
	// @output - error.
	Serve(ctx context.Context, id cid.Cid, currencyID byte, ppb *big.Int) error

	// Stop is used to stop serving a piece.
	//
	// @input - context, piece id, currency id.
	//
	// @output - error.
	Stop(ctx context.Context, id cid.Cid, currencyID byte) error

	// Inspect is used to inspect a piece serving state.
	//
	// @input - context, piece id, currency id.
	//
	// @output - if served, price-per-byte, error.
	Inspect(ctx context.Context, id cid.Cid, currencyID byte) (bool, *big.Int, error)

	// ListPieceIDs lists all pieces.
	//
	// @input - context.
	//
	// @output - piece id chan out, error chan out.
	ListPieceIDs(ctx context.Context) (<-chan cid.Cid, <-chan error)

	// ListCurrencyIDs lists all currencies.
	//
	// @input - context, piece id.
	//
	// @output - currency chan out, error chan out.
	ListCurrencyIDs(ctx context.Context, id cid.Cid) (<-chan byte, <-chan error)

	// RegisterPublishHook adds a hook function to call when a new serving presents.
	//
	// @input - hook function.
	RegisterPublishFunc(hook func(cid.Cid))

	// DeregisterPublishHook deregister the hook function.
	DeregisterPublishHook()
}
