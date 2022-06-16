package renewmgr

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

// RenewManagerImpl is the implementation of the RenewManager interface.
type RenewManagerImpl struct {
	// Datastore
	ds mdlstore.MultiDimLockableStore

	// Shutdown function.
	shutdown func()
}

// NewRenewManagerImpl creates a new RenewManagerImpl.
//
// @input - context, options.
//
// @output - renew manager, error.
func NewRenewManagerImpl(ctx context.Context, opts Opts) (*RenewManagerImpl, error) {
	log.Infof("Start renew manager...")
	// Open store
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, opts.Path)
	if err != nil {
		log.Errorf("Fail to open mdlstore: %v", err.Error())
		return nil, err
	}
	defer func() {
		if err != nil {
			log.Infof("Fail to start renew manager, close ds...")
			err = ds.Shutdown(context.Background())
			if err != nil {
				log.Errorf("Fail to close ds after failing to start renew manager: %v", err.Error())
			}
		}
	}()
	// Do a ds cleanup.
	txn, err := ds.NewTransaction(ctx, false)
	if err != nil {
		log.Errorf("Fail to start new transasction for ds cleanup: %v", err.Error())
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
		children2, err := txn.GetChildren(ctx, currencyID)
		if err != nil {
			log.Errorf("Fail to get children for %v: %v", currencyID, err.Error())
			return nil, err
		}
		for fromAddr := range children2 {
			if fromAddr == DefaultPolicyID {
				// Skip default policy
				continue
			}
			children3, err := txn.GetChildren(ctx, currencyID, fromAddr)
			if err != nil {
				log.Errorf("Fail to get children for %v-%v: %v", currencyID, fromAddr, err.Error())
				return nil, err
			}
			if len(children3) == 0 {
				// Empty, need to remove it.
				err = txn.Delete(ctx, currencyID, fromAddr)
				if err != nil {
					log.Errorf("Fail to remove empty %v-%v: %v", currencyID, fromAddr, err.Error())
					return nil, err
				}
			}
		}
	}
	err = txn.Commit(ctx)
	if err != nil {
		log.Errorf("Fail to commit transaction: %v", err.Error())
		return nil, err
	}
	s := &RenewManagerImpl{
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
func (s *RenewManagerImpl) Shutdown() {
	log.Infof("Start shutdown...")
	s.shutdown()
}

// SetDefaultPolicy sets the default renew policy.
//
// @input - context, currency id, renew duration.
//
// @output - error.
func (s *RenewManagerImpl) SetDefaultPolicy(ctx context.Context, currencyID byte, duration time.Duration) error {
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

// GetDefaultPolicy gets the default renew policy.
//
// @input - context, currency id.
//
// @output - renew duration, error.
func (s *RenewManagerImpl) GetDefaultPolicy(ctx context.Context, currencyID byte) (time.Duration, error) {
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

// RemoveDefaultPolicy removes the default renew policy.
//
// @input - context, currency id.
//
// @output - error.
func (s *RenewManagerImpl) RemoveDefaultPolicy(ctx context.Context, currencyID byte) error {
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

// SetSenderPolicy sets a renew policy for a given sender.
//
// @input - context, currency id, payment channel sender, renew duration.
//
// @output - error.
func (s *RenewManagerImpl) SetSenderPolicy(ctx context.Context, currencyID byte, fromAddr string, duration time.Duration) error {
	log.Debugf("Set sender policy for %v-%v to be %v", currencyID, fromAddr, duration)
	if fromAddr == DefaultPolicyID {
		return fmt.Errorf("fromAddr cannot be default policy id")
	}
	release, err := s.ds.Lock(ctx, currencyID, fromAddr, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v: %v", currencyID, fromAddr, DefaultPolicyID, err.Error())
		return err
	}
	defer release()

	txn, err := s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	err = txn.Put(ctx, []byte(duration.String()), currencyID, fromAddr, DefaultPolicyID)
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v-%v-%v: %v", []byte(duration.String()), currencyID, fromAddr, DefaultPolicyID, err.Error())
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
// @output - renew duration, error.
func (s *RenewManagerImpl) GetSenderPolicy(ctx context.Context, currencyID byte, fromAddr string) (time.Duration, error) {
	log.Debugf("Get sender policy for %v-%v", currencyID, fromAddr)
	if fromAddr == DefaultPolicyID {
		return 0, fmt.Errorf("fromAddr cannot be default policy id")
	}
	release1, err := s.ds.RLock(ctx, currencyID, fromAddr, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, fromAddr, DefaultPolicyID, err.Error())
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

	var val []byte
	exists, err := txn.Has(ctx, currencyID, fromAddr, DefaultPolicyID)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, fromAddr, DefaultPolicyID, err.Error())
		return 0, err
	}
	if exists {
		val, err = txn.Get(ctx, currencyID, fromAddr, DefaultPolicyID)
		if err != nil {
			log.Warnf("Fail to get ds value for %v-%v-%v: %v", currencyID, fromAddr, DefaultPolicyID, err.Error())
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
func (s *RenewManagerImpl) RemoveSenderPolicy(ctx context.Context, currencyID byte, fromAddr string) error {
	log.Debugf("Remove sender policy for %v-%v", currencyID, fromAddr)
	if fromAddr == DefaultPolicyID {
		return fmt.Errorf("fromAddr cannot be default policy id")
	}
	// Check first.
	release, err := s.ds.RLock(ctx, currencyID, fromAddr, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, fromAddr, DefaultPolicyID, err.Error())
		return err
	}

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, fromAddr, DefaultPolicyID)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, fromAddr, DefaultPolicyID, err.Error())
		return err
	}
	if !exists {
		release()
		return fmt.Errorf("policy does not exist for %v-%v", currencyID, fromAddr)
	}
	release()

	release, err = s.ds.Lock(ctx, currencyID, fromAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, fromAddr, err.Error())
		return err
	}

	// Read again.
	txn, err = s.ds.NewTransaction(ctx, false)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err = txn.Has(ctx, currencyID, fromAddr, DefaultPolicyID)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, fromAddr, DefaultPolicyID, err.Error())
		return err
	}
	if !exists {
		release()
		return fmt.Errorf("policy does not exist for %v-%v", currencyID, fromAddr)
	}

	err = txn.Delete(ctx, currencyID, fromAddr, DefaultPolicyID)
	if err != nil {
		release()
		log.Warnf("Fail to remove %v-%v-%v: %v", currencyID, fromAddr, DefaultPolicyID, err.Error())
		return err
	}

	// Check if parent is empty
	children, err := txn.GetChildren(ctx, currencyID, fromAddr)
	if err != nil {
		release()
		log.Warnf("Fail to get children for %v-%v: %v", currencyID, fromAddr, err.Error())
		return err
	}

	err = txn.Commit(ctx)
	if err != nil {
		release()
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	// Succeed
	release()

	// Check if parent is empty
	if len(children) == 0 {
		release, err = s.ds.Lock(ctx, currencyID)
		if err != nil {
			log.Warnf("Fail to obtain write access to remove empty %v-%v: %v", currencyID, fromAddr, err.Error())
			return nil
		}
		defer release()
		txn, err := s.ds.NewTransaction(ctx, false)
		if err != nil {
			log.Warnf("Fail to start new transaction to remove empty %v-%v: %v", currencyID, fromAddr, err.Error())
			return nil
		}
		defer txn.Discard(context.Background())
		children, err = txn.GetChildren(ctx, currencyID, fromAddr)
		if err != nil {
			log.Warnf("Fail to get children for to be removed empty %v-%v: %v", currencyID, fromAddr, err.Error())
			return nil
		}
		if len(children) == 0 {
			// Note is still empty.
			err = txn.Delete(ctx, currencyID, fromAddr)
			if err != nil {
				log.Warnf("Fail to remove empty %v-%v: %v", currencyID, fromAddr, err.Error())
				return nil
			}
			err = txn.Commit(ctx)
			if err != nil {
				log.Warnf("Fail to commit transaction to remove empty %v-%v: %v", currencyID, fromAddr, err.Error())
			}
		}
	}
	return nil
}

