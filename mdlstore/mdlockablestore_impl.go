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
	"fmt"
	"strings"

	"github.com/ipfs/go-datastore"
	dsquery "github.com/ipfs/go-datastore/query"
	badgerds "github.com/ipfs/go-ds-badger"
	"github.com/wcgcyx/fcr/locking"
)

// MultiDimLockableStoreImpl is the implementation of the MultiDimLockableStore interface.
type MultiDimLockableStoreImpl struct {
	// datastore
	ds *badgerds.Datastore

	// root node of the lock tree
	root *locking.LockNode
}

// TransactionImpl is the implementation of the Transaction interface.
type TransactionImpl struct {
	// datastore transaction
	txn datastore.Txn

	// root node of the lock tree
	root *locking.LockNode

	// commit functions to update the lock tree
	commits []func()
}

// NewMultiDimLockableStoreImpl creates a new MultiDimLockableStore.
//
// @input - context, path.
//
// @output - store, error.
func NewMultiDimLockableStoreImpl(ctx context.Context, path string) (*MultiDimLockableStoreImpl, error) {
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	if path == "" {
		return nil, fmt.Errorf("empty path provided")
	}
	ds, err := badgerds.NewDatastore(path, &dsopts)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			ds.Close()
		}
	}()
	// If there is no root node, create a root node in the ds.
	exists, err := ds.Has(ctx, datastore.NewKey(""))
	if err != nil {
		return nil, err
	}
	if !exists {
		dsVal, err := encDSVal([]byte{0}, make(map[string]bool))
		if err != nil {
			return nil, err
		}
		err = ds.Put(ctx, datastore.NewKey(""), dsVal)
		if err != nil {
			return nil, err
		}
	}
	tempRes, err := ds.Query(ctx, dsquery.Query{KeysOnly: true})
	if err != nil {
		return nil, err
	}
	output := make(chan string, dsquery.KeysOnlyBufSize)

	go func() {
		defer func() {
			tempRes.Close()
			close(output)
		}()

		for {
			e, ok := tempRes.NextSync()
			if !ok {
				break
			}
			if e.Error != nil {
				err = e.Error
				break
			}
			select {
			case output <- e.Key[1:]:
			}
		}
	}()
	root := locking.CreateLockNode()
	for keys := range output {
		path := make([]interface{}, 0)
		for _, key := range strings.Split(keys, DatastoreKeySeperator) {
			path = append(path, key)
		}
		root.Insert(path...)
	}
	if err != nil {
		return nil, err
	}
	return &MultiDimLockableStoreImpl{ds: ds, root: root}, nil
}

// Shutdown safely shuts down the store.
//
// @input - context.
//
// @output - error.
func (s *MultiDimLockableStoreImpl) Shutdown(ctx context.Context) error {
	// Attempt to obtain the root lock.
	_, err := s.root.Lock(ctx)
	if err != nil {
		return err
	}
	return s.ds.Close()
}

// RLock is used to read lock a given path.
//
// @input - context, path.
//
// @output - function to release lock, error.
func (s *MultiDimLockableStoreImpl) RLock(ctx context.Context, path ...interface{}) (func(), error) {
	return s.root.RLock(ctx, path...)
}

// Lock is used to write lock a given path.
//
// @input - context, path.
//
// @output - function to release lock, error.
func (s *MultiDimLockableStoreImpl) Lock(ctx context.Context, path ...interface{}) (func(), error) {
	return s.root.Lock(ctx, path...)
}

// NewTransaction creates a new transaction.
//
// @input - context, boolean indicating if only read operations will be accessed.
//
// @output - transaction, error.
func (s *MultiDimLockableStoreImpl) NewTransaction(ctx context.Context, readOnly bool) (Transaction, error) {
	txn, err := s.ds.NewTransaction(ctx, readOnly)
	if err != nil {
		return nil, err
	}
	return &TransactionImpl{txn: txn, root: s.root, commits: make([]func(), 0)}, nil
}

// Get gets the value for a given path.
//
// @input - context, path.
//
// @output - data, error.
func (t *TransactionImpl) Get(ctx context.Context, path ...interface{}) ([]byte, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("cannot read root node")
	}
	key, err := getDSKey(path...)
	if err != nil {
		return nil, err
	}
	val, err := t.txn.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	data, _, err := decDSVal(val)
	return data, err
}

// GetChildren gets the children for a given path.
//
// @input - context, path.
//
// @output - children, error.
func (t *TransactionImpl) GetChildren(ctx context.Context, path ...interface{}) (map[string]bool, error) {
	key, err := getDSKey(path...)
	if err != nil {
		return nil, err
	}
	val, err := t.txn.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	_, children, err := decDSVal(val)
	return children, err
}

// Has checks if given path exists.
//
// @input - context, path.
//
// @output - boolean indicating if given path exists, error.
func (t *TransactionImpl) Has(ctx context.Context, path ...interface{}) (bool, error) {
	key, err := getDSKey(path...)
	if err != nil {
		return false, err
	}
	return t.txn.Has(ctx, key)
}

