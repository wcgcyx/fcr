package routestore

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
	"time"

	logging "github.com/ipfs/go-log"
)

const (
	RouteKeySeperator = "-"
)

// Logger
var log = logging.Logger("routestore")

// RouteStore is the interface for a route store that storing all routes.
type RouteStore interface {
	// MaxHop is used to get the max hop supported for given currency.
	//
	// @input - currency id.
	//
	// @output - max hop, error.
	MaxHop(ctx context.Context, currencyID byte) (uint64, error)

	// AddDirectLink is used to add direct link from this node.
	//
	// @input - context, currency id, recipient address.
	//
	// @output - error.
	AddDirectLink(ctx context.Context, currencyID byte, toAddr string) error

	// RemoveDirectLink is used to remove direct link from this node.
	//
	// @input - context, currency id, recipient address.
	//
	// @output - error.
	RemoveDirectLink(ctx context.Context, currencyID byte, toAddr string) error

	// AddRoute is used to add a route.
	//
	// @input - context, currency id, route, time to live.
	//
	// @output - error.
	AddRoute(ctx context.Context, currencyID byte, route []string, ttl time.Duration) error

	// RemoveRoute is used to remove a route.
	//
	// @input - context, currency id, route.
	//
	// @output - error.
	RemoveRoute(ctx context.Context, currencyID byte, route []string) error

	// ListRoutesTo is used to list all routes that has given destination.
	//
	// @input - context, currency id.
	//
	// @output - route chan out, error chan out.
	ListRoutesTo(ctx context.Context, currencyID byte, toAddr string) (<-chan []string, <-chan error)

	// ListRoutesFrom is used to list all routes that has given source.
	//
	// @input - context, currency id.
	//
	// @output - route chan out, error chan out.
	ListRoutesFrom(ctx context.Context, currencyID byte, toAddr string) (<-chan []string, <-chan error)
}
