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
	"fmt"
	"math/big"
	"strconv"

	"github.com/wcgcyx/fcr/mdlstore"
)

// PaychServingManagerImpl is the implementation of the PaychServingManager interface.
type PaychServingManagerImpl struct {
	// Datastore
	ds mdlstore.MultiDimLockableStore

	// Cache, mainly used for payment manager's query.
	cache map[byte]map[string]map[string]cachedServing

	// Shutdown function.
	shutdown func()
}

// NewPaychServingManagerImpl creates a new PaychServingManagerImpl.
//
// @input - context, options.
//
// @output - paych serving manager, error.
func NewPaychServingManagerImpl(ctx context.Context, opts Opts) (*PaychServingManagerImpl, error) {
	log.Infof("Start paych serving manager...")
	// Open store.
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, opts.Path)
	if err != nil {
		log.Errorf("Fail to open mdlstore: %v", err.Error())
		return nil, err
	}
	defer func() {
		if err != nil {
			log.Infof("Fail to start paych serving manager, close ds...")
			err = ds.Shutdown(context.Background())
			if err != nil {
				log.Errorf("Fail to close ds after failing to start paych serving manager: %v", err.Error())
			}
		}
	}()
	// Load data to cache
	cache := make(map[byte]map[string]map[string]cachedServing)
	txn, err := ds.NewTransaction(ctx, false)
	if err != nil {
		log.Errorf("Fail to start new transaction for loading cache: %v", err.Error())
		return nil, err
	}
	defer txn.Discard(context.Background())
	children1, err := txn.GetChildren(ctx)
	if err != nil {
		log.Errorf("Fail to get children for root: %v", err.Error())
		return nil, err
	}
	for child1 := range children1 {
		currency, err := strconv.Atoi(child1)
		if err != nil {
			log.Errorf("Fail to convert child key %v to currency id: %v", child1, err.Error())
			return nil, err
		}
		currencyID := byte(currency)
		cache[currencyID] = make(map[string]map[string]cachedServing)
		children2, err := txn.GetChildren(ctx, currencyID)
		if err != nil {
			log.Errorf("Fail to get children for %v: %v", currencyID, err.Error())
			return nil, err
		}
		for toAddr := range children2 {
			cache[currencyID][toAddr] = make(map[string]cachedServing)
			children3, err := txn.GetChildren(ctx, currencyID, toAddr)
			if err != nil {
				log.Errorf("Fail to get children for %v-%v: %v", currencyID, toAddr, err.Error())
				return nil, err
			}
			if len(children3) == 0 {
				err = txn.Delete(ctx, currencyID, toAddr)
				if err != nil {
					log.Errorf("Fail to remove empty %v-%v: %v", currencyID, toAddr, err.Error())
					return nil, err
				}
				delete(cache[currencyID], toAddr)
			}
			for chAddr := range children3 {
				val, err := txn.Get(ctx, currencyID, toAddr, chAddr)
				if err != nil {
					log.Errorf("Fail to read ds value for %v-%v-%v: %v", currencyID, toAddr, chAddr, err.Error())
					return nil, err
				}
				ppp, period, err := decDSVal(val)
				if err != nil {
					log.Error("Fail to decode ds value of %v: %v", val, err.Error())
					return nil, err
				}
				cache[currencyID][toAddr][chAddr] = cachedServing{
					ppp:    ppp,
					period: period,
				}
			}
		}
	}
	err = txn.Commit(ctx)
	if err != nil {
		log.Errorf("Fail to commit transaction: %v", err.Error())
		return nil, err
	}
	s := &PaychServingManagerImpl{
		ds:    ds,
		cache: cache,
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
func (s *PaychServingManagerImpl) Shutdown() {
	log.Infof("Start shutdown...")
	s.shutdown()
}

// Serve is used to serve a payment channel.
//
// @input - context, currency id, recipient address, channel address, price-per-period, period.
//
// @output - error.
func (s *PaychServingManagerImpl) Serve(ctx context.Context, currencyID byte, toAddr string, chAddr string, ppp *big.Int, period *big.Int) error {
	log.Debugf("Start serving %v-%v-%v with %v-%v", currencyID, toAddr, chAddr, ppp, period)
	if !((ppp.Cmp(big.NewInt(0)) == 0 && period.Cmp(big.NewInt(0)) == 0) || (ppp.Cmp(big.NewInt(0)) > 0 && period.Cmp(big.NewInt(0)) > 0)) {
		return fmt.Errorf("ppp and period must be either both 0 or both positive, got ppp %v, period %v", ppp, period)
	}
	// Check if already served.
	release, err := s.ds.RLock(ctx, currencyID, toAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, toAddr, chAddr, err.Error())
		return err
	}
	_, ok := s.cache[currencyID]
	if ok {
		_, ok = s.cache[currencyID][toAddr]
		if ok {
			_, ok = s.cache[currencyID][toAddr][chAddr]
			if ok {
				release()
				return fmt.Errorf("%v-%v-%v is already in serving", currencyID, toAddr, chAddr)
			}
		}
	}
	release()

	release, err = s.ds.Lock(ctx, currencyID, toAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v: %v", currencyID, toAddr, chAddr, err.Error())
		return err
	}
	defer release()

	// Check again
	_, ok = s.cache[currencyID]
	if ok {
		_, ok = s.cache[currencyID][toAddr]
		if ok {
			_, ok = s.cache[currencyID][toAddr][chAddr]
			if ok {
				return fmt.Errorf("%v-%v-%v is already in serving", currencyID, toAddr, chAddr)
			}
		}
	}

	// Write to ds
	txn, err := s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	val, err := encDSVal(ppp, period)
	if err != nil {
		log.Errorf("Fail to encode ds value for %v %v, this should never happen: %v", ppp, period, err.Error())
		return err
	}

	err = txn.Put(ctx, val, currencyID, toAddr, chAddr)
	if err != nil {
		log.Warnf("Fail to put value %v for %v-%v-%v: %v", val, currencyID, toAddr, chAddr, err.Error())
		return err
	}

	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}

	// Update cache
	_, ok = s.cache[currencyID]
	if !ok {
		s.cache[currencyID] = make(map[string]map[string]cachedServing)
	}
	_, ok = s.cache[currencyID][toAddr]
	if !ok {
		s.cache[currencyID][toAddr] = make(map[string]cachedServing)
	}
	s.cache[currencyID][toAddr][chAddr] = cachedServing{
		ppp:    big.NewInt(0).Set(ppp),
		period: big.NewInt(0).Set(period),
	}
	return nil
}

