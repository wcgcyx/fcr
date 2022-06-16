package mdlstore

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

	dsquery "github.com/ipfs/go-datastore/query"
)

const (
	DatastoreKeySeperator = "/"
)

// MultiDimLockableStore is the interface for a multi-dimensional lockable datastore.
// Every opetation needs to be done via transaction.
type MultiDimLockableStore interface {
	RWLock
	// NewTransaction creates a new transaction.
	//
	// @input - context, boolean indicating if only read operations will be accessed.
	//
	// @output - transaction, error.
	NewTransaction(ctx context.Context, readOnly bool) (Transaction, error)
}

// Transaction is used to batch queries and mutations to a datastore into atomic groups.
type Transaction interface {
	Read
	Write

	// Commit commits a transaction. Push all changes to the datastore.
	//
	// @input - context.
	//
	// @output - error.
	Commit(ctx context.Context) error

	// Discard discards a transaction.
	//
	// @input - context.
	Discard(ctx context.Context)
}

// Read is the interface to query datastore.
type Read interface {
	// Get gets the value for a given path.
	//
	// @input - context, path.
	//
	// @output - value, error.
	Get(ctx context.Context, path ...interface{}) ([]byte, error)

	// GetChildren gets the children for a given path.
	//
	// @input - context, path.
	//
	// @output - children, error.
	GetChildren(ctx context.Context, path ...interface{}) (map[string]bool, error)

	// Has checks if given path exists.
	//
	// @input - context, path.
	//
	// @output - boolean indicating if given path exists, error.
	Has(ctx context.Context, path ...interface{}) (bool, error)

	// Query queries the datastore for given path.
	//
	// @input - context, query, path.
	//
	// @output - query results, error.
	Query(ctx context.Context, query Query, path ...interface{}) (QueryResults, error)
}

// Write is the interface to mutate datastore.
type Write interface {
	// Put puts the value for a given path.
	//
	// @input - context, value, path.
	//
	// @output - error.
	Put(ctx context.Context, value []byte, path ...interface{}) error

	// Delete deletes a given path.
	//
	// @input - context, path.
	//
	// @output - error.
	Delete(ctx context.Context, path ...interface{}) error
}

// RWLock is the interface to do read/write lock.
type RWLock interface {
	// RLock is used to read lock a given path.
	//
	// @input - context, path.
	//
	// @output - function to release lock, error.
	RLock(ctx context.Context, path ...interface{}) (func(), error)

	// Lock is used to write lock a given path.
	//
	// @input - context, path.
	//
	// @output - function to release lock, error.
	Lock(ctx context.Context, path ...interface{}) (func(), error)
}

// Query is a wrapper over the dsquery.Query.
type Query struct {
	Filters           []dsquery.Filter // filter results. apply sequentially
	Orders            []dsquery.Order  // order results. apply hierarchically
	Limit             int              // maximum number of results
	Offset            int              // skip given number of results
	KeysOnly          bool             // return only keys.
	ReturnExpirations bool             // return expirations (see TTLDatastore)
	ReturnsSizes      bool             // always return sizes. If not set, datastore impl can return
	//                        		   // it anyway if it doesn't involve a performance cost. If KeysOnly
	//                         		   // is not set, Size should always be set.
}

// QueryResults is a wrapper over the dsquery.Query.
type QueryResults struct {
	Results dsquery.Results // Query results.
}
