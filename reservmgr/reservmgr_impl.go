package reservmgr

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

// ReservationManagerImpl is the implementation of the ReservationManager interface.
type ReservationManagerImpl struct {
	// Datastore
	ds mdlstore.MultiDimLockableStore

	// Shutdown function.
	shutdown func()
}

// NewReservationManagerImpl creates a new ReservationManagerImpl.
//
// @input - context, options.
//
// @output - reservation manager, error.
func NewReservationManagerImpl(ctx context.Context, opts Opts) (*ReservationManagerImpl, error) {
	log.Infof("Start reservation manager...")
	// Open store
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, opts.Path)
	if err != nil {
		log.Errorf("Fail to open mdlstore: %v", err.Error())
		return nil, err
	}
	defer func() {
		if err != nil {
			log.Infof("Fail to start reservation manager, close ds...")
			err = ds.Shutdown(context.Background())
			if err != nil {
				log.Errorf("Fail to close ds after failing to start reservation manager: %v", err.Error())
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
		for chAddr := range children2 {
			if chAddr == DefaultPolicyID {
				// Skip default policy
				continue
			}
			children3, err := txn.GetChildren(ctx, currencyID, chAddr)
			if err != nil {
				log.Errorf("Fail to get children for %v-%v: %v", currencyID, chAddr, err.Error())
				return nil, err
			}
			if len(children3) == 0 {
				// Empty, need to remove it.
				err = txn.Delete(ctx, currencyID, chAddr)
				if err != nil {
					log.Errorf("Fail to remove empty %v-%v: %v", currencyID, chAddr, err.Error())
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
	s := &ReservationManagerImpl{
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
func (s *ReservationManagerImpl) Shutdown() {
	log.Infof("Start shutdown...")
	s.shutdown()
}

// SetDefaultPolicy sets the default reservation policy.
//
// @input - context, currency id, if unlimited reservation, maximum single reservation amount.
//
// @output - error.
func (s *ReservationManagerImpl) SetDefaultPolicy(ctx context.Context, currencyID byte, unlimited bool, max *big.Int) error {
	log.Debugf("Set default policy for %v to be %v-%v", currencyID, unlimited, max)
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

	dsVal := []byte(big.NewInt(-1).String())
	if !unlimited {
		if max.Cmp(big.NewInt(0)) < 0 {
			return fmt.Errorf("max cannot be negative")
		}
		dsVal = []byte(max.String())
	}

	err = txn.Put(ctx, dsVal, currencyID, DefaultPolicyID)
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v-%v: %v", dsVal, currencyID, DefaultPolicyID, err.Error())
		return err
	}

	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// GetDefaultPolicy gets the default reservation policy.
//
// @input - context, currency id.
//
// @output - if unlimited, max single reservation amount, error.
func (s *ReservationManagerImpl) GetDefaultPolicy(ctx context.Context, currencyID byte) (bool, *big.Int, error) {
	log.Debugf("Get default policy for %v", currencyID)
	release, err := s.ds.RLock(ctx, currencyID, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
		return false, nil, err
	}
	defer release()

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return false, nil, err
	}
	defer txn.Discard(context.Background())

	val, err := txn.Get(ctx, currencyID, DefaultPolicyID)
	if err != nil {
		log.Warnf("Fail to get ds value for %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
		return false, nil, err
	}

	max, ok := big.NewInt(0).SetString(string(val), 10)
	if !ok {
		log.Errorf("Fail to decode max from %v, should never happen: %v", val, err.Error())
		return false, nil, fmt.Errorf("fail to decode ds value")
	}
	if max.Cmp(big.NewInt(0)) < 0 {
		return true, nil, nil
	}
	return false, max, nil
}

// RemoveDefaultPolicy removes the default reservation policy.
//
// @input - context, currency id.
//
// @output - error.
func (s *ReservationManagerImpl) RemoveDefaultPolicy(ctx context.Context, currencyID byte) error {
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

// SetPaychPolicy sets the reservation policy for a given paych.
//
// @input - context, currency id, channel address, if unlimited reservation, maximum single reservation amount.
//
// @output - error.
func (s *ReservationManagerImpl) SetPaychPolicy(ctx context.Context, currencyID byte, chAddr string, unlimited bool, max *big.Int) error {
	log.Debugf("Set paych policy for %v-%v to be %v-%v", currencyID, chAddr, unlimited, max)
	if chAddr == DefaultPolicyID {
		return fmt.Errorf("chAddr cannot be default policy id")
	}
	release, err := s.ds.Lock(ctx, currencyID, chAddr, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v: %v", currencyID, chAddr, DefaultPolicyID, err.Error())
		return err
	}
	defer release()

	txn, err := s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	dsVal := []byte(big.NewInt(-1).String())
	if !unlimited {
		if max.Cmp(big.NewInt(0)) < 0 {
			return fmt.Errorf("max cannot be negative")
		}
		dsVal = []byte(max.String())
	}

	err = txn.Put(ctx, dsVal, currencyID, chAddr, DefaultPolicyID)
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v-%v-%v: %v", dsVal, currencyID, chAddr, DefaultPolicyID, err.Error())
		return err
	}

	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// GetPaychPolicy gets the reservation policy for a given paych.
//
// @input - context, currency id, channel address.
//
// @output - if unlimited reservation, maximum single reservation amount, error.
func (s *ReservationManagerImpl) GetPaychPolicy(ctx context.Context, currencyID byte, chAddr string) (bool, *big.Int, error) {
	log.Debugf("Get paych policy for %v-%v", currencyID, chAddr)
	if chAddr == DefaultPolicyID {
		return false, nil, fmt.Errorf("chAddr cannot be default policy id")
	}
	release1, err := s.ds.RLock(ctx, currencyID, chAddr, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, chAddr, DefaultPolicyID, err.Error())
		return false, nil, err
	}
	defer release1()

	release2, err := s.ds.RLock(ctx, currencyID, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
		return false, nil, err
	}
	defer release2()

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return false, nil, err
	}

	var val []byte
	exists, err := txn.Has(ctx, currencyID, chAddr, DefaultPolicyID)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, chAddr, DefaultPolicyID, err.Error())
		return false, nil, err
	}
	if exists {
		val, err = txn.Get(ctx, currencyID, chAddr, DefaultPolicyID)
		if err != nil {
			log.Warnf("Fail to get ds value for %v-%v-%v: %v", currencyID, chAddr, DefaultPolicyID, err.Error())
			return false, nil, err
		}
	} else {
		val, err = txn.Get(ctx, currencyID, DefaultPolicyID)
		if err != nil {
			log.Warnf("Fail to get ds value for %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
			return false, nil, err
		}
	}

	max, ok := big.NewInt(0).SetString(string(val), 10)
	if !ok {
		log.Errorf("Fail to decode max from %v, should never happen: %v", val, err.Error())
		return false, nil, fmt.Errorf("fail to decode ds value")
	}
	if max.Cmp(big.NewInt(0)) < 0 {
		return true, nil, nil
	}
	return false, max, nil
}

// RemovePaychPolicy removes the reservation policy for a given paych.
//
// @input - context, currency id, channel address.
//
// @output - error.
func (s *ReservationManagerImpl) RemovePaychPolicy(ctx context.Context, currencyID byte, chAddr string) error {
	log.Debugf("Remove paych policy for %v-%v", currencyID, chAddr)
	if chAddr == DefaultPolicyID {
		return fmt.Errorf("chAddr cannot be default policy id")
	}
	// Check first.
	release, err := s.ds.RLock(ctx, currencyID, chAddr, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, chAddr, DefaultPolicyID, err.Error())
		return err
	}

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, chAddr, DefaultPolicyID)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, chAddr, DefaultPolicyID, err.Error())
		return err
	}
	if !exists {
		release()
		return fmt.Errorf("policy does not exist for %v-%v", currencyID, chAddr)
	}
	release()

	release, err = s.ds.Lock(ctx, currencyID, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, chAddr, err.Error())
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

	exists, err = txn.Has(ctx, currencyID, chAddr, DefaultPolicyID)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, chAddr, DefaultPolicyID, err.Error())
		return err
	}
	if !exists {
		release()
		return fmt.Errorf("policy does not exist for %v-%v", currencyID, chAddr)
	}

	err = txn.Delete(ctx, currencyID, chAddr, DefaultPolicyID)
	if err != nil {
		release()
		log.Warnf("Fail to remove %v-%v-%v: %v", currencyID, chAddr, DefaultPolicyID, err.Error())
		return err
	}

	// Check if parent is empty
	children, err := txn.GetChildren(ctx, currencyID, chAddr)
	if err != nil {
		release()
		log.Warnf("Fail to get children for %v-%v: %v", currencyID, chAddr, err.Error())
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
			log.Warnf("Fail to obtain write access to remove empty %v-%v: %v", currencyID, chAddr, err.Error())
			return nil
		}
		defer release()
		txn, err := s.ds.NewTransaction(ctx, false)
		if err != nil {
			log.Warnf("Fail to start new transaction to remove empty %v-%v: %v", currencyID, chAddr, err.Error())
			return nil
		}
		defer txn.Discard(context.Background())
		children, err = txn.GetChildren(ctx, currencyID, chAddr)
		if err != nil {
			log.Warnf("Fail to get children for to be removed empty %v-%v: %v", currencyID, chAddr, err.Error())
			return nil
		}
		if len(children) == 0 {
			// Note is still empty.
			err = txn.Delete(ctx, currencyID, chAddr)
			if err != nil {
				log.Warnf("Fail to remove empty %v-%v: %v", currencyID, chAddr, err.Error())
				return nil
			}
			err = txn.Commit(ctx)
			if err != nil {
				log.Warnf("Fail to commit transaction to remove empty %v-%v: %v", currencyID, chAddr, err.Error())
			}
		}
	}
	return nil
}