// Query queries the datastore for given path.
//
// @input - context, query, path.
//
// @output - query results, error.
func (t *TransactionImpl) Query(ctx context.Context, query Query, path ...interface{}) (QueryResults, error) {
	prefix := ""
	if len(path) > 0 {
		key, err := getDSKey(path...)
		if err != nil {
			return QueryResults{}, err
		}
		prefix = key.String()
	}
	res, err := t.txn.Query(ctx, dsquery.Query{
		Prefix:            prefix,
		Filters:           query.Filters,
		Orders:            query.Orders,
		Limit:             query.Limit,
		Offset:            query.Offset,
		KeysOnly:          query.KeysOnly,
		ReturnExpirations: query.ReturnExpirations,
		ReturnsSizes:      query.ReturnsSizes,
	})
	return QueryResults{Results: res}, err
}

// Put puts the value for a given path.
//
// @input - context, value, path.
//
// @output - error.
func (t *TransactionImpl) Put(ctx context.Context, value []byte, path ...interface{}) error {
	if len(path) == 0 {
		return fmt.Errorf("cannot modify root node")
	}
	parentPath := []interface{}{}
	parentKey, _ := getDSKey(parentPath...)
	for i := 1; i < len(path); i++ {
		subPath := path[0:i]
		subKey, err := getDSKey(subPath...)
		if err != nil {
			return err
		}
		// Check if parent has this subPath.
		dsVal, err := t.txn.Get(ctx, parentKey)
		if err != nil {
			return err
		}
		parentData, children, err := decDSVal(dsVal)
		if err != nil {
			return err
		}
		child := fmt.Sprintf("%v", subPath[len(subPath)-1])
		_, ok := children[child]
		if !ok {
			// Parent does not have this child, update parent and create child.
			// Update parent.
			children[child] = true
			dsVal, err = encDSVal(parentData, children)
			if err != nil {
				return err
			}
			err = t.txn.Put(ctx, parentKey, dsVal)
			if err != nil {
				return err
			}
			// Create child.
			dsVal, err = encDSVal(nil, make(map[string]bool))
			if err != nil {
				return err
			}
			err = t.txn.Put(ctx, subKey, dsVal)
			if err != nil {
				return err
			}
		}
		parentPath = subPath
		parentKey = subKey
	}
	// This is the leaf node.
	// Check if parent has this node.
	dsVal, err := t.txn.Get(ctx, parentKey)
	if err != nil {
		return err
	}
	parentData, children, err := decDSVal(dsVal)
	if err != nil {
		return err
	}
	child := fmt.Sprintf("%v", path[len(path)-1])
	_, ok := children[child]
	if !ok {
		// Update parent.
		children[child] = true
		dsVal, err = encDSVal(parentData, children)
		if err != nil {
			return err
		}
		err = t.txn.Put(ctx, parentKey, dsVal)
		if err != nil {
			return err
		}
	}
	// Upsert leaf.
	key, err := getDSKey(path...)
	if err != nil {
		return err
	}
	dsVal, err = encDSVal(value, make(map[string]bool))
	if err != nil {
		return err
	}
	if err = t.txn.Put(ctx, key, dsVal); err == nil {
		t.commits = append(t.commits, func() { t.root.Insert(path...) })
	}
	return err
}

// Delete deletes a given path.
//
// @input - context, path.
//
// @output - error.
func (t *TransactionImpl) Delete(ctx context.Context, path ...interface{}) error {
	if len(path) == 0 {
		return fmt.Errorf("cannot delete root node")
	}
	// Update parent and remove path.
	// Update parent.
	parentPath := path[0 : len(path)-1]
	parentKey, err := getDSKey(parentPath...)
	if err != nil {
		return err
	}
	dsVal, err := t.txn.Get(ctx, parentKey)
	if err != nil {
		return err
	}
	parentData, children, err := decDSVal(dsVal)
	if err != nil {
		return err
	}
	child := fmt.Sprintf("%v", path[len(path)-1])
	delete(children, child)
	dsVal, err = encDSVal(parentData, children)
	if err != nil {
		return err
	}
	err = t.txn.Put(ctx, parentKey, dsVal)
	if err != nil {
		return err
	}
	// Remove child.
	key, err := getDSKey(path...)
	if err != nil {
		return err
	}
	if err = t.txn.Delete(ctx, key); err == nil {
		t.commits = append(t.commits, func() { t.root.Remove(path...) })
	}
	return err
}

// Commit commits a transaction. Push all changes to the datastore.
//
// @input - context.
//
// @output - error.
func (t *TransactionImpl) Commit(ctx context.Context) error {
	err := t.txn.Commit(ctx)
	if err == nil {
		for _, commit := range t.commits {
			commit()
		}
	}
	return err
}

// Discard discards a transaction.
//
// @input - context.
func (t *TransactionImpl) Discard(ctx context.Context) {
	t.txn.Discard(ctx)
	t.commits = make([]func(), 0)
}
