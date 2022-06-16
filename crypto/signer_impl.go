package crypto

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

	golock "github.com/viney-shih/go-lock"
	"github.com/wcgcyx/fcr/mdlstore"
)

// SignerImpl is the implementation of the Signer interface.
type SignerImpl struct {
	// Datastore
	ds mdlstore.MultiDimLockableStore

	// Address cache
	addrs    map[byte]string
	keyTypes map[byte]byte

	// Currently retiring keys.
	retiring     map[byte]chan bool
	retiringLock golock.RWMutex

	// Shutdown function.
	shutdown func()
}

// NewSignerImpl creates a new Signer.
//
// @input - context, options.
//
// @output - signer, error.
func NewSignerImpl(ctx context.Context, opts Opts) (*SignerImpl, error) {
	log.Infof("Start signer...")
	var err error
	// Open store.
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, opts.Path)
	if err != nil {
		log.Errorf("Fail to open mdlstore: %v", err.Error())
		return nil, err
	}
	defer func() {
		if err != nil {
			log.Infof("Fail to start signer, close ds...")
			err = ds.Shutdown(context.Background())
			if err != nil {
				log.Errorf("Fail to close ds after failing to start signer: %v", err.Error())
			}
		}
	}()
	// Try to list all keys.
	txn, err := ds.NewTransaction(ctx, true)
	if err != nil {
		log.Errorf("Fail to start new transaction: %v", err.Error())
		return nil, err
	}
	defer txn.Discard(context.Background())
	children, err := txn.GetChildren(ctx)
	if err != nil {
		log.Errorf("Fail to get children: %v", err.Error())
		return nil, err
	}
	addrs := make(map[byte]string, 0)
	keyTypes := make(map[byte]byte, 0)
	for child := range children {
		currencyID, err := strconv.Atoi(child)
		if err != nil {
			log.Errorf("Fail to convert child key %v to currency id: %v", child, err.Error())
			return nil, err
		}
		val, err := txn.Get(ctx, currencyID)
		if err != nil {
			log.Errorf("Fail to get value for currency id %v: %v", currencyID, err.Error())
			return nil, err
		}
		keyType, addr, _, err := decDSVal(val)
		if err != nil {
			log.Errorf("Fail to decode ds value: %v", err.Error())
			return nil, err
		}
		addrs[byte(currencyID)] = addr
		keyTypes[byte(currencyID)] = keyType
	}
	s := &SignerImpl{
		ds:           ds,
		addrs:        addrs,
		keyTypes:     keyTypes,
		retiring:     make(map[byte]chan bool),
		retiringLock: golock.NewCASMutex()}
	s.shutdown = func() {
		log.Infof("Stop all retiring routines...")
		s.retiringLock.Lock()
		for _, trigger := range s.retiring {
			trigger <- true
		}
		log.Infof("Stop datastore...")
		err := ds.Shutdown(context.Background())
		if err != nil {
			log.Errorf("Fail to stop datastore: %v", err.Error())
		}
	}
	return s, nil
}

// Shutdown safely shuts down the component.
func (s *SignerImpl) Shutdown() {
	log.Infof("Start shutdown...")
	s.shutdown()
}

// SetKey is used to set a private key for a given currency id. It will return error if there is
// an existing key already associated with the given currency id.
//
// @input - context, currency id, key type, private key.
//
// @output - error.
func (s *SignerImpl) SetKey(ctx context.Context, currencyID byte, keyType byte, prv []byte) error {
	log.Debugf("Signer set key for %v", currencyID)
	go s.touchKey(currencyID)
	resolver, err := getResolver(currencyID, keyType)
	if err != nil {
		log.Debugf("Fail to get resolver for %v-%v: %v", currencyID, keyType, err.Error())
		return err
	}
	addr, err := resolver(prv, true)
	if err != nil {
		log.Debugf("Fail to resolve private key: %v", err.Error())
		return err
	}
	// Check if private key is valid
	signer, err := getSigner(currencyID, keyType)
	if err != nil {
		log.Debugf("Fail to get signer for %v-%v: %v", currencyID, keyType)
		return err
	}
	_, err = signer(prv, []byte{0})
	if err != nil {
		log.Debugf("Fail to test signing: %v", err.Error())
		return err
	}
	// Read first
	release, err := s.ds.RLock(ctx, currencyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v: %v", currencyID, err.Error())
		return err
	}
	_, ok := s.addrs[currencyID]
	if ok {
		release()
		log.Debugf("key exists for %v", currencyID)
		return fmt.Errorf("key exists for %v", currencyID)
	}
	release()
	// Lock and read again.
	release, err = s.ds.Lock(ctx, currencyID)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v: %v", currencyID, err.Error())
		return err
	}
	defer release()
	_, ok = s.addrs[currencyID]
	if ok {
		log.Debugf("key exists for %v", currencyID)
		return fmt.Errorf("key exists for %v", currencyID)
	}
	txn, err := s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())
	dsVal, err := encDSVal(keyType, addr, prv)
	if err != nil {
		log.Errorf("Fail to encode ds value: %v", err.Error())
		return err
	}
	err = txn.Put(ctx, dsVal, currencyID)
	if err != nil {
		log.Warnf("Fail to put ds value for %v: %v", currencyID, err.Error())
		return err
	}
	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	s.addrs[currencyID] = addr
	s.keyTypes[currencyID] = keyType
	return nil
}

