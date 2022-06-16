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
	"fmt"
	"math/big"
	"strconv"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/mr-tron/base58/base58"
	"github.com/wcgcyx/fcr/mdlstore"
)

// PieceServingManagerImpl is the implementation of the PieceServingManager interface.
type PieceServingManagerImpl struct {
	// Datastore
	ds mdlstore.MultiDimLockableStore

	// Publish hook
	publishHookLock sync.RWMutex
	publishHook     func(cid.Cid)

	// Shutdown function.
	shutdown func()
}

// NewPieceServingManager creates a new PieceServingManager.
//
// @input - context, options.
//
// @output - piece serving manager, error.
func NewPieceServingManager(ctx context.Context, opts Opts) (*PieceServingManagerImpl, error) {
	log.Infof("Start piece serving manager...")
	// Open store.
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, opts.Path)
	if err != nil {
		log.Errorf("Fail to open mdlstore: %v", err.Error())
		return nil, err
	}
	defer func() {
		if err != nil {
			log.Infof("Fail to start piece serving manager, close ds...")
			err = ds.Shutdown(context.Background())
			if err != nil {
				log.Errorf("Fail to close ds after failing to start piece serving manager: %v", err.Error())
			}
		}
	}()
	// Do a ds cleanup.
	toRemove := make([]string, 0)
	txn, err := ds.NewTransaction(ctx, false)
	if err != nil {
		log.Errorf("Fail to start new transasction for ds cleanup: %v", err.Error())
		return nil, err
	}
	defer txn.Discard(context.Background())
	tempRes, err := txn.Query(ctx, mdlstore.Query{KeysOnly: true})
	if err != nil {
		log.Errorf("Fail to query ds: %v", err.Error())
		return nil, err
	}
	defer tempRes.Results.Close()
	for {
		e, ok := tempRes.Results.NextSync()
		if !ok {
			break
		}
		if e.Error != nil {
			err = e.Error
			log.Errorf("Fail to get next entry: %v", err.Error())
			return nil, err
		}
		if len(e.Key) > 1 {
			key := e.Key[1:]
			dim, keyHash, _, err := decDSKey(key)
			if err != nil {
				log.Errorf("Fail to decode ds key %v: %v", key, err.Error())
				return nil, err
			}
			if dim != 1 {
				continue
			}
			children, err := txn.GetChildren(ctx, keyHash)
			if err != nil {
				log.Errorf("Fail to get children for %v: %v", keyHash, err.Error())
				return nil, err
			}
			if len(children) == 0 {
				toRemove = append(toRemove, keyHash)
			}
		}
	}
	if len(toRemove) > 0 {
		for _, keyHash := range toRemove {
			err = txn.Delete(ctx, keyHash)
			if err != nil {
				log.Errorf("Fail to remove empty %v: %v", keyHash, err.Error())
				return nil, err
			}
		}
	}
	err = txn.Commit(ctx)
	if err != nil {
		log.Errorf("Fail to commit transaction: %v", err.Error())
		return nil, err
	}
	s := &PieceServingManagerImpl{
		ds:              ds,
		publishHookLock: sync.RWMutex{},
	}
	s.shutdown = func() {
		log.Infof("Stop datastore...")
		err := ds.Shutdown(context.Background())
		if err != nil {
			log.Errorf("Fail to stop datastore: %v", err.Error())
		}
	}
	return s, nil
}

// Shutdown safely shuts down the component.
func (s *PieceServingManagerImpl) Shutdown() {
	log.Infof("Start shutdown...")
	s.shutdown()
}

// RegisterPublishHook adds a hook function to call when a new serving presents.
//
// @input - hook function.
func (s *PieceServingManagerImpl) RegisterPublishFunc(hook func(cid.Cid)) {
	s.publishHookLock.Lock()
	defer s.publishHookLock.Unlock()
	s.publishHook = hook
}

// DeregisterPublishHook deregister the hook function.
func (s *PieceServingManagerImpl) DeregisterPublishHook() {
	s.publishHookLock.Lock()
	defer s.publishHookLock.Unlock()
	s.publishHook = nil
}