// Stop is used to stop serving a payment channel.
//
// @input - context, currency id, recipient address, channel address.
//
// @output - error.
func (s *PaychServingManagerImpl) Stop(ctx context.Context, currencyID byte, toAddr string, chAddr string) error {
	log.Debugf("Stop serving %v-%v-%v", currencyID, toAddr, chAddr)
	// Check if served.
	release, err := s.ds.RLock(ctx, currencyID, toAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, toAddr, chAddr, err.Error())
		return err
	}

	_, ok := s.cache[currencyID]
	if !ok {
		release()
		return fmt.Errorf("%v not in serving", currencyID)
	}
	_, ok = s.cache[currencyID][toAddr]
	if !ok {
		release()
		return fmt.Errorf("%v-%v not in serving", currencyID, toAddr)
	}
	_, ok = s.cache[currencyID][toAddr][chAddr]
	if !ok {
		release()
		return fmt.Errorf("%v-%v-%v not in serving", currencyID, toAddr, chAddr)
	}
	release()

	release, err = s.ds.Lock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}

	// Check again
	_, ok = s.cache[currencyID]
	if !ok {
		release()
		return fmt.Errorf("%v not in serving", currencyID)
	}
	_, ok = s.cache[currencyID][toAddr]
	if !ok {
		release()
		return fmt.Errorf("%v-%v not in serving", currencyID, toAddr)
	}
	_, ok = s.cache[currencyID][toAddr][chAddr]
	if !ok {
		release()
		return fmt.Errorf("%v-%v-%v not in serving", currencyID, toAddr, chAddr)
	}

	// Remove from ds
	txn, err := s.ds.NewTransaction(ctx, false)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	err = txn.Delete(ctx, currencyID, toAddr, chAddr)
	if err != nil {
		release()
		log.Warnf("Fail to remove %v-%v-%v: %v", currencyID, toAddr, chAddr, err.Error())
		return err
	}

	err = txn.Commit(ctx)
	if err != nil {
		release()
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}

	// Update cache
	delete(s.cache[currencyID][toAddr], chAddr)
	length := len(s.cache[currencyID][toAddr])
	release()

	if length == 0 {
		// Empty, attempt clean
		release, err = s.ds.Lock(ctx, currencyID)
		if err != nil {
			log.Warnf("Fail to obtain write lock to remove empty %v-%v: %v", currencyID, toAddr, err.Error())
			return nil
		}
		defer release()

		txn, err := s.ds.NewTransaction(ctx, false)
		if err != nil {
			log.Warnf("Fail to start a new transaction to remove empty %v-%v: %v", currencyID, toAddr, err.Error())
			return nil
		}
		defer txn.Discard(context.Background())

		length = len(s.cache[currencyID][toAddr])
		if length == 0 {
			// Note is still empty.
			// Cache can still be cleaned no matter what.
			delete(s.cache[currencyID], toAddr)
			err = txn.Delete(ctx, currencyID, toAddr)
			if err != nil {
				log.Warnf("Fail to remove empty %v-%v: %v", currencyID, toAddr, err.Error())
				return nil
			}
			err = txn.Commit(ctx)
			if err != nil {
				log.Warnf("Fail to commit transaction to remove empty %v-%v: %v", currencyID, toAddr, err.Error())
			}
		}
	}
	return nil
}

