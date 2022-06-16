package substore

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
var log = logging.Logger("substore")

// SubStore is the interface for a storage that stores all subscribers.
type SubStore interface {
	// AddSubscriber is used to add a subscriber.
	//
	// @input - context, currency id, from address.
	//
	// @output - error.
	AddSubscriber(ctx context.Context, currencyID byte, fromAddr string) error

	// RemoveSubscriber is used to remove a subscriber.
	//
	// @input - context, currency id, from address.
	//
	// @output - error.
	RemoveSubscriber(ctx context.Context, currencyID byte, fromAddr string) error

	// ListCurrencyIDs is used to list all currencies.
	//
	// @input - context.
	//
	// @output - currency id chan out, error chan out.
	ListCurrencyIDs(ctx context.Context) (<-chan byte, <-chan error)

	// ListSubscribers is used to list all subscribers.
	//
	// @input - context, currency id.
	//
	// @output - subscriber chan out, error chan out.
	ListSubscribers(ctx context.Context, currencyID byte) (<-chan string, <-chan error)
}