// SetPaychPolicy sets a renew policy for a given paych.
//
// @input - context, currency id, payment channel sender, paych addr, renew duration.
//
// @output - error.
func (s *RenewManagerImpl) SetPaychPolicy(ctx context.Context, currencyID byte, fromAddr string, chAddr string, duration time.Duration) error {
	log.Debugf("Set paych policy for %v-%v-%v to be %v", currencyID, fromAddr, chAddr, duration)
	if fromAddr == DefaultPolicyID || chAddr == DefaultPolicyID {
		return fmt.Errorf("fromAddr or chAddr cannot be default policy id")
	}
	release, err := s.ds.Lock(ctx, currencyID, fromAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v: %v", currencyID, fromAddr, chAddr, err.Error())
		return err
	}
	defer release()

	txn, err := s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	err = txn.Put(ctx, []byte(duration.String()), currencyID, fromAddr, chAddr)
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v-%v-%v: %v", []byte(duration.String()), currencyID, fromAddr, chAddr, err.Error())
		return err
	}

	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// GetPaychPolicy gets the policy for a given currency id and paych pair.
//
// @input - context, currency id, payment channel sender, paych addr.
//
// @output - renew duration, error.
func (s *RenewManagerImpl) GetPaychPolicy(ctx context.Context, currencyID byte, fromAddr string, chAddr string) (time.Duration, error) {
	log.Debugf("Get paych policy for %v-%v-%v", currencyID, fromAddr, chAddr)
	if fromAddr == DefaultPolicyID || chAddr == DefaultPolicyID {
		return 0, fmt.Errorf("fromAddr or chAddr cannot be default policy id")
	}
	release1, err := s.ds.RLock(ctx, currencyID, fromAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, fromAddr, chAddr, err.Error())
		return 0, err
	}
	defer release1()

	release2, err := s.ds.RLock(ctx, currencyID, fromAddr, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, fromAddr, DefaultPolicyID, err.Error())
		return 0, err
	}
	defer release2()

	release3, err := s.ds.RLock(ctx, currencyID, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
		return 0, err
	}
	defer release3()

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return 0, err
	}

	var val []byte
	exists, err := txn.Has(ctx, currencyID, fromAddr, chAddr)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, fromAddr, chAddr, err.Error())
		return 0, err
	}
	if exists {
		val, err = txn.Get(ctx, currencyID, fromAddr, chAddr)
		if err != nil {
			log.Warnf("Fail to get ds value for %v-%v-%v: %v", currencyID, fromAddr, chAddr, err.Error())
			return 0, err
		}
	} else {
		exists, err := txn.Has(ctx, currencyID, fromAddr, DefaultPolicyID)
		if err != nil {
			log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, fromAddr, DefaultPolicyID, err.Error())
			return 0, err
		}
		if exists {
			val, err = txn.Get(ctx, currencyID, fromAddr, DefaultPolicyID)
			if err != nil {
				log.Warnf("Fail to get ds value for %v-%v-%v: %v", currencyID, fromAddr, DefaultPolicyID, err.Error())
				return 0, err
			}
		} else {
			val, err = txn.Get(ctx, currencyID, DefaultPolicyID)
			if err != nil {
				log.Warnf("Fail to get ds value for %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
				return 0, err
			}
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
// @input - context, currency id, payment channel sender, paych addr.
//
// @output - error.
func (s *RenewManagerImpl) RemovePaychPolicy(ctx context.Context, currencyID byte, fromAddr string, chAddr string) error {
	log.Debugf("Remove paych policy for %v-%v-%v", currencyID, fromAddr, chAddr)
	if fromAddr == DefaultPolicyID || chAddr == DefaultPolicyID {
		return fmt.Errorf("fromAddr or chAddr cannot be default policy id")
	}
	// Check first.
	release, err := s.ds.RLock(ctx, currencyID, fromAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, fromAddr, chAddr, err.Error())
		return err
	}

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, fromAddr, chAddr)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, fromAddr, chAddr, err.Error())
		return err
	}
	if !exists {
		release()
		return fmt.Errorf("policy does not exist for %v-%v-%v", currencyID, fromAddr, chAddr)
	}
	release()

	release, err = s.ds.Lock(ctx, currencyID, fromAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, fromAddr, err.Error())
		return err
	}

	// Read again.
	txn, err = s.ds.NewTransaction(ctx, false)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err = txn.Has(ctx, currencyID, fromAddr, chAddr)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, fromAddr, chAddr, err.Error())
		return err
	}
	if !exists {
		release()
		return fmt.Errorf("policy does not exist for %v-%v-%v", currencyID, fromAddr, chAddr)
	}

	err = txn.Delete(ctx, currencyID, fromAddr, chAddr)
	if err != nil {
		release()
		log.Warnf("Fail to remove %v-%v-%v: %v", currencyID, fromAddr, chAddr, err.Error())
		return err
	}

	// Check if parent is empty
	children, err := txn.GetChildren(ctx, currencyID, fromAddr)
	if err != nil {
		release()
		log.Warnf("Fail to get children for %v-%v: %v", currencyID, fromAddr, err.Error())
		return err
	}

	err = txn.Commit(ctx)
	if err != nil {
		release()
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	// Succeed
	release()

	// Check if parent is empty
	if len(children) == 0 {
		release, err = s.ds.Lock(ctx, currencyID)
		if err != nil {
			log.Warnf("Fail to obtain write lock to remove empty %v-%v: %v", currencyID, fromAddr, err.Error())
			return nil
		}
		defer release()

		txn, err := s.ds.NewTransaction(ctx, false)
		if err != nil {
			log.Warnf("Fail to start a new transaction to remove empty %v-%v: %v", currencyID, fromAddr, err.Error())
			return nil
		}
		defer txn.Discard(context.Background())

		children, err = txn.GetChildren(ctx, currencyID, fromAddr)
		if err != nil {
			log.Warnf("Fail to get children to remove empty %v-%v: %v", currencyID, fromAddr, err.Error())
			return nil
		}
		if len(children) == 0 {
			// Note is still empty.
			err = txn.Delete(ctx, currencyID, fromAddr)
			if err != nil {
				log.Warnf("Fail to remove empty %v-%v: %v", currencyID, fromAddr, err.Error())
				return nil
			}
			err = txn.Commit(ctx)
			if err != nil {
				log.Warnf("Fail to commit transaction to remove empty %v-%v: %v", currencyID, fromAddr, err.Error())
			}
		}
	}
	return nil
}

