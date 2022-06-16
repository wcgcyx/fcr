package settlemgr

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
	"strconv"
	"time"

	"github.com/wcgcyx/fcr/mdlstore"
)

// SettlementManagerImpl is the implementation of the SettlementManager interface.
type SettlementManagerImpl struct {
	// Datastore
	ds mdlstore.MultiDimLockableStore

	// Shutdown function.
	shutdown func()
}

// NewSettlementManagerImpl creates a new SettlementManagerImpl.
//
// @input - context, options.
//
// @output - settlement manager, error.
func NewSettlementManagerImpl(ctx context.Context, opts Opts) (*SettlementManagerImpl, error) {
	log.Infof("Start settlement manager...")
	// Open store.
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, opts.Path)
	if err != nil {
		log.Errorf("Fail to open mdlstore: %v", err.Error())
		return nil, err
	}
	s := &SettlementManagerImpl{
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
func (s *SettlementManagerImpl) Shutdown() {
	log.Infof("Start shutdown...")
	s.shutdown()
}

// SetDefaultPolicy sets the default settlement policy.
//
// @input - context, currency id, settlement duration.
//
// @output - error.
func (s *SettlementManagerImpl) SetDefaultPolicy(ctx context.Context, currencyID byte, duration time.Duration) error {
	log.Debugf("Set default policy for %v to be %v", currencyID, duration)
	release, err := s.ds.Lock(ctx, currencyID, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
		return err
	}
	defer release()

	txn, err := s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	err = txn.Put(ctx, []byte(duration.String()), currencyID, DefaultPolicyID)
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v-%v: %v", []byte(duration.String()), currencyID, DefaultPolicyID, err.Error())
		return err
	}

	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// GetDefaultPolicy gets the default settlement policy.
//
// @input - context, currency id.
//
// @output - settlement duration, error.
func (s *SettlementManagerImpl) GetDefaultPolicy(ctx context.Context, currencyID byte) (time.Duration, error) {
	log.Debugf("Get default policy for %v", currencyID)
	release, err := s.ds.RLock(ctx, currencyID, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
		return 0, err
	}
	defer release()

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return 0, err
	}
	defer txn.Discard(context.Background())

	val, err := txn.Get(ctx, currencyID, DefaultPolicyID)
	if err != nil {
		log.Warnf("Fail to get ds value for %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
		return 0, err
	}

	duration, err := time.ParseDuration(string(val))
	if err != nil {
		log.Errorf("Fail to decode duration from %v, should never happen: %v", val, err.Error())
		return 0, err
	}
	return duration, nil
}

// RemoveDefaultPolicy removes the default settlement policy.
//
// @input - context, currency id.
//
// @output - error.
func (s *SettlementManagerImpl) RemoveDefaultPolicy(ctx context.Context, currencyID byte) error {
	log.Debugf("Remove default policy for %v", currencyID)
	// Check first.
	release, err := s.ds.RLock(ctx, currencyID, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
		return err
	}

	txn, err := s.ds.NewTransaction(ctx, false)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, DefaultPolicyID)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
		return err
	}
	if !exists {
		release()
		return fmt.Errorf("default policy does not exist for %v", currencyID)
	}
	release()

	// Check again.
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

	exists, err = txn.Has(ctx, currencyID, DefaultPolicyID)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
		return err
	}
	if !exists {
		return fmt.Errorf("default policy does not exist for %v", currencyID)
	}

	err = txn.Delete(ctx, currencyID, DefaultPolicyID)
	if err != nil {
		log.Warnf("Fail to remove %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
		return err
	}

	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// SetSenderPolicy sets a settlement policy for a given sender.
//
// @input - context, currency id, payment channel sender, settlement duration.
//
// @output - error.
func (s *SettlementManagerImpl) SetSenderPolicy(ctx context.Context, currencyID byte, fromAddr string, duration time.Duration) error {
	log.Debugf("Set sender policy for %v-%v to be %v", currencyID, fromAddr, duration)
	if fromAddr == DefaultPolicyID {
		return fmt.Errorf("fromAddr cannot be default policy id")
	}
	release, err := s.ds.Lock(ctx, currencyID, fromAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, fromAddr, err.Error())
		return err
	}
	defer release()

	txn, err := s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	err = txn.Put(ctx, []byte(duration.String()), currencyID, fromAddr)
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v-%v: %v", []byte(duration.String()), currencyID, fromAddr, err.Error())
		return err
	}

	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// GetSenderPolicy gets the policy for a given currency id and sender pair.
//
// @input - context, currency id, payment channel sender.
//
// @output - settlement duration, error.
func (s *SettlementManagerImpl) GetSenderPolicy(ctx context.Context, currencyID byte, fromAddr string) (time.Duration, error) {
	log.Debugf("Get sender policy for %v-%v", currencyID, fromAddr)
	if fromAddr == DefaultPolicyID {
		return 0, fmt.Errorf("fromAddr cannot be default policy id")
	}
	release1, err := s.ds.RLock(ctx, currencyID, fromAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, fromAddr, err.Error())
		return 0, err
	}
	defer release1()

	release2, err := s.ds.RLock(ctx, currencyID, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
		return 0, err
	}
	defer release2()

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return 0, err
	}
	defer txn.Discard(context.Background())

	var val []byte
	exists, err := txn.Has(ctx, currencyID, fromAddr)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, fromAddr, err.Error())
		return 0, err
	}
	if exists {
		val, err = txn.Get(ctx, currencyID, fromAddr)
		if err != nil {
			log.Warnf("Fail to get ds value for %v-%v: %v", currencyID, fromAddr, err.Error())
			return 0, err
		}
	} else {
		val, err = txn.Get(ctx, currencyID, DefaultPolicyID)
		if err != nil {
			log.Warnf("Fail to get ds value for %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
			return 0, err
		}
	}

	duration, err := time.ParseDuration(string(val))
	if err != nil {
		log.Errorf("Fail to decode duration from %v, should never happen: %v", val, err.Error())
		return 0, err
	}
	return duration, nil
}

// RemoveSenderPolicy removes a policy.
//
// @input - context, currency id, payment channel sender.
//
// @output - error.
func (s *SettlementManagerImpl) RemoveSenderPolicy(ctx context.Context, currencyID byte, fromAddr string) error {
	log.Debugf("Remove sender policy for %v-%v", currencyID, fromAddr)
	if fromAddr == DefaultPolicyID {
		return fmt.Errorf("fromAddr cannot be default policy id")
	}
	// Check first.
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
		release()
		return fmt.Errorf("policy does not exist for %v-%v", currencyID, fromAddr)
	}
	release()

	// Check again.
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

	exists, err = txn.Has(ctx, currencyID, fromAddr)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, fromAddr, err.Error())
		return err
	}
	if !exists {
		return fmt.Errorf("policy does not exist for %v-%v", currencyID, fromAddr)
	}

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

// ListCurrencyIDs lists all currency ids.
//
// @input - context.
//
// @output - currency id chan out, error chan out.
func (s *SettlementManagerImpl) ListCurrencyIDs(ctx context.Context) (<-chan byte, <-chan error) {
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
			log.Warnf("Fail to get children for root: %v", err.Error())
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

// ListSenders lists all senders with a policy set.
//
// @input - context, currency id.
//
// @output - sender chan out (could be default policy id or sender addr), error chan out.
func (s *SettlementManagerImpl) ListSenders(ctx context.Context, currencyID byte) (<-chan string, <-chan error) {
	out := make(chan string, 128)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Listing senders for %v", currencyID)
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