// Inspect is used to inspect a payment channel serving state.
//
// @input - context, currency id, recipient address, channel address.
//
// @output - if served, price-per-period, period, error.
func (s *PaychServingManagerImpl) Inspect(ctx context.Context, currencyID byte, toAddr string, chAddr string) (bool, *big.Int, *big.Int, error) {
	log.Debugf("Inspect %v-%v-%v", currencyID, toAddr, chAddr)
	// Do a check through cache.
	release, err := s.ds.RLock(ctx, currencyID, toAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, toAddr, chAddr, err.Error())
		return false, nil, nil, err
	}
	defer release()

	_, ok := s.cache[currencyID]
	if ok {
		_, ok = s.cache[currencyID][toAddr]
		if ok {
			_, ok = s.cache[currencyID][toAddr][chAddr]
			if ok {
				return true, big.NewInt(0).Set(s.cache[currencyID][toAddr][chAddr].ppp), big.NewInt(0).Set(s.cache[currencyID][toAddr][chAddr].period), nil
			}
		}
	}
	return false, nil, nil, nil
}

// ListCurrencyIDs lists all currencies.
//
// @input - context.
//
// @output - currency chan out, error chan out.
func (s *PaychServingManagerImpl) ListCurrencyIDs(ctx context.Context) (<-chan byte, <-chan error) {
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
		defer release()

		for currencyID := range s.cache {
			select {
			case out <- currencyID:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errChan
}

// ListRecipients lists all recipients.
//
// @input - context, currency id.
//
// @output - recipient chan out, error chan out.
func (s *PaychServingManagerImpl) ListRecipients(ctx context.Context, currencyID byte) (<-chan string, <-chan error) {
	out := make(chan string, 128)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Listing recipients for %v", currencyID)
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
		defer release()

		_, ok := s.cache[currencyID]
		if !ok {
			return
		}
		for toAddr := range s.cache[currencyID] {
			select {
			case out <- toAddr:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errChan
}

// ListServings lists all servings.
//
// @input - context, currency id, recipient address.
//
// @output - paych chan out, error chan out.
func (s *PaychServingManagerImpl) ListServings(ctx context.Context, currencyID byte, toAddr string) (<-chan string, <-chan error) {
	out := make(chan string, 128)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Listing servings for %v-%v", currencyID, toAddr)
		defer func() {
			close(out)
			close(errChan)
		}()
		release, err := s.ds.RLock(ctx, currencyID, toAddr)
		if err != nil {
			errChan <- err
			log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, toAddr, err.Error())
			return
		}
		defer release()

		_, ok := s.cache[currencyID]
		if !ok {
			return
		}
		_, ok = s.cache[currencyID][toAddr]
		if !ok {
			return
		}
		for chAddr := range s.cache[currencyID][toAddr] {
			select {
			case out <- chAddr:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errChan
}
