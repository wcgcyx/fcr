package payservstore

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"math/big"
	"time"
)

// PayServStore is used to store all the servings for the payment proxy.
// It is used by the user to serve a particular payment address.
type PayServStore interface {
	// Serve will start serving a address.
	// It takes a currency id, the to address, ppp (price per period), period as arguments.
	// It returns error.
	Serve(currencyID uint64, toAddr string, ppp *big.Int, period *big.Int) error

	// Inspect will check a given serving.
	// It takes a currency id, the to address as arguments.
	// It returns the ppp (price per period), period, expiration and error.
	Inspect(currencyID uint64, toAddr string) (*big.Int, *big.Int, int64, error)

	// Retire starts the retiring process of a serving.
	// It takes a currency id, the to address, time to live (ttl) as arguments.
	// It returns the error.
	Retire(currencyID uint64, toAddr string, ttl time.Duration) error

	// ListActive will lists all the active servings.
	// It takes a context as the argument.
	// It returns a map from currency id to list of active servings and error.
	ListActive(ctx context.Context) (map[uint64][]string, error)

	// ListRetiring will lists all the retiring servings.
	// It takes a context as the argument.
	// It returns a map from currency id to list of retiring servings and error.
	ListRetiring(ctx context.Context) (map[uint64][]string, error)
}