// GetAddr is used to obtain the key type and the address derived from the private key associated
// with the given currency id.
//
// @input - context, currency id.
//
// @output - key type, address, error.
func (s *SignerImpl) GetAddr(ctx context.Context, currencyID byte) (byte, string, error) {
	log.Debugf("Signer get addr for %v", currencyID)
	go s.touchKey(currencyID)
	// Try to read.
	release, err := s.ds.RLock(ctx, currencyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v: %v", currencyID, err.Error())
		return 0, "", err
	}
	defer release()
	addr, ok := s.addrs[currencyID]
	if !ok {
		log.Debugf("Key does not exist for %v", currencyID)
		return 0, "", fmt.Errorf("key does not exist for %v", currencyID)
	}
	return s.keyTypes[currencyID], addr, nil
}

// Sign is used to sign given data with the private key associated with given currency id.
//
// @input - context, currency id, data.
//
// @output - key type, signature, error.
func (s *SignerImpl) Sign(ctx context.Context, currencyID byte, data []byte) (byte, []byte, error) {
	log.Debugf("Signer sign with key linked to %v", currencyID)
	go s.touchKey(currencyID)
	// Try to read.
	release, err := s.ds.RLock(ctx, currencyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v: %v", currencyID, err.Error())
		return 0, nil, err
	}
	defer release()
	_, ok := s.addrs[currencyID]
	if !ok {
		log.Debugf("Key does not exist for %v", currencyID)
		return 0, nil, fmt.Errorf("key does not exist for currency id %v", currencyID)
	}
	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return 0, nil, err
	}
	defer txn.Discard(context.Background())
	val, err := txn.Get(ctx, currencyID)
	if err != nil {
		log.Warnf("Fail to get ds value for %v: %v", currencyID, err.Error())
		return 0, nil, err
	}
	keyType, _, prv, err := decDSVal(val)
	if err != nil {
		log.Errorf("Fail to decode ds value: %v", err.Error())
		return 0, nil, err
	}
	signer, err := getSigner(currencyID, keyType)
	if err != nil {
		log.Errorf("Fail to get signer for %v-%v: %v", currencyID, keyType, err.Error())
		return 0, nil, err
	}
	sig, err := signer(prv, data)
	if err != nil {
		log.Errorf("Fail to sign data: %v", err.Error())
		return 0, nil, err
	}
	return keyType, sig, nil
}

