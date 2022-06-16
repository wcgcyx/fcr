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
	"strconv"

	"github.com/wcgcyx/fcr/mdlstore"
)

// SubStoreImpl is the implementation of the SubStore interface.
type SubStoreImpl struct {
	// Datastore
	ds mdlstore.MultiDimLockableStore

	// Shutdown function.
	shutdown func()
}

// NewSubStoreImpl creates a new SubStoreImpl.
//
// @input - context, options.
//
// @output - store, error.
func NewSubStoreImpl(ctx context.Context, opts Opts) (*SubStoreImpl, error) {
	log.Infof("Start sub store...")
	// Open store.
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, opts.Path)
	if err != nil {
		log.Errorf("Fail to open mdlstore: %v", err.Error())
		return nil, err
	}
	s := &SubStoreImpl{
		ds: ds,
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
func (s *SubStoreImpl) Shutdown() {
	log.Infof("Start shutdown...")
	s.shutdown()
}

// AddSubscriber is used to add a subscriber.
//
// @input - context, currency id, from address.
//
// @output - error.
func (s *SubStoreImpl) AddSubscriber(ctx context.Context, currencyID byte, fromAddr string) error {
	log.Debugf("Add subscriber of %v-%v", currencyID, fromAddr)
	// First check if subscriber exists.
	release, err := s.ds.RLock(ctx, currencyID, fromAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, fromAddr, err.Error())
		return err
	}

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, fromAddr)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, fromAddr, err.Error())
		return err
	}
	if exists {
		// Fail silently if there is already a subscriber exists.
		release()
		log.Debugf("%v is already a subscriber", currencyID, fromAddr)
		return nil
	}
	release()

	release, err = s.ds.Lock(ctx, currencyID, fromAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, fromAddr, err.Error())
		return err
	}
	defer release()

	txn, err = s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	// Check again
	exists, err = txn.Has(ctx, currencyID, fromAddr)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, fromAddr, err.Error())
		return err
	}
	if exists {
		// Fail silently if there is already a direct link presents.
		log.Debugf("%v is already a subscriber", currencyID, fromAddr)
		return nil
	}

	// Insert
	err = txn.Put(ctx, []byte{1}, currencyID, fromAddr)
	if err != nil {
		log.Warnf("Fail to put ds value [1] for %v-%v: %v", currencyID, fromAddr, err.Error())
		return err
	}

	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// RemoveSubscriber is used to remove a subscriber.
//
// @input - context, currency id, from address.
//
// @output - error.
func (s *SubStoreImpl) RemoveSubscriber(ctx context.Context, currencyID byte, fromAddr string) error {
	log.Debugf("Remove subscriber of %v-%v", currencyID, fromAddr)
	// First check if subscriber exists.
	release, err := s.ds.RLock(ctx, currencyID, fromAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, fromAddr, err.Error())
		return err
	}

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, fromAddr)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, fromAddr, err.Error())
		return err
	}
	if !exists {
		// Fail silently if there is no matching subscriber exists.
		release()
		log.Debugf("%v is not an existing subscriber", currencyID, fromAddr)
		return nil
	}
	release()

	release, err = s.ds.Lock(ctx, currencyID)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v: %v", currencyID, err.Error())
		return err
	}
	defer release()

	txn, err = s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	// Check again
	exists, err = txn.Has(ctx, currencyID, fromAddr)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, fromAddr, err.Error())
		return err
	}
	if !exists {
		// Fail silently if there is no matching subscriber exists.
		log.Debugf("%v is not an existing subscriber", currencyID, fromAddr)
		return nil
	}

	// Delete
	err = txn.Delete(ctx, currencyID, fromAddr)
	if err != nil {
		log.Warnf("Fail to remove %v-%v: %v", currencyID, fromAddr, err.Error())
		return err
	}

	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// ListCurrencyIDs is used to list all currencies.
//
// @input - context.
//
// @output - currency id chan out, error chan out.
func (s *SubStoreImpl) ListCurrencyIDs(ctx context.Context) (<-chan byte, <-chan error) {
	out := make(chan byte, 32)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Listing currency ids")
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
			log.Warnf("Fail to start new transaction: %v", err.Error())
			return
		}
		defer txn.Discard(context.Background())

		children, err := txn.GetChildren(ctx)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to get children for root", err.Error())
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

// ListSubscribers is used to list all subscribers.
//
// @input - context, currency id.
//
// @output - subscriber chan out, error chan out.
func (s *SubStoreImpl) ListSubscribers(ctx context.Context, currencyID byte) (<-chan string, <-chan error) {
	out := make(chan string, 128)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Listing subscribers for %v", currencyID)
		defer func() {
			close(out)
			close(errChan)
		}()
		release, err := s.ds.RLock(ctx, currencyID)
		if err != nil {
			errChan <- err
			log.Debugf("Fail to obtain read lock for %v: %v", currencyID, err.Error())
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

		exists, err := txn.Has(ctx, currencyID)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to check if contains %v: %v", currencyID, err.Error())
			return
		}
		if !exists {
			release()
			return
		}

		children, err := txn.GetChildren(ctx, currencyID)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to get children for %v: %v", currencyID, err.Error())
			return
		}
		release()

		for child := range children {
			select {
			case out <- child:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errChan
}