// Serve is used to serve a piece.
//
// @input - context, piece id, currency id, price-per-byte.
//
// @output - error.
func (s *PieceServingManagerImpl) Serve(ctx context.Context, id cid.Cid, currencyID byte, ppb *big.Int) error {
	log.Debugf("Start serve %v-%v with ppb to be %v", id, currencyID, ppb)
	if ppb.Cmp(big.NewInt(0)) < 0 {
		return fmt.Errorf("ppb cannot be neagtive, got %v", ppb)
	}
	keyHash := base58.Encode(id.Bytes())
	// Check if already served.
	release, err := s.ds.RLock(ctx, keyHash, currencyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", keyHash, currencyID, err.Error())
		return err
	}

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, keyHash, currencyID)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v: %v", keyHash, currencyID, err.Error())
		return err
	}
	if exists {
		release()
		return fmt.Errorf("%v-%v is already in serving", id.String(), currencyID)
	}
	release()

	release, err = s.ds.Lock(ctx, keyHash)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v: %v", keyHash, err.Error())
		return err
	}
	defer release()

	txn, err = s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	// Check again
	exists, err = txn.Has(ctx, keyHash, currencyID)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v: %v", keyHash, currencyID, err.Error())
		return err
	}
	if exists {
		return fmt.Errorf("%v-%v is already in serving", id.String(), currencyID)
	}
	err = txn.Put(ctx, ppb.Bytes(), keyHash, currencyID)
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v-%v: %v", ppb, keyHash, currencyID, err.Error())
		return err
	}
	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	// Do publish immediately
	go func() {
		log.Debugf("Publish serving %v immediately", id)
		s.publishHookLock.RLock()
		defer s.publishHookLock.RUnlock()
		if s.publishHook != nil {
			go s.publishHook(id)
		}
	}()
	return nil
}

// Stop is used to stop serving a piece.
//
// @input - context, piece id, currency id.
//
// @output - error.
func (s *PieceServingManagerImpl) Stop(ctx context.Context, id cid.Cid, currencyID byte) error {
	log.Debugf("Stop serving %v-%v", id, currencyID)
	keyHash := base58.Encode(id.Bytes())
	// Check if already served.
	release, err := s.ds.RLock(ctx, keyHash, currencyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", keyHash, currencyID, err.Error())
		return err
	}

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, keyHash, currencyID)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v: %v", keyHash, currencyID, err.Error())
		return err
	}
	if !exists {
		release()
		return fmt.Errorf("%v-%v not in serving", id.String(), currencyID)
	}
	release()

	release, err = s.ds.Lock(ctx, keyHash)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v: %v", keyHash, err.Error())
		return err
	}

	txn, err = s.ds.NewTransaction(ctx, false)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	// Check again
	exists, err = txn.Has(ctx, keyHash, currencyID)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v: %v", keyHash, currencyID, err.Error())
		return err
	}
	if !exists {
		release()
		return fmt.Errorf("%v-%v not in serving", id.String(), currencyID)
	}

	err = txn.Delete(ctx, keyHash, currencyID)
	if err != nil {
		release()
		log.Warnf("Fail to remove %v-%v: %v", keyHash, currencyID, err.Error())
		return err
	}
	// Check if now parent is empty
	children, err := txn.GetChildren(ctx, keyHash)
	if err != nil {
		release()
		log.Warnf("fail to get children for %v: %v", keyHash, err.Error())
		return err
	}

	err = txn.Commit(ctx)
	if err != nil {
		release()
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	// Succeed.
	release()

	// Check if parent is empty
	if len(children) == 0 {
		log.Debugf("Parent is empty, try to remove child")
		release, err := s.ds.Lock(ctx)
		if err != nil {
			log.Warnf("Fail to obtain write lock to remove empty %v: %v", keyHash, err.Error())
			return nil
		}
		defer release()

		txn, err := s.ds.NewTransaction(ctx, false)
		if err != nil {
			log.Warnf("Fail to start a new transaction to remove empty %v: %v", keyHash, err.Error())
			return nil
		}
		defer txn.Discard(context.Background())

		children, err = txn.GetChildren(ctx, keyHash)
		if err != nil {
			log.Warnf("Fail to get children for %v: %v", keyHash, err.Error())
			return nil
		}
		if len(children) == 0 {
			// Note is still empty.
			err = txn.Delete(ctx, keyHash)
			if err != nil {
				log.Warnf("Fail to remove empty %v: %v", keyHash, err.Error())
				return nil
			}
			err = txn.Commit(ctx)
			if err != nil {
				log.Warnf("Fail to commit transaction to remove empty %v: %v", keyHash, err.Error())
				return nil
			}
		}
		log.Debugf("Remove empty child done")
	}
	return nil
}

