package pservmgr

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

// Logger
var log = logging.Logger("pservmgr")

// PaychServingManager is the interface for a manager that manages all served payment channels.
type PaychServingManager interface {
	// Serve is used to serve a payment channel.
	//
	// @input - context, currency id, recipient address, channel address, price-per-period, period.
	//
	// @output - error.
	Serve(ctx context.Context, currencyID byte, toAddr string, chAddr string, ppp *big.Int, period *big.Int) error

	// Stop is used to stop serving a payment channel.
	//
	// @input - context, currency id, recipient address, channel address.
	//
	// @output - error.
	Stop(ctx context.Context, currencyID byte, toAddr string, chAddr string) error

	// Inspect is used to inspect a payment channel serving state.
	//
	// @input - context, currency id, recipient address, channel address.
	//
	// @output - if served, price-per-period, period, error.
	Inspect(ctx context.Context, currencyID byte, toAddr string, chAddr string) (bool, *big.Int, *big.Int, error)

	// ListCurrencyIDs lists all currencies.
	//
	// @input - context.
	//
	// @output - currency chan out, error chan out.
	ListCurrencyIDs(ctx context.Context) (<-chan byte, <-chan error)

	// ListRecipients lists all recipients.
	//
	// @input - context, currency id.
	//
	// @output - recipient chan out, error chan out.
	ListRecipients(ctx context.Context, currencyID byte) (<-chan string, <-chan error)

	// ListServings lists all servings.
	//
	// @input - context, currency id, recipient address.
	//
	// @output - paych chan out, error chan out.
	ListServings(ctx context.Context, currencyID byte, toAddr string) (<-chan string, <-chan error)
}
