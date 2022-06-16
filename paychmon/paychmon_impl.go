package paychmon

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
	"sync"
	"time"

	gcq "github.com/enriquebris/goconcurrentqueue"
	"github.com/wcgcyx/fcr/locking"
	"github.com/wcgcyx/fcr/mdlstore"
	"github.com/wcgcyx/fcr/paymgr"
	"github.com/wcgcyx/fcr/trans"
)

// PaychMonitorImpl implements the PaychMonitor interface.
type PaychMonitorImpl struct {
	// Payment manager
	pam paymgr.PaymentManager

	// Transactor
	transactor trans.Transactor

	// Process related.
	routineCtx context.Context

	// Store of payment channel settlements.
	store mdlstore.MultiDimLockableStore

	// Queue related
	queueCtx context.Context
	wg       sync.WaitGroup
	queue    gcq.Queue

	// Cache of payment channel settlements control channels.
	cache     map[bool]map[byte]map[string]*control
	cacheLock *locking.LockNode

	// Shutdown function.
	shutdown func()

	// Options
	checkFreq time.Duration
}

// NewPaychMonitorImpl creates a new PaychMonitorImpl.
//
// @input - context, payment manager, transactor, options.
//
// @output - paych monitor, error.
func NewPaychMonitorImpl(ctx context.Context, pam paymgr.PaymentManager, transactor trans.Transactor, opts Opts) (*PaychMonitorImpl, error) {
	// Parse options
	checkFreq := opts.CheckFreq
	if checkFreq == 0 {
		checkFreq = defaultCheckFreq
	}
	log.Infof("Start paych monitor...")
	// Open store.
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, opts.Path)
	if err != nil {
		log.Errorf("Fail to open mdlstore: %v", err.Error())
		return nil, err
	}
	defer func() {
		if err != nil {
			log.Infof("Fail to start paych monitor, close ds...")
			err0 := ds.Shutdown(context.Background())
			if err0 != nil {
				log.Errorf("Fail to close ds after failing to start paych monitor: %v", err0.Error())
			}
		}
	}()
	// Create queue.
	queue := gcq.NewFIFO()
	wg := sync.WaitGroup{}
	queueCtx, cancel0 := context.WithCancel(context.Background())
	defer func() {
		if err != nil {
			log.Infof("Fail to start paych monitor, shutdown queue...")
			wg.Wait()
			cancel0()
		}
	}()
	// Create routines
	routineCtx, cancel1 := context.WithCancel(context.Background())
	defer func() {
		if err != nil {
			log.Infof("Fail to start paych monitor, shutdown started routines...")
			cancel1()
		}
	}()
	m := PaychMonitorImpl{
		pam:        pam,
		transactor: transactor,
		routineCtx: routineCtx,
		store:      ds,
		queueCtx:   queueCtx,
		wg:         wg,
		queue:      queue,
		cache:      make(map[bool]map[byte]map[string]*control),
		cacheLock:  locking.CreateLockNode(),
		checkFreq:  checkFreq,
	}
	go m.processQueue()
	// Try to list and start tracking channels.
	release, err := ds.RLock(ctx)
	if err != nil {
		log.Errorf("Fail to obtain read lock for root: %v", err.Error())
		return nil, err
	}
	defer release()

	txn, err := ds.NewTransaction(ctx, true)
	if err != nil {
		log.Errorf("Fail to start new transaction: %v", err.Error())
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
			dim, outbound, currencyID, chAddr, err := decDSKey(key)
			if err != nil {
				log.Errorf("Fail to decode ds key %v: %v", key, err.Error())
				return nil, err
			}
			if dim != 3 {
				continue
			}
			val, err := txn.Get(ctx, outbound, currencyID, chAddr)
			if err != nil {
				log.Errorf("Fail to read ds value for %v-%v-%v: %v", outbound, currencyID, chAddr, err.Error())
				return nil, err
			}
			_, settlement, err := decDSVal(val)
			if err != nil {
				log.Errorf("Fail to decode ds value %v: %v", val, err.Error())
				return nil, err
			}
			// Check channel state.
			_, _, _, _, _, fromAddr, toAddr, err := m.transactor.Check(ctx, currencyID, chAddr)
			if err != nil {
				log.Errorf("Fail to check channel state for %v-%v: %v", currencyID, chAddr, err.Error())
				return nil, err
			}
			var peerAddr string
			if outbound {
				peerAddr = toAddr
			} else {
				peerAddr = fromAddr
			}
			_, ok := m.cache[outbound]
			if !ok {
				m.cache[outbound] = make(map[byte]map[string]*control)
			}
			_, ok = m.cache[outbound][currencyID]
			if !ok {
				m.cache[outbound][currencyID] = make(map[string]*control)
			}
			renew := make(chan time.Time, 1)
			retire := make(chan bool, 1)
			m.cache[outbound][currencyID][chAddr] = &control{
				renew:  renew,
				retire: retire,
			}
			m.cacheLock.Insert(outbound, currencyID, chAddr)
			defer func() {
				if err == nil {
					go m.trackRoutine(outbound, currencyID, peerAddr, chAddr, settlement, renew, retire)
				}
			}()
		}
	}
	m.shutdown = func() {
		log.Infof("Stop and finalising all routines...")
		cancel1()
		log.Infof("Shutdown queue...")
		wg.Wait()
		cancel0()
		log.Infof("Stop datastore...")
		err := ds.Shutdown(context.Background())
		if err != nil {
			log.Errorf("Fail to stop datastore: %v", err.Error())
		}
	}
	return &m, nil
}