// RetireKey is used to retire a key for a given currency id. If within timeout the key is touched
// it will return error.
//
// @input - context, currency id, timeout.
//
// @output - error channel out.
func (s *SignerImpl) RetireKey(ctx context.Context, currencyID byte, timeout time.Duration) <-chan error {
	out := make(chan error, 1)
	go func() {
		log.Warnf("Start retiring key for %v with timeout of %v", currencyID, timeout)
		// First of all, check if currency id exists.
		release, err := s.ds.RLock(ctx, currencyID)
		if err != nil {
			out <- err
			log.Debugf("Fail to obtain read lock for %v: %v", currencyID, err.Error())
			return
		}
		toRemove, ok := s.addrs[currencyID]
		if !ok {
			release()
			log.Debugf("Key does not exist for %v", currencyID)
			out <- fmt.Errorf("key does not exist for %v", currencyID)
			return
		}
		release()
		// Now add retire key to watch list.
		// Read first to check if already retiring.
		if !s.retiringLock.RTryLockWithContext(ctx) {
			log.Debugf("Fail to obtain read lock for retiring lock")
			out <- fmt.Errorf("fail to obtain read lock for retiring lock")
			return
		}
		release = s.retiringLock.RUnlock
		_, ok = s.retiring[currencyID]
		if ok {
			release()
			log.Debugf("Key has already in retiring")
			out <- fmt.Errorf("key has already in retiring")
			return
		}
		// Switch to write lock.
		release()
		if !s.retiringLock.TryLockWithContext(ctx) {
			log.Debugf("Fail to obtain write lock for retiring lock")
			out <- fmt.Errorf("fail to obtain write lock for retiring lock")
			return
		}
		release = s.retiringLock.Unlock
		// Check again
		_, ok = s.retiring[currencyID]
		if ok {
			release()
			log.Debugf("Key just be in retiring for %v", currencyID)
			out <- fmt.Errorf("key just be in retiring for %v", currencyID)
			return
		}
		trigger := make(chan bool, 1)
		s.retiring[currencyID] = trigger
		release()
		defer func() {
			// Remove watch in the end.
			if !s.retiringLock.TryLockWithContext(ctx) {
				log.Warnf("Fail to obtain write lock to remove retiring entry")
				return
			}
			delete(s.retiring, currencyID)
			s.retiringLock.Unlock()
		}()
		// Now start the wait process
		timeoutCh := time.After(timeout)
		select {
		case <-timeoutCh:
			// Timeout occurs, now stop the watch and remove the key.
			// Remove the key.
			log.Infof("Now removing key for %v", currencyID)
			release, err := s.ds.Lock(ctx)
			if err != nil {
				log.Debugf("Fail to obtain write lock for root: %v", err.Error())
				out <- err
				return
			}
			addr := s.addrs[currencyID]
			if addr != toRemove {
				release()
				log.Debugf("Planning to remove %v but changed to remove %v", toRemove, addr)
				out <- fmt.Errorf("planning to remove %v but changed to remove %v", toRemove, addr)
				return
			}
			// Now to remove.
			txn, err := s.ds.NewTransaction(ctx, false)
			if err != nil {
				release()
				log.Warnf("Fail to start new transaction: %v", err.Error())
				out <- err
				return
			}
			defer txn.Discard(context.Background())
			err = txn.Delete(ctx, currencyID)
			if err != nil {
				release()
				log.Warnf("Fail to delete %v: %v", currencyID, err.Error())
				out <- err
				return
			}
			err = txn.Commit(ctx)
			if err != nil {
				release()
				log.Warnf("Fail to commit transaction: %v", err.Error())
				out <- err
				return
			}
			delete(s.addrs, currencyID)
			delete(s.keyTypes, currencyID)
			release()
			out <- nil
		case <-trigger:
			// Failed.
			log.Warnf("Key is accessed for %v, fail to retire", currencyID)
			out <- fmt.Errorf("key is accessed for %v now, fail to retire", currencyID)
		case <-ctx.Done():
			// User cancelled
			log.Warnf(ctx.Err().Error())
			out <- ctx.Err()
		}
	}()
	return out
}

// StopRetire is used to stop retiring a key for a given currency id.
//
// @input - context, currency id.
//
// @output - error.
func (s *SignerImpl) StopRetire(ctx context.Context, currencyID byte) error {
	log.Debugf("Stop retiring key for %v", currencyID)
	s.touchKey(currencyID)
	if !s.retiringLock.TryLockWithContext(ctx) {
		log.Debugf("Fail to obtain write lock for retiring lock")
		return fmt.Errorf("fail to obtain lock for retiring lock")
	}
	delete(s.retiring, currencyID)
	s.retiringLock.Unlock()
	return nil
}

// ListCurrencyIDs lists all currencies.
//
// @input - context.
//
// @output - currency chan out, error chan out.
func (s *SignerImpl) ListCurrencyIDs(ctx context.Context) (<-chan byte, <-chan error) {
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
			log.Debugf("Fail to obtain read lock for root: %v", err.Error())
			errChan <- err
			return
		}
		txn, err := s.ds.NewTransaction(ctx, true)
		if err != nil {
			release()
			log.Warnf("Fail to start a new transaction: %v", err.Error())
			errChan <- err
			return
		}
		defer txn.Discard(context.Background())
		children, err := txn.GetChildren(ctx)
		if err != nil {
			release()
			log.Warnf("Fail to get children at root: %v", err.Error())
			errChan <- err
			return
		}
		release()
		for child := range children {
			currencyID, err := strconv.Atoi(child)
			if err != nil {
				log.Errorf("Fail to convert child key %v to currency id: %v", child, err.Error())
				errChan <- err
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

// touchKey touches a key for a given currency id.
//
// @input - currency id.
func (s *SignerImpl) touchKey(currencyID byte) {
	log.Debugf("Touch currency id %v", currencyID)
	s.retiringLock.RLock()
	trigger, ok := s.retiring[currencyID]
	if ok {
		log.Debugf("Send trigger for %v", currencyID)
		trigger <- true
	}
	s.retiringLock.RUnlock()
}