// SetPeerPolicy sets the reservation policy for a given peer.
//
// @input - context, currency id, channel address, peer address, if unlimited reservation, maximum single reservation amount.
//
// @output - error.
func (s *ReservationManagerImpl) SetPeerPolicy(ctx context.Context, currencyID byte, chAddr string, peerAddr string, unlimited bool, max *big.Int) error {
	log.Debugf("Set peer policy for %v-%v-%v to be %v-%v", currencyID, chAddr, peerAddr, unlimited, max)
	if chAddr == DefaultPolicyID || peerAddr == DefaultPolicyID {
		return fmt.Errorf("chAddr or peerAddr cannot be default policy id")
	}
	release, err := s.ds.Lock(ctx, currencyID, chAddr, peerAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v: %v", currencyID, chAddr, peerAddr, err.Error())
		return err
	}
	defer release()

	txn, err := s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	dsVal := []byte(big.NewInt(-1).String())
	if !unlimited {
		if max.Cmp(big.NewInt(0)) < 0 {
			return fmt.Errorf("max cannot be negative")
		}
		dsVal = []byte(max.String())
	}

	err = txn.Put(ctx, dsVal, currencyID, chAddr, peerAddr)
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v-%v-%v: %v", dsVal, currencyID, chAddr, peerAddr, err.Error())
		return err
	}

	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// GetPeerPolicy gets the reservation policy for a given peer.
