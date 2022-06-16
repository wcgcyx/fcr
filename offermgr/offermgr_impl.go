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
	"encoding/binary"
	"fmt"

	golock "github.com/viney-shih/go-lock"
	"github.com/wcgcyx/fcr/mdlstore"
)

// OfferManagerImpl is the implementation for OfferManager.
type OfferManagerImpl struct {
	// Cached nonce
	nonce uint64
	lock  golock.RWMutex

	// Datastore.
	ds mdlstore.MultiDimLockableStore

	// Shutdown function.
	shutdown func()
}

// NewOfferManagerImpl creates a new offer manager.
//
// @input - context, options.
//
// @output - offer manager, error.
func NewOfferManagerImpl(ctx context.Context, opts Opts) (*OfferManagerImpl, error) {
	log.Infof("Start offer nonce manager...")
	// Open store.
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, opts.Path)
	if err != nil {
		log.Errorf("Fail to open mdlstore: %v", err.Error())
		return nil, err
	}
	defer func() {
		if err != nil {
			log.Infof("Fail to start offer nonce manager, close ds...")
			err = ds.Shutdown(context.Background())
			if err != nil {
				log.Errorf("Fail to close ds after failing to start offer nonce manager: %v", err.Error())
			}
		}
	}()
	txn, err := ds.NewTransaction(ctx, false)
	if err != nil {
		log.Errorf("Fail to start new transasction for loading nonce cache: %v", err.Error())
		return nil, err
	}
	defer txn.Discard(context.Background())
	exists, err := txn.Has(ctx, 0)
	if err != nil {
		log.Errorf("Fail to check if contains key 0: %v", err.Error())
		return nil, err
	}
	var nonce uint64
	if exists {
		val, err := txn.Get(ctx, 0)
		if err != nil {
			log.Errorf("Fail to read the ds value for key 0: %v", err.Error())
			return nil, err
		}
		nonce = binary.LittleEndian.Uint64(val)
	} else {
		nonce = 0
	}
	mgr := &OfferManagerImpl{
		nonce: nonce,
		lock:  golock.NewCASMutex(),
		ds:    ds,
	}
	mgr.shutdown = func() {
		log.Infof("Save nonce to ds...")
		if !mgr.lock.TryLockWithContext(context.Background()) {
			log.Errorf("Fail to obtain write lock for current nonce")
			// Still continue the shutdown operation
		}
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, mgr.nonce)
		txn, err := mgr.ds.NewTransaction(ctx, false)
		if err != nil {
			log.Errorf("Fail to start new transaction to save current nonce of %v: %v", mgr.nonce, err.Error())
			// Still continue ds shutdown
		} else {
			defer txn.Discard(context.Background())
			err = txn.Put(ctx, b, 0)
			if err != nil {
				log.Errorf("Fail to put current nonce %v: %v", mgr.nonce, err.Error())
			}
			err = txn.Commit(context.Background())
			if err != nil {
				log.Errorf("Fail to commit transaction to save current nonce %v: %v", mgr.nonce, err.Error())
			}
		}
		log.Infof("Stop datastore...")
		err = ds.Shutdown(context.Background())
		if err != nil {
			log.Errorf("Fail to stop datastore: %v", err.Error())
		}
	}
	return mgr, nil
}

// Shutdown safely shuts down the component.
func (mgr *OfferManagerImpl) Shutdown() {
	log.Infof("Start shutdown...")
	mgr.shutdown()
}

// GetNonce is used to get the current nonce.
//
// @input - context.
//
// @output - nonce, error.
func (mgr *OfferManagerImpl) GetNonce(ctx context.Context) (uint64, error) {
	log.Debugf("Get nonce")
	if !mgr.lock.TryLockWithContext(ctx) {
		log.Debugf("Fail to obtain write lock for nonce")
		return 0, fmt.Errorf("fail to obtain write lock for nonce")
	}
	defer mgr.lock.Unlock()
	mgr.nonce += 1
	return mgr.nonce, nil
}

// SetNonce is used to set the current nonce.
//
// @input - context, nonce.
//
// @output - error.
func (mgr *OfferManagerImpl) SetNonce(ctx context.Context, nonce uint64) error {
	log.Debugf("Set nonce to be %v", nonce)
	if !mgr.lock.TryLockWithContext(ctx) {
		log.Debugf("Fail to obtain write lock for nonce")
		return fmt.Errorf("fail to obtain write lock for nonce")
	}
	defer mgr.lock.Unlock()
	mgr.nonce = nonce
	return nil
}