// Shutdown safely shuts down the component.
func (m *PaychMonitorImpl) Shutdown() {
	log.Infof("Start shutdown...")
	m.shutdown()
}

// addToQueue adds a ds operation to the queue.
//
// @input - ds operation.
func (m *PaychMonitorImpl) addToQueue(op func()) {
	m.wg.Add(1)
	go func() {
		m.queue.Enqueue(func() {
			defer m.wg.Done()
			op()
		})
	}()
}

// processQueue processes operation from the queue one by one.
func (m *PaychMonitorImpl) processQueue() {
	for {
		// This queue will never be locked, so it will not return error.
		op, err := m.queue.DequeueOrWaitForNextElementContext(m.queueCtx)
		if err != nil {
			if err == context.Canceled {
				log.Infof("Message queue shutdown...")
				// Shutdown of message queue.
				return
			} else {
				log.Errorf("Received error from queue, it should never happen, operation skipped: %v", err.Error())
			}
		} else {
			op.(func())()
		}
	}
}

// Track starts tracking a payment channel.
//
// @input - context, if it is outbound channel, currency id, peer address, channel address, settlement time.
//
// @output - error.
func (m *PaychMonitorImpl) Track(ctx context.Context, outbound bool, currencyID byte, chAddr string, settlement time.Time) error {
	log.Debugf("Start tracking %v-%v-%v with settlement %v", outbound, currencyID, chAddr, settlement)
	// Check channel state.
	_, _, _, _, _, fromAddr, toAddr, err := m.transactor.Check(ctx, currencyID, chAddr)
	if err != nil {
		return err
	}
	var peerAddr string
	if outbound {
		peerAddr = toAddr
	} else {
		peerAddr = fromAddr
	}

	// Check if exists.
	release, err := m.cacheLock.RLock(ctx, outbound, currencyID, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", outbound, currencyID, chAddr, err.Error())
		return err
	}
	_, ok := m.cache[outbound]
	if ok {
		_, ok = m.cache[outbound][currencyID]
		if ok {
			_, ok = m.cache[outbound][currencyID][chAddr]
			if ok {
				release()
				log.Debugf("channel %v-%v-%v exists", outbound, currencyID, chAddr)
				return nil
			}
		}
	}
	release()

	release, err = m.cacheLock.Lock(ctx, outbound, currencyID, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v: %v", outbound, currencyID, chAddr, err.Error())
		return err
	}
	defer release()

	// Check again
	_, ok = m.cache[outbound]
	if ok {
		_, ok = m.cache[outbound][currencyID]
		if ok {
			_, ok = m.cache[outbound][currencyID][chAddr]
			if ok {
				log.Debugf("channel %v-%v-%v exists", outbound, currencyID, chAddr)
				return nil
			}
		}
	} else {
		m.cache[outbound] = make(map[byte]map[string]*control)
	}
	_, ok = m.cache[outbound][currencyID]
	if !ok {
		m.cache[outbound][currencyID] = make(map[string]*control)
	}
	renew := make(chan time.Time, 1)
	retire := make(chan bool, 1)
	m.cache[outbound][currencyID][chAddr] = &control{
		renew:  renew,
		retire: retire,
	}
	m.cacheLock.Insert(outbound, currencyID, chAddr)
	// Add to DS
	m.addToQueue(func() {
		now := time.Now()
		release, err := m.store.Lock(m.routineCtx, outbound, currencyID, chAddr)
		if err != nil {
			log.Warnf("Fail to obtain write lock for %v-%v-%v with updated %v and settlement %v: %v", outbound, currencyID, chAddr, now, settlement, err.Error())
			return
		}
		defer release()

		txn, err := m.store.NewTransaction(m.routineCtx, false)
		if err != nil {
			log.Warnf("Fail to start new transaction to insert channel %v-%v-%v with updated %v and settlement: %v", outbound, currencyID, chAddr, now, settlement, err.Error())
			return
		}
		defer txn.Discard(context.Background())

		dsVal, err := encDSVal(now, settlement)
		if err != nil {
			log.Errorf("Fail to encode data to insert channel %v-%v-%v with updated %v and settlement: %v", outbound, currencyID, chAddr, now, settlement, err.Error())
			return
		}
		err = txn.Put(m.routineCtx, dsVal, outbound, currencyID, chAddr)
		if err != nil {
			log.Warnf("Fail to put value for channel %v-%v-%v with updated %v and settlement %v into store: %v", outbound, currencyID, chAddr, now, settlement, err.Error())
			return
		}
		err = txn.Commit(m.routineCtx)
		if err != nil {
			log.Warnf("Fail to commit insertion of channel %v-%v-%v with updated %v and settlement %v into store: %v", outbound, currencyID, chAddr, now, settlement, err.Error())
			return
		}
	})
	// Start tracking
	go m.trackRoutine(outbound, currencyID, peerAddr, chAddr, settlement, renew, retire)
	return nil
}