//
// @input - context, currency id, channel address, peer address.
//
// @output - if unlimited reservation, maximum single reservation amount, error.
func (s *ReservationManagerImpl) GetPeerPolicy(ctx context.Context, currencyID byte, chAddr string, peerAddr string) (bool, *big.Int, error) {
	log.Debugf("Get peer policy for %v-%v-%v", currencyID, chAddr, peerAddr)
	if chAddr == DefaultPolicyID || peerAddr == DefaultPolicyID {
		return false, nil, fmt.Errorf("chAddr or peerAddr cannot be default policy id")
	}
	release1, err := s.ds.RLock(ctx, currencyID, chAddr, peerAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, chAddr, peerAddr, err.Error())
		return false, nil, err
	}
	defer release1()

	release2, err := s.ds.RLock(ctx, currencyID, chAddr, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, chAddr, DefaultPolicyID, err.Error())
		return false, nil, err
	}
	defer release2()

	release3, err := s.ds.RLock(ctx, currencyID, DefaultPolicyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
		return false, nil, err
	}
	defer release3()

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return false, nil, err
	}

	var val []byte
	exists, err := txn.Has(ctx, currencyID, chAddr, peerAddr)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, chAddr, peerAddr, err.Error())
		return false, nil, err
	}
	if exists {
		val, err = txn.Get(ctx, currencyID, chAddr, peerAddr)
		if err != nil {
			log.Warnf("Fail to get ds value for %v-%v-%v: %v", currencyID, chAddr, peerAddr, err.Error())
			return false, nil, err
		}
	} else {
		exists, err := txn.Has(ctx, currencyID, chAddr, DefaultPolicyID)
		if err != nil {
			log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, chAddr, DefaultPolicyID, err.Error())
			return false, nil, err
		}
		if exists {
			val, err = txn.Get(ctx, currencyID, chAddr, DefaultPolicyID)
			if err != nil {
				log.Warnf("Fail to get ds value for %v-%v-%v: %v", currencyID, chAddr, DefaultPolicyID, err.Error())
				return false, nil, err
			}
		} else {
			val, err = txn.Get(ctx, currencyID, DefaultPolicyID)
			if err != nil {
				log.Warnf("Fail to get ds value for %v-%v: %v", currencyID, DefaultPolicyID, err.Error())
				return false, nil, err
			}
		}
	}

	max, ok := big.NewInt(0).SetString(string(val), 10)
	if !ok {
		log.Errorf("Fail to decode max from %v, should never happen: %v", val, err.Error())
		return false, nil, fmt.Errorf("fail to decode ds value")
	}
	if max.Cmp(big.NewInt(0)) < 0 {
		return true, nil, nil
	}
	return false, max, nil
}