// Inspect is used to inspect a piece serving state.
//
// @input - context, piece id, currency id.
//
// @output - if served, price-per-byte, error.
func (s *PieceServingManagerImpl) Inspect(ctx context.Context, id cid.Cid, currencyID byte) (bool, *big.Int, error) {
	log.Debugf("Inspect %v-%v", id, currencyID)
	keyHash := base58.Encode(id.Bytes())
	// Check if served.
	release, err := s.ds.RLock(ctx, keyHash, currencyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", keyHash, currencyID, err.Error())
		return false, nil, err
	}
	defer release()

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return false, nil, err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, keyHash, currencyID)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v: %v", keyHash, currencyID, err.Error())
		return false, nil, err
	}
	if !exists {
		log.Debugf("Serving %v-%v does not exist", id, currencyID)
		return false, nil, nil
	}
	val, err := txn.Get(ctx, keyHash, currencyID)
	if err != nil {
		log.Warnf("Fail to get ds value for %v-%v: %v", keyHash, currencyID, err.Error())
		return false, nil, err
	}
	return true, big.NewInt(0).SetBytes(val), nil
}

// ListPieceIDs lists all pieces.
//
// @input - context.
//
// @output - piece id chan out, error chan out.
func (s *PieceServingManagerImpl) ListPieceIDs(ctx context.Context) (<-chan cid.Cid, <-chan error) {
	out := make(chan cid.Cid, 32)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Listing piece ids")
		defer func() {
			close(out)
			close(errChan)
		}()
		release, err := s.ds.RLock(ctx)
		if err != nil {
			errChan <- err
			log.Debugf("Fail to obtain read lock for root: %v", err.Error())
			return
		}

		txn, err := s.ds.NewTransaction(ctx, true)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to start a new transaction: %v", err.Error())
			return
		}
		defer txn.Discard(context.Background())

		children, err := txn.GetChildren(ctx)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to get children for root: %v", err.Error())
			return
		}
		release()

		for child := range children {
			mh, err := base58.Decode(child)
			if err != nil {
				errChan <- err
				log.Errorf("Fail to decode child %v, this should never happen: %v", child, err.Error())
				return
			}
			_, id, err := cid.CidFromBytes(mh)
			if err != nil {
				errChan <- err
				log.Errorf("Fail to create cid for %v, this should never happen: %v", child, err.Error())
				return
			}
			select {
			case out <- id:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errChan
}

// ListCurrencyIDs lists all currencies.
//
// @input - context, piece id.
//
// @output - currency chan out, error chan out.
func (s *PieceServingManagerImpl) ListCurrencyIDs(ctx context.Context, id cid.Cid) (<-chan byte, <-chan error) {
	keyHash := base58.Encode(id.Bytes())
	out := make(chan byte, 32)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Listing currency ids for %v", id)
		defer func() {
			close(out)
			close(errChan)
		}()
		release, err := s.ds.RLock(ctx, keyHash)
		if err != nil {
			errChan <- err
			log.Debugf("Fail to obtain read lock for %v: %v", keyHash, err.Error())
			return
		}

		txn, err := s.ds.NewTransaction(ctx, true)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to start new transaction: %v", err.Error())
			return
		}
		defer txn.Discard(context.Background())

		exists, err := txn.Has(ctx, keyHash)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to check if contains %v: %v", keyHash, err.Error())
			return
		}
		if !exists {
			release()
			return
		}

		children, err := txn.GetChildren(ctx, keyHash)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to get children for %v: %v", keyHash, err.Error())
			return
		}
		release()

		for child := range children {
			currencyID, err := strconv.Atoi(child)
			if err != nil {
				errChan <- err
				log.Errorf("Fail to convert child key %v to currency id: %v", child, err.Error())
				return
			}
			select {
			case out <- byte(currencyID):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errChan
}