// Check checks the current settlement of a payment channel.
//
// @input - context, if it is outbound channel, currency id, channel address.
//
// @output - last update time, settlement time, error.
func (m *PaychMonitorImpl) Check(ctx context.Context, outbound bool, currencyID byte, chAddr string) (time.Time, time.Time, error) {
	log.Debugf("Check current settlement for %v-%v-%v", outbound, currencyID, chAddr)
	// Check if exists.
	release, err := m.store.RLock(ctx, outbound, currencyID, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", outbound, currencyID, chAddr, err.Error())
		return time.Time{}, time.Time{}, err
	}
	defer release()

	txn, err := m.store.NewTransaction(ctx, true)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return time.Time{}, time.Time{}, err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, outbound, currencyID, chAddr)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v-%v: %v", outbound, currencyID, chAddr, err.Error())
		return time.Time{}, time.Time{}, err
	}
	if !exists {
		return time.Time{}, time.Time{}, fmt.Errorf("channel %v-%v-%v does not exist", outbound, currencyID, chAddr)
	}
	val, err := txn.Get(ctx, outbound, currencyID, chAddr)
	if err != nil {
		log.Warnf("Fail to read the ds value for %v-%v-%v: %v", outbound, currencyID, chAddr, err.Error())
		return time.Time{}, time.Time{}, err
	}
	updated, settleAt, err := decDSVal(val)
	if err != nil {
		log.Errorf("Fail to decode ds value %v, this should never happen: %v", val, err.Error())
		return time.Time{}, time.Time{}, err
	}
	return updated, settleAt, nil
}

