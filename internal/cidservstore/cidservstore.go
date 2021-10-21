package cidservstore

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"math/big"
	"time"

	"github.com/ipfs/go-cid"
)

// CIDServStore is used to store all the servings for the cid.
// It is used by the user to serve a particular cid.
type CIDServStore interface {
	// Serve will start serving a cid
	// It takes a currency id, the cid, ppb (price per byte) as arguments.
	// It returns error.
	Serve(currencyID uint64, id cid.Cid, ppb *big.Int) error

	// Inspect will check a given serving.
	// It takes a currency id, the cid as arguments.
	// It returns the ppb (price per byte), expiration and error.
	Inspect(currencyID uint64, id cid.Cid) (*big.Int, int64, error)

	// Retire starts the retiring process of a serving.
	// It takes a currency id, the cid, time to live (ttl) as arguments.
	// It returns the error.
	Retire(currencyID uint64, id cid.Cid, ttl time.Duration) error

	// ListActive will lists all the active servings.
	// It takes a context as the argument.
	// It returns a map from currency id to list of active servings and error.
	ListActive(ctx context.Context) (map[uint64][]cid.Cid, error)

	// ListRetiring will lists all the retiring servings.
	// It takes a context as the argument.
	// It returns a map from currency id to list of retiring servings and error.
	ListRetiring(ctx context.Context) (map[uint64][]cid.Cid, error)
}