// ListCurrencyIDs lists all currency ids.
//
// @input - context.
//
// @output - currency id chan out, error chan out.
func (s *RenewManagerImpl) ListCurrencyIDs(ctx context.Context) (<-chan byte, <-chan error) {
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
func (s *RenewManagerImpl) ListSenders(ctx context.Context, currencyID byte) (<-chan string, <-chan error) {
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

// ListSenders lists all paychs with a policy set.
//
// @input - context, currency id, sender addr.
//
// @output - paych addr chan out (could be default policy id or paych addr), error chan out.
func (s *RenewManagerImpl) ListPaychs(ctx context.Context, currencyID byte, fromAddr string) (<-chan string, <-chan error) {
	out := make(chan string, 128)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Listing paychs for %v-%v", currencyID, fromAddr)
		defer func() {
			close(out)
			close(errChan)
		}()
		release, err := s.ds.RLock(ctx, currencyID, fromAddr)
		if err != nil {
			errChan <- err
			log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, fromAddr, err.Error())
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

		exists, err := txn.Has(ctx, currencyID, fromAddr)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to check if contains %v-%v: %v", currencyID, fromAddr, err.Error())
			return
		}
		if !exists {
			release()
			return
		}
		children, err := txn.GetChildren(ctx, currencyID, fromAddr)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to get children for %v-%v: %v", currencyID, fromAddr, err.Error())
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