// Renew renews a payment channel.
//
// @input - context, if it is outbound channel, currency id, channel address, new settlement time.
//
// @output - error.
func (m *PaychMonitorImpl) Renew(ctx context.Context, outbound bool, currencyID byte, chAddr string, newSettlement time.Time) error {
	log.Debugf("Renew payment channel for %v-%v-%v with new settlement of %v", outbound, currencyID, chAddr, newSettlement)
	// Check if exists.
	release, err := m.cacheLock.RLock(ctx, outbound, currencyID, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", outbound, currencyID, chAddr, err.Error())
		return err
	}
	_, ok := m.cache[outbound]
	if !ok {
		release()
		return fmt.Errorf("outbound %v does not exist", outbound)
	}
	_, ok = m.cache[outbound][currencyID]
	if !ok {
		release()
		return fmt.Errorf("currency id %v-%v does not exist", outbound, currencyID)
	}
	_, ok = m.cache[outbound][currencyID][chAddr]
	if !ok {
		release()
		return fmt.Errorf("channel %v-%v-%v does not exist", outbound, currencyID, chAddr)
	}
	release()

	release, err = m.cacheLock.Lock(ctx, outbound, currencyID, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v: %v", outbound, currencyID, chAddr, err.Error())
		return err
	}
	defer release()

	// Check again
	_, ok = m.cache[outbound]
	if !ok {
		return fmt.Errorf("outbound %v does not exist", outbound)
	}
	_, ok = m.cache[outbound][currencyID]
	if !ok {
		return fmt.Errorf("currency id %v-%v does not exist", outbound, currencyID)
	}
	_, ok = m.cache[outbound][currencyID][chAddr]
	if !ok {
		return fmt.Errorf("channel %v-%v-%v does not exist", outbound, currencyID, chAddr)
	}
	// Add to DS
	m.addToQueue(func() {
		now := time.Now()
		release, err := m.store.Lock(m.routineCtx, outbound, currencyID, chAddr)
		if err != nil {
			log.Warnf("Fail to obtain write lock for %v-%v-%v with updated %v and settlement %v: %v", outbound, currencyID, chAddr, now, newSettlement, err.Error())
			return
		}
		defer release()

		txn, err := m.store.NewTransaction(ctx, false)
		if err != nil {
			log.Warnf("Fail to start new transaction to insert channel %v-%v-%v with updated %v and settlement: %v", outbound, currencyID, chAddr, now, newSettlement, err.Error())
			return
		}
		defer txn.Discard(context.Background())

		dsVal, err := encDSVal(now, newSettlement)
		if err != nil {
			log.Errorf("Fail to encode data to insert channel %v-%v-%v with updated %v and settlement: %v", outbound, currencyID, chAddr, now, newSettlement, err.Error())
			return
		}
		err = txn.Put(ctx, dsVal, outbound, currencyID, chAddr)
		if err != nil {
			log.Warnf("Fail to put value for channel %v-%v-%v with updated %v and settlement %v into store: %v", outbound, currencyID, chAddr, now, newSettlement, err.Error())
			return
		}
		err = txn.Commit(ctx)
		if err != nil {
			log.Warnf("Fail to commit insertion of channel %v-%v-%v with updated %v and settlement %v into store: %v", outbound, currencyID, chAddr, now, newSettlement, err.Error())
			return
		}
	})
	// Update new settlement
	m.cache[outbound][currencyID][chAddr].renew <- newSettlement
	return nil
}

// Retire retires a payment channel.
//
// @input - context, if it is outbound channel, currency id, channel address.
//
// @output - error.
func (m *PaychMonitorImpl) Retire(ctx context.Context, outbound bool, currencyID byte, chAddr string) error {
	log.Debugf("Retire payment channel for %v%v-%v", outbound, currencyID, chAddr)
	// Check if exists.
	release, err := m.cacheLock.RLock(ctx, outbound, currencyID, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", outbound, currencyID, chAddr, err.Error())
		return err
	}
	defer release()
	_, ok := m.cache[outbound]
	if !ok {
		return fmt.Errorf("outbound %v does not exist", outbound)
	}
	_, ok = m.cache[outbound][currencyID]
	if !ok {
		return fmt.Errorf("currency id %v-%v does not exist", outbound, currencyID)
	}
	_, ok = m.cache[outbound][currencyID][chAddr]
	if !ok {
		return fmt.Errorf("channel %v-%v-%v does not exist", outbound, currencyID, chAddr)
	}
	m.cache[outbound][currencyID][chAddr].retire <- true
	return nil
}
