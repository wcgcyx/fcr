package paychstore

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
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"time"

	gcq "github.com/enriquebris/goconcurrentqueue"
	logging "github.com/ipfs/go-log"
	"github.com/wcgcyx/fcr/mdlstore"
	"github.com/wcgcyx/fcr/paychstate"
)

// PaychStoreImpl is the implementation of the PaychStore interface.
type PaychStoreImpl struct {
	// Logger
	log *logging.ZapEventLogger

	// Datastore
	ds mdlstore.MultiDimLockableStore

	// Queue related
	ctx   context.Context
	wg    sync.WaitGroup
	queue gcq.Queue

	// Boolean indicating if payment channels stored are outbound.
	outbound bool

	// Shutdown function.
	shutdown func()

	// Options
	dsTimeout time.Duration
	dsRetry   uint64
}

// NewPaychStoreImpl creates a new PaychStoreImpl.
//
// @input - context, if it is storing active channel, if it is outbound channel, options.
//
// @output - store, error.
func NewPaychStoreImpl(ctx context.Context, active bool, outbound bool, opts Opts) (*PaychStoreImpl, error) {
	var log *logging.ZapEventLogger
	if active && outbound {
		log = log1
	} else if active && !outbound {
		log = log2
	} else if !active && outbound {
		log = log3
	} else if !active && !outbound {
		log = log4
	}
	// Parse options.
	dsTimeout := opts.DSTimeout
	if dsTimeout == 0 {
		dsTimeout = defaultDSTimeout
	}
	dsRetry := opts.DSRetry
	if dsRetry == 0 {
		dsRetry = defaultDSRetry
	}
	log.Infof("Start paych store...")
	// Open store.
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, opts.Path)
	if err != nil {
		log.Errorf("Fail to open mdlstore: %v", err.Error())
		return nil, err
	}
	defer func() {
		if err != nil {
			log.Infof("Fail to start paych store, close ds...")
			err = ds.Shutdown(context.Background())
			if err != nil {
				log.Errorf("Fail to close ds after failing to start paych store: %v", err.Error())
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
		for peerAddr := range children2 {
			children3, err := txn.GetChildren(ctx, currencyID, peerAddr)
			if err != nil {
				log.Errorf("Fail to get children for %v-%v: %v", currencyID, peerAddr, err.Error())
				return nil, err
			}
			if len(children3) == 0 {
				// Empty, need to remove it.
				err = txn.Delete(ctx, currencyID, peerAddr)
				if err != nil {
					log.Errorf("Fail to remove empty %v-%v: %v", currencyID, peerAddr, err.Error())
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
	// Create queue.
	queue := gcq.NewFIFO()
	wg := sync.WaitGroup{}
	queueCtx, cancel := context.WithCancel(context.Background())
	s := &PaychStoreImpl{
		log:       log,
		ds:        ds,
		ctx:       queueCtx,
		wg:        wg,
		queue:     queue,
		outbound:  outbound,
		dsTimeout: dsTimeout,
		dsRetry:   dsRetry,
	}
	s.shutdown = func() {
		s.log.Infof("Stop and finalising all routines...")
		wg.Wait()
		cancel()
		s.log.Infof("Stop datastore...")
		err := ds.Shutdown(context.Background())
		if err != nil {
			s.log.Errorf("Fail to stop datastore: %v", err.Error())
		}
	}
	// Start processing queue.
	go s.processQueue()
	return s, nil
}

// Shutdown safely shuts down the component.
func (s *PaychStoreImpl) Shutdown() {
	s.log.Infof("Start shutdown...")
	s.shutdown()
}

// Upsert is used to update or insert a channel state.
//
// @input - channel state.
func (s *PaychStoreImpl) Upsert(state paychstate.State) {
	s.wg.Add(1)
	go func() {
		s.queue.Enqueue(func() {
			s.log.Debugf("Start upserting channel state for %v-%v", state.CurrencyID, state.ChAddr)
			defer s.wg.Done()
			newData, err := state.Encode()
			if err != nil {
				// The state is invalid.
				// It shouldn't happen as the encoding will never fail.
				s.log.Error("State received is invalid, this should never happen: %v", err.Error())
				return
			}
			var peer string
			if s.outbound {
				peer = state.ToAddr
			} else {
				peer = state.FromAddr
			}
			// Retry operation for s.dsRetry times.
			for i := 0; i < int(s.dsRetry); i++ {
				succeed := func() bool {
					s.log.Debugf("Attempt %v", i)
					// Each operation has a timeout of s.dsTimeout.
					subCtx, cancel := context.WithTimeout(s.ctx, s.dsTimeout)
					defer cancel()

					// Read first.
					release, err := s.ds.RLock(subCtx, state.CurrencyID, peer, state.ChAddr)
					if err != nil {
						s.log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", state.CurrencyID, peer, state.ChAddr, err.Error())
						return false
					}

					txn, err := s.ds.NewTransaction(subCtx, true)
					if err != nil {
						release()
						s.log.Warnf("Fail to start new transaction: %v", err.Error())
						return false
					}
					defer txn.Discard(context.Background())

					exists, err := txn.Has(subCtx, state.CurrencyID, peer, state.ChAddr)
					if err != nil {
						release()
						s.log.Warnf("Fail to check if contains %v-%v-%v: %v", state.CurrencyID, peer, state.ChAddr, err.Error())
						return false
					}
					if exists {
						s.log.Debugf("There is an existing state, check if state to upsert is old...")
						// There is an existing state. Read to see if this state to upsert is old and should be discarded.
						data, err := txn.Get(subCtx, state.CurrencyID, peer, state.ChAddr)
						if err != nil {
							release()
							s.log.Warnf("Fail to read ds value for %v-%v-%v: %v", state.CurrencyID, peer, state.ChAddr, err.Error())
							return false
						}
						currentState := paychstate.State{}
						err = currentState.Decode(data)
						if err != nil {
							release()
							s.log.Errorf("Fail to decode current state, this should never happen: %v", err.Error())
							return false
						}
						// Check nonce
						if currentState.StateNonce > state.StateNonce {
							// Current state has higher state nonce than this state to upsert. Check overflow situation.
							// This situation is extremely unlikely to happen, unless there will be 2^64 operations
							// in one state. In this extreme case, a constant of 100 is chosen to detect overflow.
							// Unless there are 101 mutual access to this state, it should be safe.
							if state.StateNonce < 100 && (^uint64(0)-currentState.StateNonce < 100) {
								// Ignore, this is an overflow situation.
								// Meaning the current state is old and this state to upsert is new.
								// It won't happen.
							} else {
								// This state to upsert is old, discard.
								release()
								s.log.Debug("State to upsert is old, discard")
								return true
							}
						}
					}
					// Either the state does not exist, or the current state is older than the state to upsert.
					release()
					// Lock and read again.
					release, err = s.ds.Lock(subCtx, state.CurrencyID, peer, state.ChAddr)
					if err != nil {
						s.log.Debugf("Fail to obtain write lock for %v-%v-%v: %v", state.CurrencyID, peer, state.ChAddr, err.Error())
						return false
					}

					// Read again
					txn, err = s.ds.NewTransaction(subCtx, false)
					if err != nil {
						release()
						s.log.Warnf("Fail to start new transaction: %v", err.Error())
						return false
					}
					defer txn.Discard(context.Background())

					exists, err = txn.Has(subCtx, state.CurrencyID, peer, state.ChAddr)
					if err != nil {
						release()
						s.log.Warnf("Fail to check if contains %v-%v-%v: %v", state.CurrencyID, peer, state.ChAddr, err.Error())
						return false
					}
					if exists {
						s.log.Debugf("There is an existing state, check if state to upsert is old...")
						// Check again.
						data, err := txn.Get(subCtx, state.CurrencyID, peer, state.ChAddr)
						if err != nil {
							release()
							s.log.Warnf("Fail to read ds value for %v-%v-%v: %v", state.CurrencyID, peer, state.ChAddr, err.Error())
							return false
						}
						currentState := paychstate.State{}
						err = currentState.Decode(data)
						if err != nil {
							release()
							s.log.Errorf("Fail to decode current state, this should never happen: %v", err.Error())
							return false
						}
						if currentState.StateNonce > state.StateNonce {
							if state.StateNonce < 100 && (^uint64(0)-currentState.StateNonce < 100) {
							} else {
								release()
								s.log.Debug("State to upsert is old, discard")
								return true
							}
						}
					}
					// Either the state does not exist still or the current state is still older than the state to upsert.
					err = txn.Put(subCtx, newData, state.CurrencyID, peer, state.ChAddr)
					if err != nil {
						release()
						s.log.Warnf("Fail to put ds value %v-%v-%v: %v", state.CurrencyID, peer, state.ChAddr, err.Error())
						return false
					}

					err = txn.Commit(subCtx)
					if err != nil {
						release()
						s.log.Warnf("Fail to commit transaction: %v", err.Error())
						return false
					}
					// Succeed.
					release()
					return true
				}()
				if succeed {
					return
				}
			}
			// Failed after s.dsRetry times, logging to error.
			// The logging can be later used to repair the ds.
			s.log.Errorf("Fail to upsert %v", hex.EncodeToString(newData))
		})
	}()
}

// Remove is used to remove a channel state.
//
// @input - currency id, peer address, channel address.
func (s *PaychStoreImpl) Remove(currencyID byte, peerAddr string, chAddr string) {
	s.wg.Add(1)
	go func() {
		s.queue.Enqueue(func() {
			s.log.Debugf("Start removing channel state for %v-%v-%v", currencyID, peerAddr, chAddr)
			defer s.wg.Done()
			for i := 0; i < int(s.dsRetry); i++ {
				succeed := func() bool {
					s.log.Debugf("Attempt %v", i)
					// Each operation has a timeout of s.dsTimeout.
					ctx, cancel := context.WithTimeout(s.ctx, s.dsTimeout)
					defer cancel()

					// Read first.
					release, err := s.ds.RLock(ctx, currencyID, peerAddr, chAddr)
					if err != nil {
						s.log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, peerAddr, chAddr, err.Error())
						return false
					}

					txn, err := s.ds.NewTransaction(ctx, true)
					if err != nil {
						release()
						s.log.Warnf("Fail to start new transaction: %v", err.Error())
						return false
					}
					defer txn.Discard(context.Background())

					exists, err := txn.Has(ctx, currencyID, peerAddr, chAddr)
					if err != nil {
						release()
						s.log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, peerAddr, chAddr, err.Error())
						return false
					}
					if !exists {
						// The entry does not exist, discard this operation.
						release()
						s.log.Debugf("Operation discarded as %v-%v-%v does not exist", currencyID, peerAddr, chAddr)
						return true
					}
					// This entry does exist.
					release()
					release, err = s.ds.Lock(ctx, currencyID, peerAddr)
					if err != nil {
						s.log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, peerAddr, err.Error())
						return false
					}

					// Read again.
					txn, err = s.ds.NewTransaction(ctx, false)
					if err != nil {
						release()
						s.log.Warnf("Fail to start new transaction: %v", err.Error())
						return false
					}
					defer txn.Discard(context.Background())

					exists, err = txn.Has(ctx, currencyID, peerAddr, chAddr)
					if err != nil {
						release()
						s.log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, peerAddr, chAddr, err.Error())
						return false
					}
					if !exists {
						// The entry does not exist, discard this operation.
						release()
						s.log.Debugf("Operation discarded as %v-%v-%v does not exist", currencyID, peerAddr, chAddr)
						return true
					}

					// This entry still does exist.
					err = txn.Delete(ctx, currencyID, peerAddr, chAddr)
					if err != nil {
						release()
						s.log.Warnf("Fail to delete %v-%v-%v: %v", currencyID, peerAddr, chAddr, err.Error())
						return false
					}

					// Check if parent is empty
					children, err := txn.GetChildren(ctx, currencyID, peerAddr)
					if err != nil {
						release()
						s.log.Warnf("Fail to get children for %v-%v: %v", currencyID, peerAddr, err.Error())
						return false
					}

					err = txn.Commit(ctx)
					if err != nil {
						release()
						s.log.Warnf("Fail to commit transaction: %v", err.Error())
						return false
					}
					// Succeed.
					release()
					// Check if parent is empty
					if len(children) == 0 {
						release, err = s.ds.Lock(ctx, currencyID)
						if err != nil {
							s.log.Warnf("Fail to obtain write lock to remove empty %v-%v: %v", currencyID, peerAddr, err.Error())
							return true
						}
						defer release()

						txn, err := s.ds.NewTransaction(ctx, false)
						if err != nil {
							s.log.Warnf("Fail to start a new transaction to remove empty %v-%v: %v", currencyID, peerAddr, err.Error())
							return true
						}
						defer txn.Discard(context.Background())

						children, err = txn.GetChildren(ctx, currencyID, peerAddr)
						if err != nil {
							s.log.Warnf("Fail to get children for %v-%v: %v", currencyID, peerAddr, err.Error())
							return true
						}
						if len(children) == 0 {
							// Note is still empty.
							err = txn.Delete(ctx, currencyID, peerAddr)
							if err != nil {
								s.log.Warnf("Fail to remove empty %v-%v: %v", currencyID, peerAddr, err.Error())
								return true
							}
							err = txn.Commit(ctx)
							if err != nil {
								s.log.Warnf("Fail to commit transaction to remove empty %v-%v: %v", currencyID, peerAddr, err.Error())
								return true
							}
						}
					}
					return true
				}()
				if succeed {
					return
				}
			}
			// Failed after s.dsRetry times, logging to error.
			// The logging can be later used to repair the ds.
			s.log.Errorf("Fail to remove %v-%v-%v", currencyID, peerAddr, chAddr)
		})
	}()
}

// Read is used to read a channel state.
//
// @input - context, currency id, peer address, channel address.
//
// @output - channel state, error.
func (s *PaychStoreImpl) Read(ctx context.Context, currencyID byte, peerAddr string, chAddr string) (paychstate.State, error) {
	s.log.Debugf("Read channel state for %v-%v-%v", currencyID, peerAddr, chAddr)
	// Read
	release, err := s.ds.RLock(ctx, currencyID, peerAddr, chAddr)
	if err != nil {
		s.log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, peerAddr, chAddr, err.Error())
		return paychstate.State{}, err
	}
	defer release()

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		s.log.Warnf("Fail to start new transaction: %v", currencyID, peerAddr, chAddr)
		return paychstate.State{}, err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, peerAddr, chAddr)
	if err != nil {
		s.log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, peerAddr, chAddr, err.Error())
		return paychstate.State{}, err
	}
	if !exists {
		s.log.Debugf("Channel %v-%v-%v does not exist", currencyID, peerAddr, chAddr)
		return paychstate.State{}, fmt.Errorf("channel %v-%v-%v does not exist", currencyID, peerAddr, chAddr)
	}

	data, err := txn.Get(ctx, currencyID, peerAddr, chAddr)
	if err != nil {
		s.log.Warnf("Fail to read ds value for %v-%v-%v: %v", currencyID, peerAddr, chAddr, err.Error())
		return paychstate.State{}, err
	}
	res := paychstate.State{}
	err = res.Decode(data)
	if err != nil {
		s.log.Errorf("Fail to decode data %v: %v", data, err.Error())
		return paychstate.State{}, err
	}
	return res, nil
}