// RemovePeerPolicy removes the reservation policy for a given peer.
//
// @input - context, currency id, channel address, peer address.
//
// @output - error.
func (s *ReservationManagerImpl) RemovePeerPolicy(ctx context.Context, currencyID byte, chAddr string, peerAddr string) error {
	log.Debugf("Remove peer policy for %v-%v-%v", currencyID, chAddr, peerAddr)
	if chAddr == DefaultPolicyID || peerAddr == DefaultPolicyID {
		return fmt.Errorf("chAddr or peerAddr cannot be default policy id")
	}
	// Check first.
	release, err := s.ds.RLock(ctx, currencyID, chAddr, peerAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, chAddr, peerAddr, err.Error())
		return err
	}

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, chAddr, peerAddr)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, chAddr, peerAddr, err.Error())
		return err
	}
	if !exists {
		release()
		return fmt.Errorf("policy does not exist for %v-%v-%v", currencyID, chAddr, peerAddr)
	}
	release()

	release, err = s.ds.Lock(ctx, currencyID, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, chAddr, err.Error())
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

	exists, err = txn.Has(ctx, currencyID, chAddr, peerAddr)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, chAddr, peerAddr, err.Error())
		return err
	}
	if !exists {
		release()
		return fmt.Errorf("policy does not exist for %v-%v-%v", currencyID, chAddr, peerAddr)
	}

	err = txn.Delete(ctx, currencyID, chAddr, peerAddr)
	if err != nil {
		release()
		log.Warnf("Fail to remove %v-%v-%v: %v", currencyID, chAddr, peerAddr, err.Error())
		return err
	}

	// Check if parent is empty
	children, err := txn.GetChildren(ctx, currencyID, chAddr)
	if err != nil {
		release()
		log.Warnf("Fail to get children for %v-%v: %v", currencyID, chAddr, err.Error())
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
			log.Warnf("Fail to obtain write lock to remove empty %v-%v: %v", currencyID, chAddr, err.Error())
			return nil
		}
		defer release()

		txn, err := s.ds.NewTransaction(ctx, false)
		if err != nil {
			log.Warnf("Fail to start a new transaction to remove empty %v-%v: %v", currencyID, chAddr, err.Error())
			return nil
		}
		defer txn.Discard(context.Background())

		children, err = txn.GetChildren(ctx, currencyID, chAddr)
		if err != nil {
			log.Warnf("Fail to get children to remove empty %v-%v: %v", currencyID, chAddr, err.Error())
			return nil
		}
		if len(children) == 0 {
			// Note is still empty.
			err = txn.Delete(ctx, currencyID, chAddr)
			if err != nil {
				log.Warnf("Fail to remove empty %v-%v: %v", currencyID, chAddr, err.Error())
				return nil
			}
			err = txn.Commit(ctx)
			if err != nil {
				log.Warnf("Fail to commit transaction to remove empty %v-%v: %v", currencyID, chAddr, err.Error())
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
func (s *ReservationManagerImpl) ListCurrencyIDs(ctx context.Context) (<-chan byte, <-chan error) {
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

// ListPaychs lists all paychs.
//
// @input - context, currency id.
//
// @output - paych chan out, error chan out.
func (s *ReservationManagerImpl) ListPaychs(ctx context.Context, currencyID byte) (<-chan string, <-chan error) {
	out := make(chan string, 128)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Listing paychs for %v", currencyID)
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

// ListPeers lists all peers.
//
// @input - context, currency id, paych address.
//
// @output - peer chan out, error chan out.
func (s *ReservationManagerImpl) ListPeers(ctx context.Context, currencyID byte, chAddr string) (<-chan string, <-chan error) {
	out := make(chan string, 128)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Listing peers for %v-%v", currencyID, chAddr)
		defer func() {
			close(out)
			close(errChan)
		}()
		release, err := s.ds.RLock(ctx, currencyID, chAddr)
		if err != nil {
			errChan <- err
			log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, chAddr, err.Error())
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

		exists, err := txn.Has(ctx, currencyID, chAddr)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to check if contains %v-%v: %v", currencyID, chAddr, err.Error())
			return
		}
		if !exists {
			release()
			return
		}
		children, err := txn.GetChildren(ctx, currencyID, chAddr)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to get children for %v-%v: %v", currencyID, chAddr, err.Error())
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