// ListCurrencyIDs is used to list all currencies.
//
// @input - context.
//
// @output - currency id chan out, error chan out.
func (s *PaychStoreImpl) ListCurrencyIDs(ctx context.Context) (<-chan byte, <-chan error) {
	out := make(chan byte, 32)
	errChan := make(chan error, 1)
	go func() {
		s.log.Debugf("Listing currency ids")
		defer func() {
			close(out)
			close(errChan)
		}()
		release, err := s.ds.RLock(ctx)
		if err != nil {
			errChan <- err
			s.log.Debugf("Fail to obtain read lock for root: %v", err.Error())
			return
		}

		txn, err := s.ds.NewTransaction(ctx, true)
		if err != nil {
			release()
			errChan <- err
			s.log.Warnf("Fail to start a new transaction: %v", err.Error())
			return
		}
		defer txn.Discard(context.Background())

		children, err := txn.GetChildren(ctx)
		if err != nil {
			release()
			errChan <- err
			s.log.Warnf("Fail to get children for root: %v", err.Error())
			return
		}
		release()

		for child := range children {
			currencyID, err := strconv.Atoi(child)
			if err != nil {
				errChan <- err
				s.log.Errorf("Fail to convert child key %v to currency id: %v", child, err.Error())
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

// ListPeers is used to list all peers.
//
// @input - context, currency id.
//
// @output - peer address chan out, error chan out.
func (s *PaychStoreImpl) ListPeers(ctx context.Context, currencyID byte) (<-chan string, <-chan error) {
	out := make(chan string, 128)
	errChan := make(chan error, 1)
	go func() {
		s.log.Debugf("Listing peers for %v", currencyID)
		defer func() {
			close(out)
			close(errChan)
		}()
		release, err := s.ds.RLock(ctx, currencyID)
		if err != nil {
			errChan <- err
			s.log.Debugf("Fail to obtain read lock for %v: %v", currencyID, err.Error())
			return
		}

		txn, err := s.ds.NewTransaction(ctx, true)
		if err != nil {
			release()
			errChan <- err
			s.log.Warnf("Fail to start a new transaction: %v", err.Error())
			return
		}
		defer txn.Discard(context.Background())

		exists, err := txn.Has(ctx, currencyID)
		if err != nil {
			release()
			errChan <- err
			s.log.Warnf("Fail to check if contains %v: %v", currencyID, err.Error())
			return
		}
		if !exists {
			release()
			s.log.Debugf("Currency id %v does not exist", currencyID)
			return
		}
		children, err := txn.GetChildren(ctx, currencyID)
		if err != nil {
			release()
			errChan <- err
			s.log.Warnf("Fail to get children for %v: %v", currencyID, err.Error())
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

// List channels by peer.
//
// @input - context, currency id, peer address.
//
// @output - channel address chan out, error chan out.
func (s *PaychStoreImpl) ListPaychsByPeer(ctx context.Context, currencyID byte, peerAddr string) (<-chan string, <-chan error) {
	out := make(chan string, 128)
	errChan := make(chan error, 1)
	go func() {
		s.log.Debugf("Listing paychs for %v-%v", currencyID, peerAddr)
		defer func() {
			close(out)
			close(errChan)
		}()
		release, err := s.ds.RLock(ctx, currencyID, peerAddr)
		if err != nil {
			errChan <- err
			s.log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, peerAddr, err.Error())
			return
		}

		txn, err := s.ds.NewTransaction(ctx, true)
		if err != nil {
			release()
			errChan <- err
			s.log.Warnf("Fail to start a new transaction: %v", err.Error())
			return
		}
		defer txn.Discard(context.Background())

		exists, err := txn.Has(ctx, currencyID, peerAddr)
		if err != nil {
			release()
			errChan <- err
			s.log.Warnf("Fail to check if contains %v-%v: %v", currencyID, peerAddr, err.Error())
			return
		}
		if !exists {
			release()
			s.log.Debugf("Peer %v-%v does not exist", currencyID, peerAddr)
			return
		}
		children, err := txn.GetChildren(ctx, currencyID, peerAddr)
		if err != nil {
			release()
			errChan <- err
			s.log.Warnf("Fail to get children for %v-%v: %v", currencyID, peerAddr, err.Error())
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

// processQueue processes operation from the queue one by one.
func (s *PaychStoreImpl) processQueue() {
	for {
		// This queue will never be locked, so it will not return error.
		op, err := s.queue.DequeueOrWaitForNextElementContext(s.ctx)
		if err != nil {
			if err == context.Canceled {
				s.log.Infof("Message queue shutdown...")
				// Shutdown of message queue.
				return
			} else {
				s.log.Errorf("Received error from queue, it should never happen, operation skipped: %v", err.Error())
			}
		} else {
			op.(func())()
		}
	}
}
