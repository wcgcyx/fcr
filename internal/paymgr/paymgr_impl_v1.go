package paymgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"fmt"
	"math/big"
	"path/filepath"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	badgerds "github.com/ipfs/go-ds-badger"
	logging "github.com/ipfs/go-log"
	golock "github.com/viney-shih/go-lock"
	"github.com/wcgcyx/fcr/internal/crypto"
	"github.com/wcgcyx/fcr/internal/paymgr/chainmgr"
	"github.com/wcgcyx/fcr/internal/payoffer"
	"github.com/wcgcyx/fcr/internal/payofferstore"
)

const (
	activeOutChStoreSubPath = "activeoutchstore"
	outChStoreSubPath       = "outchstore"
	activeInChStoreSubPath  = "activeinchstore"
	inChStoreSubPath        = "inchstore"
	recordedStoreSubPath    = "recordedstore"
	refreshInterval         = 1 * time.Hour
	maxInactivityForSelf    = time.Minute
	lockInterval            = 1 * time.Millisecond
	lockAttempt             = 10
)

var log = logging.Logger("paymgr")

type PaymentManagerImplV1 struct {
	// Parent context
	ctx context.Context
	// Roots
	roots map[uint64]string
	// Chain managers
	cms map[uint64]chainmgr.ChainManager
	// Offer store
	os payofferstore.PayOfferStore
	// Keystore
	ks datastore.Datastore
	// Outbound channel store
	outChStoresLock   map[uint64]*sync.RWMutex
	outChActiveStores map[uint64]datastore.Datastore
	outChStores       map[uint64]datastore.Datastore
	outChLocks        map[uint64]map[string]golock.RWMutex
	// Inbound channel store
	inChStoresLock   map[uint64]*sync.RWMutex
	inChActiveStores map[uint64]datastore.Datastore
	inChStores       map[uint64]datastore.Datastore
	inChLocks        map[uint64]map[string]golock.RWMutex
	// Recorded receive
	recordedStoresLock  map[uint64]*sync.RWMutex
	recordedStores      map[uint64]datastore.Datastore
	recordedEntriesLock map[uint64]map[string]*sync.RWMutex
}

// NewPaymentManagerImplV1 creates a PaymentManager.
// It takes a context, a root db path, a list of currency ids, keystore, offerstore as arguments.
// It returns a peer manager and error.
func NewPaymentManagerImplV1(ctx context.Context, dbPath string, currencyIDs []uint64, cms map[uint64]chainmgr.ChainManager, ks datastore.Datastore, os payofferstore.PayOfferStore) (PaymentManager, error) {
	mgr := &PaymentManagerImplV1{
		ctx:                 ctx,
		roots:               make(map[uint64]string),
		cms:                 make(map[uint64]chainmgr.ChainManager),
		os:                  os,
		ks:                  ks,
		outChStoresLock:     make(map[uint64]*sync.RWMutex),
		outChActiveStores:   make(map[uint64]datastore.Datastore),
		outChStores:         make(map[uint64]datastore.Datastore),
		outChLocks:          make(map[uint64]map[string]golock.RWMutex),
		inChStoresLock:      make(map[uint64]*sync.RWMutex),
		inChActiveStores:    make(map[uint64]datastore.Datastore),
		inChStores:          make(map[uint64]datastore.Datastore),
		inChLocks:           make(map[uint64]map[string]golock.RWMutex),
		recordedStoresLock:  make(map[uint64]*sync.RWMutex),
		recordedStores:      make(map[uint64]datastore.Datastore),
		recordedEntriesLock: make(map[uint64]map[string]*sync.RWMutex),
	}
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	for _, currencyID := range currencyIDs {
		prv, err := mgr.ks.Get(datastore.NewKey(fmt.Sprintf("%v", currencyID)))
		if err != nil {
			return nil, err
		}
		root, err := crypto.GetAddress(prv)
		if err != nil {
			return nil, err
		}
		cm, ok := cms[currencyID]
		if !ok {
			return nil, fmt.Errorf("chain manager for currency id %v not supported", currencyID)
		}
		mgr.roots[currencyID] = root
		mgr.cms[currencyID] = cm
		store, err := badgerds.NewDatastore(filepath.Join(dbPath, fmt.Sprintf("%v-%v", activeOutChStoreSubPath, currencyID)), &dsopts)
		if err != nil {
			return nil, err
		}
		mgr.outChActiveStores[currencyID] = store
		mgr.outChStoresLock[currencyID] = &sync.RWMutex{}
		store, err = badgerds.NewDatastore(filepath.Join(dbPath, fmt.Sprintf("%v-%v", outChStoreSubPath, currencyID)), &dsopts)
		if err != nil {
			return nil, err
		}
		chLocks := make(map[string]golock.RWMutex)
		q := dsq.Query{KeysOnly: true}
		tempRes, err := store.Query(q)
		if err != nil {
			return nil, err
		}
		output := make(chan string, dsq.KeysOnlyBufSize)

		go func() {
			defer func() {
				tempRes.Close()
				close(output)
			}()
			for {
				e, ok := tempRes.NextSync()
				if !ok {
					break
				}
				if e.Error != nil {
					log.Errorf("error querying from storage: %v", e.Error.Error())
					return
				}
				select {
				case <-ctx.Done():
					return
				case output <- e.Key[1:]:
				}
			}
		}()
		for chAddr := range output {
			var rwMut golock.RWMutex = golock.NewCASMutex()
			chLocks[chAddr] = rwMut
		}
		mgr.outChStores[currencyID] = store
		mgr.outChLocks[currencyID] = chLocks

		store, err = badgerds.NewDatastore(filepath.Join(dbPath, fmt.Sprintf("%v-%v", activeInChStoreSubPath, currencyID)), &dsopts)
		if err != nil {
			return nil, err
		}
		mgr.inChActiveStores[currencyID] = store
		mgr.inChStoresLock[currencyID] = &sync.RWMutex{}
		store, err = badgerds.NewDatastore(filepath.Join(dbPath, fmt.Sprintf("%v-%v", inChStoreSubPath, currencyID)), &dsopts)
		if err != nil {
			return nil, err
		}
		chLocks = make(map[string]golock.RWMutex)
		q = dsq.Query{KeysOnly: true}
		tempRes, err = store.Query(q)
		if err != nil {
			return nil, err
		}
		output = make(chan string, dsq.KeysOnlyBufSize)

		go func() {
			defer func() {
				tempRes.Close()
				close(output)
			}()
			for {
				e, ok := tempRes.NextSync()
				if !ok {
					break
				}
				if e.Error != nil {
					log.Errorf("error querying from storage: %v", e.Error.Error())
					return
				}
				select {
				case <-ctx.Done():
					return
				case output <- e.Key[1:]:
				}
			}
		}()
		for chAddr := range output {
			var rwMut golock.RWMutex = golock.NewCASMutex()
			chLocks[chAddr] = rwMut
		}
		mgr.inChStores[currencyID] = store
		mgr.inChLocks[currencyID] = chLocks

		store, err = badgerds.NewDatastore(filepath.Join(dbPath, fmt.Sprintf("%v-%v", recordedStoreSubPath, currencyID)), &dsopts)
		if err != nil {
			return nil, err
		}
		mgr.recordedStores[currencyID] = store
		mgr.recordedStoresLock[currencyID] = &sync.RWMutex{}
		recordedLocks := make(map[string]*sync.RWMutex)
		q = dsq.Query{KeysOnly: true}
		tempRes, err = store.Query(q)
		if err != nil {
			return nil, err
		}
		output = make(chan string, dsq.KeysOnlyBufSize)

		go func() {
			defer func() {
				tempRes.Close()
				close(output)
			}()
			for {
				e, ok := tempRes.NextSync()
				if !ok {
					break
				}
				if e.Error != nil {
					log.Errorf("error querying from storage: %v", e.Error.Error())
					return
				}
				select {
				case <-ctx.Done():
					return
				case output <- e.Key[1:]:
				}
			}
		}()
		for fromAddr := range output {
			recordedLocks[fromAddr] = &sync.RWMutex{}
		}
		mgr.recordedEntriesLock[currencyID] = recordedLocks
	}
	go mgr.gcRoutine()
	go mgr.shutdownRoutine()
	return mgr, nil
}

// GetPrvKey gets the private key associated of given currency id.
// It takes the currency id as the argument.
// It returns the private key bytes and error if not existed.
func (mgr *PaymentManagerImplV1) GetPrvKey(currencyID uint64) ([]byte, error) {
	return mgr.ks.Get(datastore.NewKey(fmt.Sprintf("%v", currencyID)))
}

// GetRootAddress gets the root address of given currency id.
// It takes a currency id as the argument.
// It returns the root address and error if not existed.
func (mgr *PaymentManagerImplV1) GetRootAddress(currencyID uint64) (string, error) {
	root, ok := mgr.roots[currencyID]
	if !ok {
		return "", fmt.Errorf("currency id %v is not supported", currencyID)
	}
	return root, nil
}

// CreateOutboundCh creates an outbound channel to given address with given initial balance.
// It takes a context, currency id, to address, initial amount, and the settlement time as arguments.
// It returns the paych address and error.
func (mgr *PaymentManagerImplV1) CreateOutboundCh(ctx context.Context, currencyID uint64, toAddr string, amt *big.Int, settlement int64) (string, error) {
	log.Debugf("CreateOutboundCh currency %v, to %v, amt %v, settlement %v", currencyID, toAddr, amt.String(), settlement)
	activeStore, ok := mgr.outChActiveStores[currencyID]
	if !ok {
		return "", fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.outChStoresLock[currencyID]
	chStore := mgr.outChStores[currencyID]
	chLocks := mgr.outChLocks[currencyID]
	cm := mgr.cms[currencyID]
	prv, err := mgr.GetPrvKey(currencyID)
	if err != nil {
		return "", fmt.Errorf("error loading private key for currency id %v: %v", currencyID, err.Error())
	}
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	toKey := datastore.NewKey(toAddr)
	exists, err := activeStore.Has(toKey)
	if err != nil {
		storeLock.RUnlock()
		return "", fmt.Errorf("error checking storage: %v", err.Error())
	}
	if exists {
		// Existed, check if state pass settlement
		log.Debugf("existed, check if state pass settlement")
		data, err := activeStore.Get(toKey)
		if err != nil {
			storeLock.RUnlock()
			return "", fmt.Errorf("error reading storage: %v", err.Error())
		}
		chAddr := string(data)
		chLock := chLocks[chAddr]
		log.Debugf("Start RLock")
		if !tryRLock(chLock) {
			return "", fmt.Errorf("error locking channel after %v attempts", lockAttempt)
		}
		log.Debugf("Done RLock")
		chKey := datastore.NewKey(chAddr)
		data, err = chStore.Get(chKey)
		if err != nil {
			chLock.RUnlock()
			storeLock.RUnlock()
			return "", fmt.Errorf("error reading storage: %v", err.Error())
		}
		state := decodeOutChState(data)
		if time.Now().Unix() < state.settlement {
			log.Debugf("state still within settlement")
			chLock.RUnlock()
			storeLock.RUnlock()
			return "", fmt.Errorf("there is an active channel to %v in use", toAddr)
		}
		// Peoceed the execution, it will overwrite the active mapping
		log.Debugf("existed, state passes settlement, proceed")
		chLock.RUnlock()
	}
	// Not existed or just expired, create a channel
	chAddr, err := cm.Create(ctx, prv, toAddr, amt)
	if err != nil {
		storeLock.RUnlock()
		return "", fmt.Errorf("error creating channel to %v: %v", toAddr, err.Error())
	}
	storeLock.RUnlock()
	log.Debugf("Start store lock Lock")
	storeLock.Lock()
	log.Debugf("End store lock Lock")
	defer storeLock.Unlock()
	state := outChState{
		toAddr:       toAddr,
		chAddr:       chAddr,
		balance:      amt,
		redeemed:     big.NewInt(0),
		reserved:     big.NewInt(0),
		reservations: make(map[string]*reservation),
		nonce:        0,
		voucher:      "",
		settlingAt:   0,
		settlement:   settlement,
	}
	chKey := datastore.NewKey(chAddr)
	// Put it to store
	err = chStore.Put(chKey, encodeOutChState(&state))
	if err != nil {
		return "", fmt.Errorf("error writing to storage: %v", err.Error())
	}
	// Put it to active store
	err = activeStore.Put(toKey, []byte(chAddr))
	if err != nil {
		return "", fmt.Errorf("error writing to storage: %v", err.Error())
	}
	// New mutex
	var rwMut golock.RWMutex = golock.NewCASMutex()
	chLocks[chAddr] = rwMut
	return chAddr, nil
}

// TopupOutboundCh tops up an existing channel with some balance.
// It takes a context, currency id, to address, top up amount as arguments.
// It returns the error.
func (mgr *PaymentManagerImplV1) TopupOutboundCh(ctx context.Context, currencyID uint64, toAddr string, amt *big.Int) error {
	log.Debugf("TopupOutboundCh currency %v, to %v, amt %v", currencyID, toAddr, amt.String())
	activeStore, ok := mgr.outChActiveStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.outChStoresLock[currencyID]
	chStore := mgr.outChStores[currencyID]
	chLocks := mgr.outChLocks[currencyID]
	cm := mgr.cms[currencyID]
	prv, err := mgr.GetPrvKey(currencyID)
	if err != nil {
		return fmt.Errorf("error loading private key for currency id %v: %v", currencyID, err.Error())
	}
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	toKey := datastore.NewKey(toAddr)
	exists, err := activeStore.Has(toKey)
	if err != nil {
		storeLock.RUnlock()
		return fmt.Errorf("error checking storage: %v", err.Error())
	}
	if !exists {
		storeLock.RUnlock()
		return fmt.Errorf("there is no active channel to %v in use", toAddr)
	}
	// Existed, check if state pass settlement
	log.Debugf("existed, check if state pass settlement")
	data, err := activeStore.Get(toKey)
	if err != nil {
		storeLock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	chAddr := string(data)
	chLock := chLocks[chAddr]
	log.Debugf("Start RLock")
	if !tryRLock(chLock) {
		return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done RLock")
	chKey := datastore.NewKey(chAddr)
	data, err = chStore.Get(chKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	state := decodeOutChState(data)
	if time.Now().Unix() >= state.settlement {
		log.Debugf("state has passed settlement")
		storeLock.RUnlock()
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
		defer chLock.RUnlock()
		err = activeStore.Delete(toKey)
		if err != nil {
			return fmt.Errorf("error removing from storage: %v", err.Error())
		}
		return fmt.Errorf("channel to %v has just expired", toAddr)
	}
	defer storeLock.RUnlock()
	// Still within settlement time, topup
	log.Debugf("still within settlement time, topup")
	err = cm.Topup(ctx, prv, chAddr, amt)
	if err != nil {
		chLock.RUnlock()
		return fmt.Errorf("error topup the payment channel: %v", err.Error())
	}
	chLock.RUnlock()
	log.Debugf("Start Lock")
	if !tryLock(chLock) {
		return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done Lock")
	defer chLock.Unlock()
	// Read state again
	data, err = chStore.Get(chKey)
	if err != nil {
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	state = decodeOutChState(data)
	state.balance.Add(state.balance, amt)
	// Put it to store
	err = chStore.Put(chKey, encodeOutChState(state))
	if err != nil {
		return fmt.Errorf("error writing to storage: %v", err.Error())
	}
	return nil
}

// ListOutboundChs lists all the outbound channels (including those are settling.)
// It takes a context as the argument.
// It returns a map from currency id to list of paych addresses and error.
func (mgr *PaymentManagerImplV1) ListOutboundChs(ctx context.Context) (map[uint64][]string, error) {
	res := make(map[uint64][]string)
	for currencyID, chStore := range mgr.outChStores {
		storeLock := mgr.outChStoresLock[currencyID]
		log.Debugf("Start store lock RLock")
		storeLock.RLock()
		log.Debugf("End store lock RLock")
		q := dsq.Query{KeysOnly: true}
		tempRes, err := chStore.Query(q)
		if err != nil {
			storeLock.RUnlock()
			return nil, err
		}
		output := make(chan string, dsq.KeysOnlyBufSize)

		go func() {
			defer func() {
				tempRes.Close()
				close(output)
			}()
			for {
				e, ok := tempRes.NextSync()
				if !ok {
					break
				}
				if e.Error != nil {
					log.Errorf("error querying from storage: %v", e.Error.Error())
					return
				}
				select {
				case <-ctx.Done():
					return
				case output <- e.Key[1:]:
				}
			}
		}()
		for chAddr := range output {
			ids, ok := res[currencyID]
			if !ok {
				ids = make([]string, 0)
			}
			ids = append(ids, chAddr)
			res[currencyID] = ids
		}
		storeLock.RUnlock()
	}
	return res, nil
}

// ListActiveOutboundChs lists all active outbound channels.
// It takes a context as the argument.
// It returns a map from currency id to a list of channel addresses and error.
func (mgr *PaymentManagerImplV1) ListActiveOutboundChs(ctx context.Context) (map[uint64][]string, error) {
	res := make(map[uint64][]string)
	for currencyID, chStore := range mgr.outChActiveStores {
		storeLock := mgr.outChStoresLock[currencyID]
		log.Debugf("Start store lock RLock")
		storeLock.RLock()
		log.Debugf("End store lock RLock")
		q := dsq.Query{KeysOnly: true}
		tempRes, err := chStore.Query(q)
		if err != nil {
			storeLock.RUnlock()
			return nil, err
		}
		output := make(chan string, dsq.KeysOnlyBufSize)

		go func() {
			defer func() {
				tempRes.Close()
				close(output)
			}()
			for {
				e, ok := tempRes.NextSync()
				if !ok {
					break
				}
				if e.Error != nil {
					log.Errorf("error querying from storage: %v", e.Error.Error())
					return
				}
				select {
				case <-ctx.Done():
					return
				case output <- e.Key[1:]:
				}
			}
		}()
		for toAddr := range output {
			ids, ok := res[currencyID]
			if !ok {
				ids = make([]string, 0)
			}
			ids = append(ids, toAddr)
			res[currencyID] = ids
		}
		storeLock.RUnlock()
	}
	return res, nil
}

// InspectOutboundCh inspects an outbound payment channel.
// It takes a context, currency id and channel address as arguments.
// It returns recipient address, redeemed amount, total amount, settlement time of this channel (as specified when creating this channel),
// boolean indicates if channel is active, current chain height, settling height and error.
func (mgr *PaymentManagerImplV1) InspectOutboundCh(ctx context.Context, currencyID uint64, chAddr string) (string, *big.Int, *big.Int, int64, bool, int64, int64, error) {
	activeStore, ok := mgr.outChActiveStores[currencyID]
	if !ok {
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.outChStoresLock[currencyID]
	chStore := mgr.outChStores[currencyID]
	chLocks := mgr.outChLocks[currencyID]
	chLock, ok := chLocks[chAddr]
	if !ok {
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("channel %v does not exist", chAddr)
	}
	cm := mgr.cms[currencyID]
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	log.Debugf("Start RLock")
	if !tryRLock(chLock) {
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done RLock")
	chKey := datastore.NewKey(chAddr)
	data, err := chStore.Get(chKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error reading storage: %v", err.Error())
	}
	state := decodeOutChState(data)
	toKey := datastore.NewKey(state.toAddr)
	exists, err := activeStore.Has(toKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error checking storage: %v", err.Error())
	}
	active := false
	if exists {
		data, err = activeStore.Get(toKey)
		if err != nil {
			chLock.RUnlock()
			storeLock.RUnlock()
			return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error reading storage: %v", err.Error())
		}
		if chAddr == string(data) {
			active = true
		}
	}
	toRemove := false
	if active {
		// Active channel, check if this state is passing settlement
		if time.Now().Unix() >= state.settlement {
			log.Debugf("state has passed settlement")
			toRemove = true
			active = false
		}
	}
	// Check settling at
	log.Debugf("check settling at")
	settlingAt, curHeight, _, _, _, err := cm.Check(ctx, state.chAddr)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error checking paych status: %v", err.Error())
	}
	toUpdate := false
	if settlingAt != state.settlingAt {
		state.settlingAt = settlingAt
		toUpdate = true
	}
	chLock.RUnlock()
	if toUpdate {
		log.Debugf("Start Lock")
		if !tryLock(chLock) {
			return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error locking channel after %v attempts", lockAttempt)
		}
		log.Debugf("Done Lock")
		defer chLock.Unlock()
		settlingAt = state.settlingAt
		// Read state again
		data, err := chStore.Get(chKey)
		if err != nil {
			storeLock.RUnlock()
			return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error reading storage: %v", err.Error())
		}
		state = decodeOutChState(data)
		state.settlingAt = settlingAt
		err = chStore.Put(chKey, encodeOutChState(state))
		if err != nil {
			storeLock.RUnlock()
			return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error writing to storage: %v", err.Error())
		}
	}
	storeLock.RUnlock()
	if toRemove {
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
		err = activeStore.Delete(toKey)
		if err != nil {
			return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error removing from storage: %v", err.Error())
		}
	}
	return state.toAddr, state.redeemed, state.balance, state.settlement, active, curHeight, settlingAt, nil
}

// SettleOutboundCh settles an outbound payment channel.
// It takes a context, currency id and channel address as arguments.
// It returns the error.
func (mgr *PaymentManagerImplV1) SettleOutboundCh(ctx context.Context, currencyID uint64, chAddr string) error {
	log.Debugf("SettleOutboundCh currency %v channel %v", currencyID, chAddr)
	activeStore, ok := mgr.outChActiveStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.outChStoresLock[currencyID]
	chStore := mgr.outChStores[currencyID]
	chLocks := mgr.outChLocks[currencyID]
	chLock, ok := chLocks[chAddr]
	if !ok {
		return fmt.Errorf("channel %v does not exist", chAddr)
	}
	cm := mgr.cms[currencyID]
	prv, err := mgr.GetPrvKey(currencyID)
	if err != nil {
		return fmt.Errorf("error loading private key for currency id %v: %v", currencyID, err.Error())
	}
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	log.Debugf("Start RLock")
	if !tryRLock(chLock) {
		return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done RLock")
	chKey := datastore.NewKey(chAddr)
	data, err := chStore.Get(chKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	state := decodeOutChState(data)
	toKey := datastore.NewKey(state.toAddr)
	exists, err := activeStore.Has(toKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error checking storage: %v", err.Error())
	}
	active := false
	if exists {
		data, err = activeStore.Get(toKey)
		if err != nil {
			chLock.RUnlock()
			storeLock.RUnlock()
			return fmt.Errorf("error reading storage: %v", err.Error())
		}
		if chAddr == string(data) {
			active = true
		}
	}
	toRemove := false
	if active {
		toRemove = true
	}
	toUpdate := false
	if state.settlingAt == 0 {
		// Check settling at
		log.Debugf("check settling at")
		settlingAt, _, _, _, _, err := cm.Check(ctx, state.chAddr)
		if err != nil {
			chLock.RUnlock()
			storeLock.RUnlock()
			return fmt.Errorf("error checking paych status: %v", err.Error())
		}
		if settlingAt != 0 {
			state.settlingAt = settlingAt
		} else {
			err = cm.Settle(ctx, prv, state.chAddr, "")
			if err != nil {
				chLock.RUnlock()
				storeLock.RUnlock()
				return fmt.Errorf("error settling payment channel: %v", err.Error())
			}
			state.settlingAt, _, _, _, _, err = cm.Check(ctx, state.chAddr)
			if err != nil {
				chLock.RUnlock()
				storeLock.RUnlock()
				return fmt.Errorf("error checking paych status: %v", err.Error())
			}
		}
		toUpdate = true
	}
	chLock.RUnlock()
	if toUpdate {
		log.Debugf("Start Lock")
		if !tryLock(chLock) {
			return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
		}
		log.Debugf("Done Lock")
		defer chLock.Unlock()
		settlingAt := state.settlingAt
		// Read state again
		data, err := chStore.Get(chKey)
		if err != nil {
			storeLock.RUnlock()
			return fmt.Errorf("error reading storage: %v", err.Error())
		}
		state := decodeOutChState(data)
		state.settlingAt = settlingAt
		err = chStore.Put(chKey, encodeOutChState(state))
		if err != nil {
			storeLock.RUnlock()
			return fmt.Errorf("error writing to storage: %v", err.Error())
		}
	}
	storeLock.RUnlock()
	if toRemove {
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
		err = activeStore.Delete(toKey)
		if err != nil {
			return fmt.Errorf("error removing from storage: %v", err.Error())
		}
	}
	return nil
}

// CollectOutboundCh collects an outbound payment channel.
// It takes a context, currency id and channel address as arguments.
// It returns the error.
func (mgr *PaymentManagerImplV1) CollectOutboundCh(ctx context.Context, currencyID uint64, chAddr string) error {
	log.Debugf("CollectOutboundCh currency %v channel %v", currencyID, chAddr)
	activeStore, ok := mgr.outChActiveStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.outChStoresLock[currencyID]
	chStore := mgr.outChStores[currencyID]
	chLocks := mgr.outChLocks[currencyID]
	chLock, ok := chLocks[chAddr]
	if !ok {
		return fmt.Errorf("channel %v does not exist", chAddr)
	}
	cm := mgr.cms[currencyID]
	prv, err := mgr.GetPrvKey(currencyID)
	if err != nil {
		return fmt.Errorf("error loading private key for currency id %v: %v", currencyID, err.Error())
	}
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	log.Debugf("Start RLock")
	if !tryRLock(chLock) {
		return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done RLock")
	chKey := datastore.NewKey(chAddr)
	data, err := chStore.Get(chKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	state := decodeOutChState(data)
	toKey := datastore.NewKey(state.toAddr)
	exists, err := activeStore.Has(toKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error checking storage: %v", err.Error())
	}
	active := false
	if exists {
		data, err = activeStore.Get(toKey)
		if err != nil {
			chLock.RUnlock()
			storeLock.RUnlock()
			return fmt.Errorf("error reading storage: %v", err.Error())
		}
		if chAddr == string(data) {
			active = true
		}
	}
	toRemove := false
	if active {
		toRemove = true
	}
	if state.settlingAt == 0 {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("channel %v has not started settling process", chAddr)
	}
	_, curHeight, _, _, _, err := cm.Check(ctx, state.chAddr)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error checking paych status: %v", err.Error())
	}
	if curHeight < state.settlingAt {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("can not collect channel, settling at %v, current height %v", state.settlingAt, curHeight)
	}
	// Can collect
	log.Debugf("start collecting")
	err = cm.Collect(ctx, prv, chAddr)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error collecting channel: %v", err.Error())
	}
	chLock.RUnlock()
	log.Debugf("Start Lock")
	if !tryLock(chLock) {
		return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done Lock")
	defer chLock.Unlock()
	storeLock.RUnlock()
	log.Debugf("Start store lock Lock")
	storeLock.Lock()
	log.Debugf("End store lock Lock")
	defer storeLock.Unlock()
	if toRemove {
		err = activeStore.Delete(toKey)
		if err != nil {
			return fmt.Errorf("error removing from storage: %v", err.Error())
		}
	}
	err = chStore.Delete(chKey)
	if err != nil {
		return fmt.Errorf("error removing from storage: %v", err.Error())
	}
	delete(chLocks, chAddr)
	return nil
}

// AddInboundCh adds an inbound channel.
// It takes currency id, channel address and settlement time as arguments.
// It returns the error.
func (mgr *PaymentManagerImplV1) AddInboundCh(currencyID uint64, chAddr string, settlement int64) error {
	log.Debugf("AddInboundCh currrency %v channel %v settlement %v", currencyID, chAddr, settlement)
	cm, ok := mgr.cms[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	settlingAt, _, balance, fromAddr, toAddr, err := cm.Check(mgr.ctx, chAddr)
	if err != nil {
		return fmt.Errorf("error checking paych status: %v", err.Error())
	}
	if settlingAt != 0 {
		return fmt.Errorf("can not add paych with non-zero settling height, got %v", settlingAt)
	}
	if toAddr != mgr.roots[currencyID] {
		return fmt.Errorf("can not add paych, expect recipient %v, got %v", mgr.roots[currencyID], toAddr)
	}
	activeStore := mgr.inChActiveStores[currencyID]
	storeLock := mgr.inChStoresLock[currencyID]
	chStore := mgr.inChStores[currencyID]
	chLocks := mgr.inChLocks[currencyID]
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	fromKey := datastore.NewKey(fromAddr)
	existsActive, err := activeStore.Has(fromKey)
	if err != nil {
		storeLock.RUnlock()
		return fmt.Errorf("error checking storage: %v", err.Error())
	}
	chKey := datastore.NewKey(chAddr)
	existsCh, err := chStore.Has(chKey)
	if err != nil {
		storeLock.RUnlock()
		return fmt.Errorf("error checking storage: %v", err.Error())
	}
	storeLock.RUnlock()
	if !existsActive || !existsCh {
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
	}
	if !existsActive {
		err = activeStore.Put(fromKey, []byte(chAddr))
		if err != nil {
			return fmt.Errorf("error writing to storage: %v", err.Error())
		}
	}
	if !existsCh {
		state := inChState{
			fromAddr:    fromAddr,
			chAddr:      chAddr,
			balance:     balance,
			redeemed:    big.NewInt(0),
			offerStates: make(map[string]*offerState),
			nonce:       0,
			voucher:     "",
			settlingAt:  0,
			settlement:  settlement,
		}
		err = chStore.Put(chKey, encodeInChState(&state))
		if err != nil {
			return fmt.Errorf("error writing to storage: %v", err.Error())
		}
		// New mutex
		var rwMut golock.RWMutex = golock.NewCASMutex()
		chLocks[chAddr] = rwMut
	}
	return nil
}

// ListInboundChs lists all inbound channels.
// It takes a context as the argument.
// It returns a map from currency id to list of paych addresses and error.
func (mgr *PaymentManagerImplV1) ListInboundChs(ctx context.Context) (map[uint64][]string, error) {
	res := make(map[uint64][]string)
	for currencyID, chStore := range mgr.inChStores {
		storeLock := mgr.inChStoresLock[currencyID]
		log.Debugf("Start store lock RLock")
		storeLock.RLock()
		log.Debugf("End store lock RLock")
		q := dsq.Query{KeysOnly: true}
		tempRes, err := chStore.Query(q)
		if err != nil {
			storeLock.RUnlock()
			return nil, err
		}
		output := make(chan string, dsq.KeysOnlyBufSize)

		go func() {
			defer func() {
				tempRes.Close()
				close(output)
			}()
			for {
				e, ok := tempRes.NextSync()
				if !ok {
					break
				}
				if e.Error != nil {
					log.Errorf("error querying from storage: %v", e.Error.Error())
					return
				}
				select {
				case <-ctx.Done():
					return
				case output <- e.Key[1:]:
				}
			}
		}()
		for chAddr := range output {
			ids, ok := res[currencyID]
			if !ok {
				ids = make([]string, 0)
			}
			ids = append(ids, chAddr)
			res[currencyID] = ids
		}
		storeLock.RUnlock()
	}
	return res, nil
}

// InspectInboundCh inspects an inbound payment channel.
// It takes a context, currency id and channel address as arguments.
// It returns from address, redeemed amount, total amount, settlement time of this channel (as specified when creating this channel),
// boolean indicates if channel is active, boolean indicates if channel is settling, boolean indicates
// if channel can be collected (settled).
func (mgr *PaymentManagerImplV1) InspectInboundCh(ctx context.Context, currencyID uint64, chAddr string) (string, *big.Int, *big.Int, int64, bool, int64, int64, error) {
	activeStore, ok := mgr.inChActiveStores[currencyID]
	if !ok {
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.inChStoresLock[currencyID]
	chStore := mgr.inChStores[currencyID]
	chLocks := mgr.inChLocks[currencyID]
	chLock, ok := chLocks[chAddr]
	if !ok {
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("channel %v does not exist", chAddr)
	}
	cm := mgr.cms[currencyID]
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	log.Debugf("Start RLock")
	if !tryRLock(chLock) {
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done RLock")
	chKey := datastore.NewKey(chAddr)
	data, err := chStore.Get(chKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error reading storage: %v", err.Error())
	}
	state := decodeInChState(data)
	fromKey := datastore.NewKey(state.fromAddr)
	exists, err := activeStore.Has(fromKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error checking storage: %v", err.Error())
	}
	active := false
	if exists {
		data, err = activeStore.Get(fromKey)
		if err != nil {
			chLock.RUnlock()
			storeLock.RUnlock()
			return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error reading storage: %v", err.Error())
		}
		if chAddr == string(data) {
			active = true
		}
	}
	toRemove := false
	if active {
		// Active channel, check if this state is passing settlement
		if time.Now().Unix() >= state.settlement {
			log.Debugf("state has passed settlement")
			toRemove = true
			active = false
		}
	}
	// Check settling at
	log.Debugf("check settling at")
	settlingAt, curHeight, balance, _, _, err := cm.Check(ctx, state.chAddr)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error checking paych status: %v", err.Error())
	}
	toUpdate := false
	if settlingAt != state.settlingAt {
		state.settlingAt = settlingAt
		toUpdate = true
	}
	if balance.Cmp(state.balance) != 0 {
		state.balance = balance
		toUpdate = true
	}
	chLock.RUnlock()
	if toUpdate {
		log.Debugf("Start Lock")
		if !tryLock(chLock) {
			return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error locking channel after %v attempts", lockAttempt)
		}
		log.Debugf("Done Lock")
		defer chLock.Unlock()
		settlingAt := state.settlingAt
		// Read state again
		data, err := chStore.Get(chKey)
		if err != nil {
			storeLock.RUnlock()
			return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error reading storage: %v", err.Error())
		}
		state := decodeInChState(data)
		state.settlingAt = settlingAt
		err = chStore.Put(chKey, encodeInChState(state))
		if err != nil {
			storeLock.RUnlock()
			return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error writing to storage: %v", err.Error())
		}
	}
	storeLock.RUnlock()
	if toRemove {
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
		err = activeStore.Delete(fromKey)
		if err != nil {
			return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error removing from storage: %v", err.Error())
		}
	}
	return "", state.redeemed, state.balance, state.settlement, active, curHeight, settlingAt, nil
}

// SettleInboundCh settles an inbound payment channel.
// It takes a context, currency id and channel address as arguments.
// It returns the error.
func (mgr *PaymentManagerImplV1) SettleInboundCh(ctx context.Context, currencyID uint64, chAddr string) error {
	log.Debugf("SettleInboundCh currency %v, channel %v", currencyID, chAddr)
	activeStore, ok := mgr.inChActiveStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.inChStoresLock[currencyID]
	chStore := mgr.inChStores[currencyID]
	chLocks := mgr.inChLocks[currencyID]
	chLock, ok := chLocks[chAddr]
	if !ok {
		return fmt.Errorf("channel %v does not exist", chAddr)
	}
	cm := mgr.cms[currencyID]
	prv, err := mgr.GetPrvKey(currencyID)
	if err != nil {
		return fmt.Errorf("error loading private key for currency id %v: %v", currencyID, err.Error())
	}
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	log.Debugf("Start RLock")
	if !tryRLock(chLock) {
		return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done RLock")
	chKey := datastore.NewKey(chAddr)
	data, err := chStore.Get(chKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	state := decodeInChState(data)
	fromKey := datastore.NewKey(state.fromAddr)
	exists, err := activeStore.Has(fromKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error checking storage: %v", err.Error())
	}
	active := false
	if exists {
		data, err = activeStore.Get(fromKey)
		if err != nil {
			chLock.RUnlock()
			storeLock.RUnlock()
			return fmt.Errorf("error reading storage: %v", err.Error())
		}
		if chAddr == string(data) {
			active = true
		}
	}
	toRemove := false
	if active {
		if time.Now().Unix() < state.settlement {
			log.Debugf("state still within settlement")
			chLock.RUnlock()
			storeLock.RUnlock()
			return fmt.Errorf("channel %v is still active and within promised settlement time", chAddr)
		}
		toRemove = true
	}
	toUpdate := false
	if state.settlingAt == 0 {
		// Check settling at
		log.Debugf("check settling at")
		settlingAt, _, _, _, _, err := cm.Check(ctx, state.chAddr)
		if err != nil {
			chLock.RUnlock()
			storeLock.RUnlock()
			return fmt.Errorf("error checking paych status: %v", err.Error())
		}
		if settlingAt != 0 {
			state.settlingAt = settlingAt
		} else {
			err = cm.Settle(ctx, prv, state.chAddr, state.voucher)
			if err != nil {
				chLock.RUnlock()
				storeLock.RUnlock()
				return fmt.Errorf("error settling payment channel: %v", err.Error())
			}
			state.settlingAt, _, _, _, _, err = cm.Check(ctx, state.chAddr)
			if err != nil {
				chLock.RUnlock()
				storeLock.RUnlock()
				return fmt.Errorf("error checking paych status: %v", err.Error())
			}
		}
		toUpdate = true
	}
	chLock.RUnlock()
	if toUpdate {
		log.Debugf("Start Lock")
		if !tryLock(chLock) {
			return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
		}
		log.Debugf("Done Lock")
		defer chLock.Unlock()
		settlingAt := state.settlingAt
		// Read state again
		data, err := chStore.Get(chKey)
		if err != nil {
			storeLock.RUnlock()
			return fmt.Errorf("error reading storage: %v", err.Error())
		}
		state := decodeInChState(data)
		state.settlingAt = settlingAt
		if err = chStore.Put(chKey, encodeInChState(state)); err != nil {
			storeLock.RUnlock()
			return fmt.Errorf("error writing to storage: %v", err.Error())
		}
	}
	storeLock.RUnlock()
	if toRemove {
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
		err = activeStore.Delete(fromKey)
		if err != nil {
			return fmt.Errorf("error removing from storage: %v", err.Error())
		}
	}
	return nil
}

// CollectInboundCh collects an inbound payment channel.
// It takes a context, currency id and channel address as arguments.
// It returns the error.
func (mgr *PaymentManagerImplV1) CollectInboundCh(ctx context.Context, currencyID uint64, chAddr string) error {
	log.Debugf("CollectInboundCh currency %v channel %v", currencyID, chAddr)
	activeStore, ok := mgr.inChActiveStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.inChStoresLock[currencyID]
	chStore := mgr.inChStores[currencyID]
	chLocks := mgr.inChLocks[currencyID]
	chLock, ok := chLocks[chAddr]
	if !ok {
		return fmt.Errorf("channel %v does not exist", chAddr)
	}
	cm := mgr.cms[currencyID]
	prv, err := mgr.GetPrvKey(currencyID)
	if err != nil {
		return fmt.Errorf("error loading private key for currency id %v: %v", currencyID, err.Error())
	}
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	log.Debugf("Start RLock")
	if !tryRLock(chLock) {
		return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done RLock")
	chKey := datastore.NewKey(chAddr)
	data, err := chStore.Get(chKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	state := decodeInChState(data)
	fromKey := datastore.NewKey(state.fromAddr)
	exists, err := activeStore.Has(fromKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error checking storage: %v", err.Error())
	}
	active := false
	if exists {
		data, err = activeStore.Get(fromKey)
		if err != nil {
			chLock.RUnlock()
			storeLock.RUnlock()
			return fmt.Errorf("error reading storage: %v", err.Error())
		}
		if chAddr == string(data) {
			active = true
		}
	}
	toRemove := false
	if active {
		if time.Now().Unix() < state.settlement {
			log.Debugf("state still within settlement")
			chLock.RUnlock()
			storeLock.RUnlock()
			return fmt.Errorf("channel %v is still active and within promised settlement time", chAddr)
		}
		toRemove = true
	}
	if state.settlingAt == 0 {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("channel %v has not started settling process", chAddr)
	}
	_, curHeight, _, _, _, err := cm.Check(ctx, state.chAddr)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error checking paych status: %v", err.Error())
	}
	if curHeight < state.settlingAt {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("can not collect channel, settling at %v, current height %v", state.settlingAt, curHeight)
	}
	// Can collect
	log.Debugf("start collecting")
	err = cm.Collect(ctx, prv, chAddr)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error collecting channel: %v", err.Error())
	}
	chLock.RUnlock()
	log.Debugf("Start Lock")
	if !tryLock(chLock) {
		return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done Lock")
	defer chLock.Unlock()
	storeLock.RUnlock()
	log.Debugf("Start store lock Lock")
	storeLock.Lock()
	log.Debugf("End store lock Lock")
	defer storeLock.Unlock()
	if toRemove {
		err = activeStore.Delete(fromKey)
		if err != nil {
			return fmt.Errorf("error removing from storage: %v", err.Error())
		}
	}
	err = chStore.Delete(chKey)
	if err != nil {
		return fmt.Errorf("error removing from storage: %v", err.Error())
	}
	delete(chLocks, chAddr)
	return nil
}

// ReserveForSelf reserves amount for self.
// It takes a currency id, to addr, amount as arguments.
// It returns the error.
func (mgr *PaymentManagerImplV1) ReserveForSelf(currencyID uint64, toAddr string, amt *big.Int) error {
	log.Debugf("ReserveForSelf currency %v, to %v, amt %v", currencyID, toAddr, amt.String())
	if amt.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("supplied amt has non-positive amount: %v", amt.String())
	}
	activeStore, ok := mgr.outChActiveStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.outChStoresLock[currencyID]
	chStore := mgr.outChStores[currencyID]
	chLocks := mgr.outChLocks[currencyID]
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	toKey := datastore.NewKey(toAddr)
	exists, err := activeStore.Has(toKey)
	if err != nil {
		storeLock.RUnlock()
		return fmt.Errorf("error checking storage: %v", err.Error())
	}
	if !exists {
		storeLock.RUnlock()
		return fmt.Errorf("there is no active channel to %v in use", toAddr)
	}
	// Existed, check if state pass settlement
	log.Debugf("check if state pass settlement")
	data, err := activeStore.Get(toKey)
	if err != nil {
		storeLock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	chAddr := string(data)
	chLock := chLocks[chAddr]
	log.Debugf("Start RLock")
	if !tryRLock(chLock) {
		return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done RLock")
	chKey := datastore.NewKey(chAddr)
	data, err = chStore.Get(chKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	state := decodeOutChState(data)
	if time.Now().Unix() >= state.settlement {
		log.Debugf("state has passed settlement")
		storeLock.RUnlock()
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
		defer chLock.RUnlock()
		if err = activeStore.Delete(toKey); err != nil {
			return fmt.Errorf("error removing from storage: %v", err.Error())
		}
		return fmt.Errorf("channel to %v has just expired", toAddr)
	}
	defer storeLock.RUnlock()
	// still within settlement time, continue
	log.Debugf("still within settlement time, continue")
	chLock.RUnlock()
	log.Debugf("Start Lock")
	if !tryLock(chLock) {
		return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done Lock")
	defer chLock.Unlock()
	// Read state again
	data, err = chStore.Get(chKey)
	if err != nil {
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	state = decodeOutChState(data)
	update := state.CleanReservations()
	temp := big.NewInt(0).Add(state.redeemed, state.reserved)
	if big.NewInt(0).Add(temp, amt).Cmp(state.balance) > 0 {
		if update {
			if err = chStore.Put(chKey, encodeOutChState(state)); err != nil {
				return fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return fmt.Errorf("error reserving fund, not enough balance for %v", amt.String())
	}
	res, ok := state.reservations["self"]
	if !ok {
		log.Debugf("create new entry for self reservation")
		res = &reservation{
			remain:        big.NewInt(0),
			amtPaid:       big.NewInt(0),
			surchargePaid: big.NewInt(0),
		}
		state.reservations["self"] = res
	}
	res.remain.Add(res.remain, amt)
	res.last = time.Now().Unix()
	state.reserved.Add(state.reserved, amt)
	log.Debugf("total %v reserved for self", res.remain.String())
	if err = chStore.Put(chKey, encodeOutChState(state)); err != nil {
		return fmt.Errorf("error writing to storage: %v", err.Error())
	}
	return nil
}

// ReserveForSelfWithOffer reserves amount for self with given offer.
// It takes a pay offer as the argument.
// It returns the error.
func (mgr *PaymentManagerImplV1) ReserveForSelfWithOffer(offer *payoffer.PayOffer) error {
	if offer == nil {
		return fmt.Errorf("offer supplied is nil")
	}
	log.Debugf("ReserveForSelfWithOffer offer %v", offer.ID())
	if time.Now().Unix() > offer.Expiration() {
		return fmt.Errorf("offer supplied has expired at %v but now is %v", offer.Expiration(), time.Now().Unix())
	}
	currencyID := offer.CurrencyID()
	toAddr := offer.From()
	amt := offer.MaxAmt()
	if amt.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("offer supplied has non-positive amount: %v", amt.String())
	}
	// amt_required = amt + ((amt - 1) / offer_period + 1) * offer_ppp
	temp := big.NewInt(0).Sub(amt, big.NewInt(1))
	temp = big.NewInt(0).Div(temp, offer.Period())
	temp = big.NewInt(0).Add(temp, big.NewInt(1))
	amt.Add(amt, big.NewInt(0).Mul(temp, offer.PPP()))
	log.Debugf("amt required is %v", amt)
	activeStore, ok := mgr.outChActiveStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.outChStoresLock[currencyID]
	chStore := mgr.outChStores[currencyID]
	chLocks := mgr.outChLocks[currencyID]
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	toKey := datastore.NewKey(toAddr)
	exists, err := activeStore.Has(toKey)
	if err != nil {
		storeLock.RUnlock()
		return fmt.Errorf("error checking storage: %v", err.Error())
	}
	if !exists {
		storeLock.RUnlock()
		return fmt.Errorf("there is no active channel to %v in use", toAddr)
	}
	// Existed, check if state pass settlement
	log.Debugf("check if state pass settlement")
	data, err := activeStore.Get(toKey)
	if err != nil {
		storeLock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	chAddr := string(data)
	chLock := chLocks[chAddr]
	log.Debugf("Start RLock")
	if !tryRLock(chLock) {
		return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done RLock")
	chKey := datastore.NewKey(chAddr)
	data, err = chStore.Get(chKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	state := decodeOutChState(data)
	if time.Now().Unix() >= state.settlement {
		log.Debugf("state has passed settlement")
		storeLock.RUnlock()
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
		defer chLock.RUnlock()
		if err = activeStore.Delete(toKey); err != nil {
			return fmt.Errorf("error removing from storage: %v", err.Error())
		}
		return fmt.Errorf("channel to %v has just expired", toAddr)
	}
	defer storeLock.RUnlock()
	// still within settlement time, continue
	log.Debugf("still within settlement time, continue")
	chLock.RUnlock()
	log.Debugf("Start Lock")
	if !tryLock(chLock) {
		return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done Lock")
	defer chLock.Unlock()
	// Read state again
	data, err = chStore.Get(chKey)
	if err != nil {
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	state = decodeOutChState(data)
	update := state.CleanReservations()
	temp = big.NewInt(0).Add(state.redeemed, state.reserved)
	if big.NewInt(0).Add(temp, amt).Cmp(state.balance) > 0 {
		if update {
			if err = chStore.Put(chKey, encodeOutChState(state)); err != nil {
				return fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return fmt.Errorf("error reserving fund, not enough balance for %v", amt.String())
	}
	res, ok := state.reservations[fmt.Sprintf("self-%v", offer.ID().String())]
	if !ok {
		log.Debugf("create new entry for self reservation with offer %v", offer.ID())
		res = &reservation{
			remain:        big.NewInt(0),
			amtPaid:       big.NewInt(0),
			surchargePaid: big.NewInt(0),
		}
		state.reservations[fmt.Sprintf("self-%v", offer.ID().String())] = res
	}
	res.remain.Add(res.remain, amt)
	res.last = time.Now().Unix()
	res.offer = offer
	state.reserved.Add(state.reserved, amt)
	log.Debugf("total %v reserved for self with offer %v", res.remain.String(), offer.ID())
	if err = chStore.Put(chKey, encodeOutChState(state)); err != nil {
		return fmt.Errorf("error writing to storage: %v", err.Error())
	}
	return nil
}

// ReserveForOthers reserves amount for others with given offer.
// It takes a pay offer as the argument.
// It returns the error.
func (mgr *PaymentManagerImplV1) ReserveForOthers(offer *payoffer.PayOffer) error {
	if offer == nil {
		return fmt.Errorf("offer supplied is nil")
	}
	log.Debugf("ReserveForOthers with offer %v", offer.ID())
	if time.Now().Unix() > offer.Expiration() {
		return fmt.Errorf("offer supplied has expired at %v but now is %v", offer.Expiration(), time.Now().Unix())
	}
	currencyID := offer.CurrencyID()
	var toAddr string
	amt := offer.MaxAmt()
	if amt.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("offer supplied has non-positive amount: %v", amt.String())
	}
	if offer.LinkedOffer() == cid.Undef {
		toAddr = offer.To()
	} else {
		linkedOffer, err := mgr.os.GetOffer(offer.LinkedOffer())
		if err != nil {
			return fmt.Errorf("error getting offer from payofferstore: %v", err.Error())
		}
		toAddr = linkedOffer.From()
		// amt_required = amt + ((amt - 1) / offer_period + 1) * offer_ppp
		temp := big.NewInt(0).Sub(amt, big.NewInt(1))
		temp = big.NewInt(0).Div(temp, offer.Period())
		temp = big.NewInt(0).Add(temp, big.NewInt(1))
		amt.Add(amt, big.NewInt(0).Mul(temp, offer.PPP()))
	}
	log.Debugf("amt required is %v", amt)
	activeStore, ok := mgr.outChActiveStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.outChStoresLock[currencyID]
	chStore := mgr.outChStores[currencyID]
	chLocks := mgr.outChLocks[currencyID]
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	toKey := datastore.NewKey(toAddr)
	exists, err := activeStore.Has(toKey)
	if err != nil {
		storeLock.RUnlock()
		return fmt.Errorf("error checking storage: %v", err.Error())
	}
	if !exists {
		storeLock.RUnlock()
		return fmt.Errorf("there is no active channel to %v in use", toAddr)
	}
	// Existed, check if state pass settlement
	log.Debugf("check if state pass settlement")
	data, err := activeStore.Get(toKey)
	if err != nil {
		storeLock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	chAddr := string(data)
	chLock := chLocks[chAddr]
	log.Debugf("Start RLock")
	if !tryRLock(chLock) {
		return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done RLock")
	chKey := datastore.NewKey(chAddr)
	data, err = chStore.Get(chKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	state := decodeOutChState(data)
	if time.Now().Unix() >= state.settlement {
		log.Debugf("state has passed settlement")
		storeLock.RUnlock()
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
		defer chLock.RUnlock()
		if err = activeStore.Delete(toKey); err != nil {
			return fmt.Errorf("error removing from storage: %v", err.Error())
		}
		return fmt.Errorf("channel to %v has just expired", toAddr)
	}
	defer storeLock.RUnlock()
	// still within settlement time, continue
	log.Debugf("still within settlement time, continue")
	chLock.RUnlock()
	log.Debugf("Start Lock")
	if !tryLock(chLock) {
		return fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done Lock")
	defer chLock.Unlock()
	// Read state again
	data, err = chStore.Get(chKey)
	if err != nil {
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	state = decodeOutChState(data)
	update := state.CleanReservations()
	temp := big.NewInt(0).Add(state.redeemed, state.reserved)
	if big.NewInt(0).Add(temp, amt).Cmp(state.balance) > 0 {
		if update {
			if err = chStore.Put(chKey, encodeOutChState(state)); err != nil {
				return fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return fmt.Errorf("error reserving fund, not enough balance for %v", amt.String())
	}
	res, ok := state.reservations[offer.ID().String()]
	if !ok {
		log.Debugf("create new entry for offer %v", offer.ID())
		res = &reservation{
			remain:        big.NewInt(0),
			amtPaid:       big.NewInt(0),
			surchargePaid: big.NewInt(0),
		}
		state.reservations[offer.ID().String()] = res
	}
	res.remain.Add(res.remain, amt)
	res.last = time.Now().Unix()
	res.offer = offer
	state.reserved.Add(state.reserved, amt)
	log.Debugf("total %v reserved for offer %v", res.remain, offer.ID())
	if err = chStore.Put(chKey, encodeOutChState(state)); err != nil {
		return fmt.Errorf("error writing to storage: %v", err.Error())
	}
	return nil
}

// PayForSelf is used to pay for self.
// It takes a currency id, to addr, amount as arguments.
// It returns a voucher, a function to commit, a function to revert and error.
func (mgr *PaymentManagerImplV1) PayForSelf(currencyID uint64, toAddr string, amt *big.Int) (string, func(), func(), error) {
	log.Debugf("PayForSelf currency %v, to %v, amt %v", currencyID, toAddr, amt.String())
	if amt.Cmp(big.NewInt(0)) <= 0 {
		return "", nil, nil, fmt.Errorf("zero amount supplied")
	}
	activeStore, ok := mgr.outChActiveStores[currencyID]
	if !ok {
		return "", nil, nil, fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.outChStoresLock[currencyID]
	chStore := mgr.outChStores[currencyID]
	chLocks := mgr.outChLocks[currencyID]
	cm := mgr.cms[currencyID]
	prv, err := mgr.GetPrvKey(currencyID)
	if err != nil {
		return "", nil, nil, fmt.Errorf("error loading private key for currency id %v: %v", currencyID, err.Error())
	}
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	toKey := datastore.NewKey(toAddr)
	exists, err := activeStore.Has(toKey)
	if err != nil {
		storeLock.RUnlock()
		return "", nil, nil, fmt.Errorf("error checking storage: %v", err.Error())
	}
	if !exists {
		storeLock.RUnlock()
		return "", nil, nil, fmt.Errorf("there is no active channel to %v in use", toAddr)
	}
	// Existed, check if state pass settlement
	log.Debugf("check if state pass settlement")
	data, err := activeStore.Get(toKey)
	if err != nil {
		storeLock.RUnlock()
		return "", nil, nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	chAddr := string(data)
	chLock := chLocks[chAddr]
	log.Debugf("Start RLock")
	if !tryRLock(chLock) {
		return "", nil, nil, fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done RLock")
	chKey := datastore.NewKey(chAddr)
	data, err = chStore.Get(chKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return "", nil, nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	state := decodeOutChState(data)
	if time.Now().Unix() >= state.settlement {
		log.Debugf("state has passed settlement")
		storeLock.RUnlock()
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
		defer chLock.RUnlock()
		if err = activeStore.Delete(toKey); err != nil {
			return "", nil, nil, fmt.Errorf("error removing from storage: %v", err.Error())
		}
		return "", nil, nil, fmt.Errorf("channel to %v has just expired", toAddr)
	}
	defer storeLock.RUnlock()
	// still within settlement time, continue
	log.Debugf("still within settlement time, continue")
	chLock.RUnlock()
	log.Debugf("Start Lock")
	if !tryLock(chLock) {
		return "", nil, nil, fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done Lock")
	// Read state again
	data, err = chStore.Get(chKey)
	if err != nil {
		return "", nil, nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	state = decodeOutChState(data)
	update := state.CleanReservations()
	resID := "self"
	res, ok := state.reservations[resID]
	if !ok {
		defer chLock.Unlock()
		if update {
			if err = chStore.Put(chKey, encodeOutChState(state)); err != nil {
				return "", nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return "", nil, nil, fmt.Errorf("no fund reserved to self to address %v", toAddr)
	}
	if res.remain.Cmp(amt) < 0 {
		defer chLock.Unlock()
		if update {
			if err = chStore.Put(chKey, encodeOutChState(state)); err != nil {
				return "", nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return "", nil, nil, fmt.Errorf("reservation does not have enough balance to pay")
	}
	// Generate voucher
	voucher, err := cm.GenerateVoucher(prv, state.chAddr, 0, state.nonce+1, big.NewInt(0).Add(state.redeemed, amt))
	if err != nil {
		defer chLock.Unlock()
		if update {
			if err := chStore.Put(chKey, encodeOutChState(state)); err != nil {
				return "", nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return "", nil, nil, fmt.Errorf("error generating voucher: %v", err.Error())
	}
	// Update state
	res.remain.Sub(res.remain, amt)
	res.amtPaid.Add(res.amtPaid, amt)
	res.last = time.Now().Unix()
	if res.remain.Cmp(big.NewInt(0)) == 0 {
		delete(state.reservations, resID)
	}
	state.reserved.Sub(state.reserved, amt)
	state.redeemed.Add(state.redeemed, amt)
	state.nonce += 1
	state.voucher = voucher
	commit := func() {
		if err = chStore.Put(chKey, encodeOutChState(state)); err != nil {
			log.Error("error commiting the state: %v", err.Error())
		}
		chLock.Unlock()
	}
	revert := func() {
		chLock.Unlock()
	}
	return voucher, commit, revert, nil
}

// PayForSelfWithOffer is used to pay for self with given offer.
// It takes the amount, a pay offer as arguments.
// It returns a voucher, a function to commit, a function to revert and error.
func (mgr *PaymentManagerImplV1) PayForSelfWithOffer(amt *big.Int, offer *payoffer.PayOffer) (string, func(), func(), error) {
	if amt.Cmp(big.NewInt(0)) <= 0 {
		return "", nil, nil, fmt.Errorf("zero amount supplied")
	}
	if offer == nil {
		return "", nil, nil, fmt.Errorf("offer supplied is nil")
	}
	log.Debugf("PayForSelfWithOffer amt %v, offer %v", amt, offer.ID())
	if time.Now().Unix() > offer.Expiration() {
		return "", nil, nil, fmt.Errorf("offer supplied has expired at %v but now is %v", offer.Expiration(), time.Now().Unix())
	}
	currencyID := offer.CurrencyID()
	toAddr := offer.From()
	activeStore, ok := mgr.outChActiveStores[currencyID]
	if !ok {
		return "", nil, nil, fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.outChStoresLock[currencyID]
	chStore := mgr.outChStores[currencyID]
	chLocks := mgr.outChLocks[currencyID]
	cm := mgr.cms[currencyID]
	prv, err := mgr.GetPrvKey(currencyID)
	if err != nil {
		return "", nil, nil, fmt.Errorf("error loading private key for currency id %v: %v", currencyID, err.Error())
	}
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	toKey := datastore.NewKey(toAddr)
	exists, err := activeStore.Has(toKey)
	if err != nil {
		storeLock.RUnlock()
		return "", nil, nil, fmt.Errorf("error checking storage: %v", err.Error())
	}
	if !exists {
		storeLock.RUnlock()
		return "", nil, nil, fmt.Errorf("there is no active channel to %v in use", toAddr)
	}
	// Existed, check if state pass settlement
	log.Debugf("check if state pass settlement")
	data, err := activeStore.Get(toKey)
	if err != nil {
		storeLock.RUnlock()
		return "", nil, nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	chAddr := string(data)
	chLock := chLocks[chAddr]
	log.Debugf("Start RLock")
	if !tryRLock(chLock) {
		return "", nil, nil, fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done RLock")
	chKey := datastore.NewKey(chAddr)
	data, err = chStore.Get(chKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return "", nil, nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	state := decodeOutChState(data)
	if time.Now().Unix() >= state.settlement {
		log.Debugf("state has passed settlement")
		storeLock.RUnlock()
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
		defer chLock.RUnlock()
		if err = activeStore.Delete(toKey); err != nil {
			return "", nil, nil, fmt.Errorf("error removing from storage: %v", err.Error())
		}
		return "", nil, nil, fmt.Errorf("channel to %v has just expired", toAddr)
	}
	defer storeLock.RUnlock()
	// still within settlement time, continue
	log.Debugf("still within settlement time, continue")
	chLock.RUnlock()
	log.Debugf("Start Lock")
	if !tryLock(chLock) {
		return "", nil, nil, fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done Lock")
	// Read state again
	data, err = chStore.Get(chKey)
	if err != nil {
		return "", nil, nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	state = decodeOutChState(data)
	update := state.CleanReservations()
	resID := fmt.Sprintf("self-%v", offer.ID().String())
	res, ok := state.reservations[resID]
	if !ok {
		defer chLock.Unlock()
		if update {
			if err = chStore.Put(chKey, encodeOutChState(state)); err != nil {
				return "", nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return "", nil, nil, fmt.Errorf("no fund reserved to self to address %v with offer %v", toAddr, offer.ID())
	}
	// Need to calculate the actual amount required.
	expectedAmt := big.NewInt(0).Add(res.amtPaid, amt)
	temp := big.NewInt(0).Sub(expectedAmt, big.NewInt(1))
	temp = big.NewInt(0).Div(temp, offer.Period())
	temp = big.NewInt(0).Add(temp, big.NewInt(1))
	expectedSurcharge := big.NewInt(0).Mul(temp, offer.PPP())
	log.Debugf("expected amt %v, expected surcharge %v", expectedAmt, expectedSurcharge)
	// Amt = (new amt + new surcharge) - (old amt + old surcharge)
	amt = big.NewInt(0).Sub(big.NewInt(0).Add(expectedAmt, expectedSurcharge), big.NewInt(0).Add(res.amtPaid, res.surchargePaid))
	if res.remain.Cmp(amt) < 0 {
		defer chLock.Unlock()
		if update {
			if err = chStore.Put(chKey, encodeOutChState(state)); err != nil {
				return "", nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return "", nil, nil, fmt.Errorf("reservation does not have enough balance to pay")
	}
	// Generate voucher
	voucher, err := cm.GenerateVoucher(prv, state.chAddr, 0, state.nonce+1, big.NewInt(0).Add(state.redeemed, amt))
	if err != nil {
		defer chLock.Unlock()
		if update {
			if err := chStore.Put(chKey, encodeOutChState(state)); err != nil {
				return "", nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return "", nil, nil, fmt.Errorf("error generating voucher: %v", err.Error())
	}
	// Update state
	res.remain.Sub(res.remain, amt)
	res.amtPaid = expectedAmt
	res.surchargePaid = expectedSurcharge
	res.last = time.Now().Unix()
	if res.remain.Cmp(big.NewInt(0)) == 0 {
		delete(state.reservations, resID)
	}
	state.reserved.Sub(state.reserved, amt)
	state.redeemed.Add(state.redeemed, amt)
	state.nonce += 1
	state.voucher = voucher
	commit := func() {
		if err = chStore.Put(chKey, encodeOutChState(state)); err != nil {
			log.Error("error commiting the state: %v", err.Error())
		}
		chLock.Unlock()
	}
	revert := func() {
		chLock.Unlock()
	}
	return voucher, commit, revert, nil
}

// PayForOthers is used to pay for others with given offer.
// It takes the amount, a pay offer as arguments.
// It returns a voucher, a function to commit, a function to revert and error.
func (mgr *PaymentManagerImplV1) PayForOthers(amt *big.Int, offer *payoffer.PayOffer) (string, func(), func(), error) {
	if amt.Cmp(big.NewInt(0)) <= 0 {
		return "", nil, nil, fmt.Errorf("zero amount supplied")
	}
	if offer == nil {
		return "", nil, nil, fmt.Errorf("offer supplied is nil")
	}
	log.Debugf("PayForOthers amt %v offer %v", amt, offer.ID())
	if time.Now().Unix() > offer.Expiration() {
		return "", nil, nil, fmt.Errorf("offer supplied has expired at %v but now is %v", offer.Expiration(), time.Now().Unix())
	}
	currencyID := offer.CurrencyID()
	var toAddr string
	var linkedOffer *payoffer.PayOffer
	var err error
	if offer.LinkedOffer() == cid.Undef {
		toAddr = offer.To()
	} else {
		linkedOffer, err = mgr.os.GetOffer(offer.LinkedOffer())
		if err != nil {
			return "", nil, nil, fmt.Errorf("error getting offer from payofferstore: %v", err.Error())
		}
		toAddr = linkedOffer.From()
	}
	activeStore, ok := mgr.outChActiveStores[currencyID]
	if !ok {
		return "", nil, nil, fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.outChStoresLock[currencyID]
	chStore := mgr.outChStores[currencyID]
	chLocks := mgr.outChLocks[currencyID]
	cm := mgr.cms[currencyID]
	prv, err := mgr.GetPrvKey(currencyID)
	if err != nil {
		return "", nil, nil, fmt.Errorf("error loading private key for currency id %v: %v", currencyID, err.Error())
	}
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	toKey := datastore.NewKey(toAddr)
	exists, err := activeStore.Has(toKey)
	if err != nil {
		storeLock.RUnlock()
		return "", nil, nil, fmt.Errorf("error checking storage: %v", err.Error())
	}
	if !exists {
		storeLock.RUnlock()
		return "", nil, nil, fmt.Errorf("there is no active channel to %v in use", toAddr)
	}
	// Existed, check if state pass settlement
	log.Debugf("check if state pass settlement")
	data, err := activeStore.Get(toKey)
	if err != nil {
		storeLock.RUnlock()
		return "", nil, nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	chAddr := string(data)
	chLock := chLocks[chAddr]
	log.Debugf("Start RLock")
	if !tryRLock(chLock) {
		return "", nil, nil, fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done RLock")
	chKey := datastore.NewKey(chAddr)
	data, err = chStore.Get(chKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return "", nil, nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	state := decodeOutChState(data)
	if time.Now().Unix() >= state.settlement {
		log.Debugf("state has passed settlement")
		storeLock.RUnlock()
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
		defer chLock.RUnlock()
		if err = activeStore.Delete(toKey); err != nil {
			return "", nil, nil, fmt.Errorf("error removing from storage: %v", err.Error())
		}
		return "", nil, nil, fmt.Errorf("channel to %v has just expired", toAddr)
	}
	defer storeLock.RUnlock()
	// still within settlement time, continue
	log.Debugf("still within settlement time, continue")
	chLock.RUnlock()
	log.Debugf("Start Lock")
	if !tryLock(chLock) {
		return "", nil, nil, fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done Lock")
	// Read state again
	data, err = chStore.Get(chKey)
	if err != nil {
		return "", nil, nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	state = decodeOutChState(data)
	update := state.CleanReservations()
	resID := offer.ID().String()
	res, ok := state.reservations[resID]
	if !ok {
		defer chLock.Unlock()
		if update {
			if err = chStore.Put(chKey, encodeOutChState(state)); err != nil {
				return "", nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return "", nil, nil, fmt.Errorf("no fund reserved to %v with given offer %v", toAddr, offer.ID())
	}
	expectedAmt := big.NewInt(0).Add(res.amtPaid, amt)
	var expectedSurcharge *big.Int
	if offer.LinkedOffer() == cid.Undef {
		expectedSurcharge = big.NewInt(0)
	} else {
		temp := big.NewInt(0).Sub(expectedAmt, big.NewInt(1))
		temp = big.NewInt(0).Div(temp, linkedOffer.Period())
		temp = big.NewInt(0).Add(temp, big.NewInt(1))
		expectedSurcharge = big.NewInt(0).Mul(temp, linkedOffer.PPP())
	}
	log.Debugf("expected amt %v, expected surcharge %v", expectedAmt, expectedSurcharge)
	// Amt = (new amt + new surcharge) - (old amt + old surcharge)
	amt = big.NewInt(0).Sub(big.NewInt(0).Add(expectedAmt, expectedSurcharge), big.NewInt(0).Add(res.amtPaid, res.surchargePaid))
	if res.remain.Cmp(amt) < 0 {
		defer chLock.Unlock()
		if update {
			if err = chStore.Put(chKey, encodeOutChState(state)); err != nil {
				return "", nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return "", nil, nil, fmt.Errorf("reservation does not have enough balance to pay")
	}
	// Generate voucher
	voucher, err := cm.GenerateVoucher(prv, state.chAddr, 0, state.nonce+1, big.NewInt(0).Add(state.redeemed, amt))
	if err != nil {
		defer chLock.Unlock()
		if update {
			if err := chStore.Put(chKey, encodeOutChState(state)); err != nil {
				return "", nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return "", nil, nil, fmt.Errorf("error generating voucher: %v", err.Error())
	}
	// Update state
	res.remain.Sub(res.remain, amt)
	res.amtPaid = expectedAmt
	res.surchargePaid = expectedSurcharge
	res.last = time.Now().Unix()
	if res.remain.Cmp(big.NewInt(0)) == 0 {
		delete(state.reservations, resID)
	}
	state.reserved.Sub(state.reserved, amt)
	state.redeemed.Add(state.redeemed, amt)
	state.nonce += 1
	state.voucher = voucher
	commit := func() {
		if err = chStore.Put(chKey, encodeOutChState(state)); err != nil {
			log.Error("error commiting the state: %v", err.Error())
		}
		chLock.Unlock()
	}
	revert := func() {
		chLock.Unlock()
	}
	return voucher, commit, revert, nil
}

// RecordReceiveForSelf is used to record a receive of payment for self.
// It takes a currency id, from address, voucher, origin address as arguments.
// It returns the amount received, a function to commit, a function to revert and error.
func (mgr *PaymentManagerImplV1) RecordReceiveForSelf(currencyID uint64, fromAddr string, voucher string, originAddr string) (*big.Int, func(), func(), error) {
	log.Debugf("RecordReceiveForSelf currency %v, from %v, voucher %v, origin %v", currencyID, fromAddr, voucher, originAddr)
	activeStore, ok := mgr.inChActiveStores[currencyID]
	if !ok {
		return nil, nil, nil, fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.inChStoresLock[currencyID]
	chStore := mgr.inChStores[currencyID]
	chLocks := mgr.inChLocks[currencyID]
	cm := mgr.cms[currencyID]
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	fromKey := datastore.NewKey(fromAddr)
	exists, err := activeStore.Has(fromKey)
	if err != nil {
		storeLock.RUnlock()
		return nil, nil, nil, fmt.Errorf("error checking storage: %v", err.Error())
	}
	if !exists {
		storeLock.RUnlock()
		return nil, nil, nil, fmt.Errorf("there is no active channel from %v in use", fromAddr)
	}
	// Existed, check if state pass settlement
	log.Debugf("check if state pass settlement")
	data, err := activeStore.Get(fromKey)
	if err != nil {
		storeLock.RUnlock()
		return nil, nil, nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	chAddr := string(data)
	chLock := chLocks[chAddr]
	log.Debugf("Start RLock")
	if !tryRLock(chLock) {
		return nil, nil, nil, fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done RLock")
	chKey := datastore.NewKey(chAddr)
	data, err = chStore.Get(chKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return nil, nil, nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	state := decodeInChState(data)
	if time.Now().Unix() >= state.settlement {
		log.Debugf("state has passed settlement")
		storeLock.RUnlock()
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
		defer chLock.RUnlock()
		if err = activeStore.Delete(fromKey); err != nil {
			return nil, nil, nil, fmt.Errorf("error removing from storage: %v", err.Error())
		}
		return nil, nil, nil, fmt.Errorf("channel from %v has just expired", fromAddr)
	}
	defer storeLock.RUnlock()
	// still within settlement time, continue
	log.Debugf("still within settlement time, continue")
	chLock.RUnlock()
	log.Debugf("Start Lock")
	if !tryLock(chLock) {
		return nil, nil, nil, fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done Lock")
	// Read state again
	data, err = chStore.Get(chKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	state = decodeInChState(data)
	update := state.CleanReservations()
	log.Debugf("state updated %v", update)
	sender, channelAddr, lane, nonce, redeemed, err := cm.VerifyVoucher(voucher)
	if err != nil {
		defer chLock.Unlock()
		if update {
			if err := chStore.Put(chKey, encodeInChState(state)); err != nil {
				return nil, nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return nil, nil, nil, fmt.Errorf("error verifying voucher: %v", err.Error())
	}
	if sender != fromAddr || channelAddr != chAddr || lane != 0 || nonce <= state.nonce {
		defer chLock.Unlock()
		if update {
			if err = chStore.Put(chKey, encodeInChState(state)); err != nil {
				return nil, nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		log.Debugf("voucher sender %v, from %v, channel voucher %v, channel %v, lane %v, nonce %v, state nonce %v", sender, fromAddr, channelAddr, chAddr, lane, nonce, state.nonce)
		return nil, nil, nil, fmt.Errorf("invalid voucher received")
	}
	if redeemed.Cmp(state.balance) > 0 {
		log.Debugf("redeemed is bigger than balance, check balance once")
		_, _, newBalance, _, _, err := cm.Check(mgr.ctx, chAddr)
		if err != nil {
			defer chLock.Unlock()
			if update {
				if err := chStore.Put(chKey, encodeInChState(state)); err != nil {
					return nil, nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
				}
			}
			return nil, nil, nil, fmt.Errorf("error checking paych state: %v", err.Error())
		}
		if newBalance.Cmp(state.balance) > 0 {
			log.Debugf("balance updated")
			update = true
			state.balance = newBalance
			if redeemed.Cmp(state.balance) > 0 {
				log.Debugf("balance still not enough")
				defer chLock.Unlock()
				if update {
					if err = chStore.Put(chKey, encodeInChState(state)); err != nil {
						return nil, nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
					}
				}
				return nil, nil, nil, fmt.Errorf("not enough balance in the paych")
			}
		} else {
			defer chLock.Unlock()
			if update {
				if err = chStore.Put(chKey, encodeInChState(state)); err != nil {
					return nil, nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
				}
			}
			return nil, nil, nil, fmt.Errorf("not enough balance in the paych")
		}
	}
	// Voucher is valid.
	amt := big.NewInt(0).Sub(redeemed, state.redeemed)
	if amt.Cmp(big.NewInt(0)) <= 0 {
		defer chLock.Unlock()
		if update {
			if err = chStore.Put(chKey, encodeInChState(state)); err != nil {
				return nil, nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return nil, nil, nil, fmt.Errorf("zero voucher received")
	}
	log.Debugf("received %v from %v origin %v", amt, fromAddr, originAddr)
	// Update state
	state.redeemed = redeemed
	state.nonce = nonce
	state.voucher = voucher
	recordedStore := mgr.recordedStores[currencyID]
	recordedStoreLock := mgr.recordedStoresLock[currencyID]
	recordLocks := mgr.recordedEntriesLock[currencyID]
	commit := func() {
		defer chLock.Unlock()
		if err = chStore.Put(chKey, encodeInChState(state)); err != nil {
			log.Error("error commiting the state: %v", err.Error())
			return
		}
		originKey := datastore.NewKey(originAddr)
		recordedStoreLock.RLock()
		exists, err := recordedStore.Has(originKey)
		if err != nil {
			recordedStoreLock.RUnlock()
			log.Error("error commiting the state: %v", err.Error())
			return
		}
		if !exists {
			// Need a new entry
			recordedStoreLock.RUnlock()
			recordedStoreLock.Lock()
			defer recordedStoreLock.Unlock()
			// Check again
			exists, err = recordedStore.Has(originKey)
			if err != nil {
				log.Error("error commiting the state: %v", err.Error())
				return
			}
			if !exists {
				recordLocks[originAddr] = &sync.RWMutex{}
			}
		} else {
			defer recordedStoreLock.RUnlock()
		}
		recordLock := recordLocks[originAddr]
		recordLock.Lock()
		defer recordLock.Unlock()
		if exists {
			data, err := recordedStore.Get(originKey)
			if err != nil {
				log.Error("error commiting the state: %v", err.Error())
				return
			}
			amt.Add(amt, big.NewInt(0).SetBytes(data))
		}
		if err = recordedStore.Put(originKey, amt.Bytes()); err != nil {
			log.Error("error commiting the state: %v", err.Error())
		}
	}
	revert := func() {
		chLock.Unlock()
	}
	return amt, commit, revert, nil
}

// RecordReceiveForOthers is used to record a receive of payment for others.
// It takes a from address, voucher, pay offer as arguments.
// It returns the amount received (after surcharge), a function to commit, a function to revert and error.
func (mgr *PaymentManagerImplV1) RecordReceiveForOthers(fromAddr string, voucher string, offer *payoffer.PayOffer) (*big.Int, func(), func(), error) {
	if offer == nil {
		return nil, nil, nil, fmt.Errorf("offer supplied is nil")
	}
	log.Debugf("RecordReceiveForOthers from %v, voucher %v, offer %v", fromAddr, voucher, offer.ID())
	if time.Now().Unix() > offer.Expiration() {
		return nil, nil, nil, fmt.Errorf("offer supplied has expired at %v but now is %v", offer.Expiration(), time.Now().Unix())
	}
	currencyID := offer.CurrencyID()
	activeStore, ok := mgr.inChActiveStores[currencyID]
	if !ok {
		return nil, nil, nil, fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.inChStoresLock[currencyID]
	chStore := mgr.inChStores[currencyID]
	chLocks := mgr.inChLocks[currencyID]
	cm := mgr.cms[currencyID]
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	fromKey := datastore.NewKey(fromAddr)
	exists, err := activeStore.Has(fromKey)
	if err != nil {
		storeLock.RUnlock()
		return nil, nil, nil, fmt.Errorf("error checking storage: %v", err.Error())
	}
	if !exists {
		storeLock.RUnlock()
		return nil, nil, nil, fmt.Errorf("there is no active channel from %v in use", fromAddr)
	}
	// Existed, check if state pass settlement
	log.Debugf("check if state pass settlement")
	data, err := activeStore.Get(fromKey)
	if err != nil {
		storeLock.RUnlock()
		return nil, nil, nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	chAddr := string(data)
	chLock := chLocks[chAddr]
	log.Debugf("Start RLock")
	if !tryRLock(chLock) {
		return nil, nil, nil, fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done RLock")
	chKey := datastore.NewKey(chAddr)
	data, err = chStore.Get(chKey)
	if err != nil {
		chLock.RUnlock()
		storeLock.RUnlock()
		return nil, nil, nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	state := decodeInChState(data)
	if time.Now().Unix() >= state.settlement {
		log.Debugf("state has passed settlement")
		storeLock.RUnlock()
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
		defer chLock.RUnlock()
		if err = activeStore.Delete(fromKey); err != nil {
			return nil, nil, nil, fmt.Errorf("error removing from storage: %v", err.Error())
		}
		return nil, nil, nil, fmt.Errorf("channel from %v has just expired", fromAddr)
	}
	defer storeLock.RUnlock()
	// still within settlement time, continue
	log.Debugf("still within settlement time, continue")
	chLock.RUnlock()
	log.Debugf("Start Lock")
	if !tryLock(chLock) {
		return nil, nil, nil, fmt.Errorf("error locking channel after %v attempts", lockAttempt)
	}
	log.Debugf("Done Lock")
	// Read state again
	data, err = chStore.Get(chKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	state = decodeInChState(data)
	_, existsBefore := state.offerStates[offer.ID().String()]
	update := state.CleanReservations()
	_, existsAfter := state.offerStates[offer.ID().String()]
	log.Debugf("state updated %v, exist before %v, exist after %v", update, existsBefore, existsAfter)
	if !existsAfter && existsBefore {
		defer chLock.Unlock()
		if update {
			if err = chStore.Put(chKey, encodeInChState(state)); err != nil {
				return nil, nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return nil, nil, nil, fmt.Errorf("offer state has expired")
	}
	sender, channelAddr, lane, nonce, redeemed, err := cm.VerifyVoucher(voucher)
	if err != nil {
		defer chLock.Unlock()
		if update {
			if err := chStore.Put(chKey, encodeInChState(state)); err != nil {
				return nil, nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return nil, nil, nil, fmt.Errorf("error verifying voucher: %v", err.Error())
	}
	if sender != fromAddr || channelAddr != chAddr || lane != 0 || nonce <= state.nonce {
		defer chLock.Unlock()
		if update {
			if err = chStore.Put(chKey, encodeInChState(state)); err != nil {
				return nil, nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		log.Debugf("voucher sender %v, from %v, channel voucher %v, channel %v, lane %v, nonce %v, state nonce %v", sender, fromAddr, channelAddr, chAddr, lane, nonce, state.nonce)
		return nil, nil, nil, fmt.Errorf("invalid voucher received")
	}
	if redeemed.Cmp(state.balance) > 0 {
		log.Debugf("redeemed is bigger than balance, check balance once")
		_, _, newBalance, _, _, err := cm.Check(mgr.ctx, chAddr)
		if err != nil {
			defer chLock.Unlock()
			if update {
				if err := chStore.Put(chKey, encodeInChState(state)); err != nil {
					return nil, nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
				}
			}
			return nil, nil, nil, fmt.Errorf("error checking paych state: %v", err.Error())
		}
		if newBalance.Cmp(state.balance) > 0 {
			log.Debugf("balance updated")
			update = true
			state.balance = newBalance
			if redeemed.Cmp(state.balance) > 0 {
				log.Debugf("balance still not enough")
				defer chLock.Unlock()
				if update {
					if err = chStore.Put(chKey, encodeInChState(state)); err != nil {
						return nil, nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
					}
				}
				return nil, nil, nil, fmt.Errorf("not enough balance in the paych")
			}
		} else {
			defer chLock.Unlock()
			if update {
				if err = chStore.Put(chKey, encodeInChState(state)); err != nil {
					return nil, nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
				}
			}
			return nil, nil, nil, fmt.Errorf("not enough balance in the paych")
		}
	}
	// Voucher is valid.
	amt := big.NewInt(0).Sub(redeemed, state.redeemed)
	if amt.Cmp(big.NewInt(0)) <= 0 {
		defer chLock.Unlock()
		if update {
			if err = chStore.Put(chKey, encodeInChState(state)); err != nil {
				return nil, nil, nil, fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
		return nil, nil, nil, fmt.Errorf("zero voucher received")
	}
	// Update state
	os, ok := state.offerStates[offer.ID().String()]
	if !ok {
		os = &offerState{
			amtReceived:       big.NewInt(0),
			surchargeReceived: big.NewInt(0),
			offer:             offer,
		}
		state.offerStates[offer.ID().String()] = os
	}
	received := big.NewInt(0)
	log.Debugf("amt before first step: %v", amt)
	// First fill up surcharge of this period if any.
	// ((surcharge - 1) / ppp) * ppp - surcharge
	var remainder *big.Int
	if os.surchargeReceived.Cmp(big.NewInt(0)) == 0 {
		remainder = offer.PPP()
	} else {
		temp := big.NewInt(0).Sub(os.surchargeReceived, big.NewInt(1))
		temp = big.NewInt(0).Add(big.NewInt(0).Div(temp, offer.PPP()), big.NewInt(1))
		temp = big.NewInt(0).Mul(temp, offer.PPP())
		remainder = big.NewInt(0).Sub(temp, os.surchargeReceived)
	}
	log.Debugf("remainder is %v", remainder)
	if remainder.Cmp(amt) >= 0 {
		log.Debugf("remainder greater than amt")
		os.surchargeReceived.Add(os.surchargeReceived, amt)
		amt = big.NewInt(0)
	} else {
		log.Debugf("remainder smaller than amt")
		os.surchargeReceived.Add(os.surchargeReceived, remainder)
		amt.Sub(amt, remainder)
	}
	log.Debugf("amt after first step: %v", amt)
	if amt.Cmp(big.NewInt(0)) > 0 {
		// Second fill up payment of this period.
		// received - (surcharge / ppp) * period
		temp := big.NewInt(0).Div(os.surchargeReceived, offer.PPP())
		temp = big.NewInt(0).Mul(temp, offer.Period())
		remainder = big.NewInt(0).Sub(temp, os.amtReceived)
		log.Debugf("remainder is %v", remainder)
		if remainder.Cmp(amt) >= 0 {
			log.Debugf("remainder greater than amt")
			os.amtReceived.Add(os.amtReceived, amt)
			received.Add(received, amt)
			amt = big.NewInt(0)
		} else {
			log.Debugf("remainder smaller than amt")
			os.amtReceived.Add(os.amtReceived, remainder)
			received.Add(received, remainder)
			amt.Sub(amt, remainder)
		}
		log.Debugf("amt after second step: %v", amt)
		if amt.Cmp(big.NewInt(0)) > 0 {
			// Third, fill up big period (period + ppp)
			temp := big.NewInt(0).Add(offer.Period(), offer.PPP())
			period := big.NewInt(0).Div(amt, temp)
			temp = big.NewInt(0).Mul(period, offer.PPP())
			os.surchargeReceived.Add(os.surchargeReceived, temp)
			temp = big.NewInt(0).Mul(period, offer.Period())
			os.amtReceived.Add(os.amtReceived, temp)
			temp = big.NewInt(0).Mul(period, offer.Period())
			received.Add(received, temp)
			temp = big.NewInt(0).Add(offer.Period(), offer.PPP())
			temp = big.NewInt(0).Mul(period, temp)
			amt.Sub(amt, temp)
			log.Debugf("amt after third step: %v", amt)
			if amt.Cmp(big.NewInt(0)) > 0 {
				// Lastly, fill up the last period.
				if amt.Cmp(offer.PPP()) >= 0 {
					os.surchargeReceived.Add(os.surchargeReceived, offer.PPP())
					temp := big.NewInt(0).Sub(amt, offer.PPP())
					os.amtReceived.Add(os.amtReceived, temp)
					temp = big.NewInt(0).Sub(amt, offer.PPP())
					received.Add(received, temp)
				} else {
					os.surchargeReceived.Add(os.surchargeReceived, amt)
				}
			}
		}
	}
	os.last = time.Now().Unix()
	state.redeemed = redeemed
	state.nonce = nonce
	state.voucher = voucher
	commit := func() {
		if err = chStore.Put(chKey, encodeInChState(state)); err != nil {
			log.Error("error commiting the state: %v", err.Error())
		}
		chLock.Unlock()
	}
	revert := func() {
		chLock.Unlock()
	}
	return received, commit, revert, nil
}

// Receive will output the amount recorded between last call of this function and this call of the function.
// It takes a currency id, the origin address as arguments.
// It returns the amount received and error.
func (mgr *PaymentManagerImplV1) Receive(currencyID uint64, originAddr string) (*big.Int, error) {
	log.Debugf("Receive currency %v, origin %v", currencyID, originAddr)
	store, ok := mgr.recordedStores[currencyID]
	if !ok {
		return nil, fmt.Errorf("currency id %v is not supported", currencyID)
	}
	storeLock := mgr.recordedStoresLock[currencyID]
	recordLocks := mgr.recordedEntriesLock[currencyID]
	originKey := datastore.NewKey(originAddr)
	log.Debugf("Start store lock RLock")
	storeLock.RLock()
	log.Debugf("End store lock RLock")
	exists, err := store.Has(originKey)
	if err != nil {
		storeLock.RUnlock()
		return nil, fmt.Errorf("error checking storage: %v", err.Error())
	}
	if !exists {
		storeLock.RUnlock()
		return big.NewInt(0), nil
	}
	storeLock.RUnlock()
	log.Debugf("Start store lock Lock")
	storeLock.Lock()
	log.Debugf("End store lock Lock")
	defer storeLock.Unlock()
	// Check again
	exists, err = store.Has(originKey)
	if err != nil {
		return nil, fmt.Errorf("error checking storage: %v", err.Error())
	}
	if !exists {
		return big.NewInt(0), nil
	}
	recordLock := recordLocks[originAddr]
	recordLock.Lock()
	defer recordLock.Unlock()
	data, err := store.Get(originKey)
	if err != nil {
		return nil, fmt.Errorf("error reading storage: %v", err.Error())
	}
	delete(recordLocks, originAddr)
	if err = store.Delete(originKey); err != nil {
		return nil, fmt.Errorf("error removing from storage: %v", err.Error())
	}
	return big.NewInt(0).SetBytes(data), nil
}

// gcRoutine is the garbage collection routine.
func (mgr *PaymentManagerImplV1) gcRoutine() {
	for {
		// Do a gc
		go mgr.gc()
		tc := time.After(refreshInterval)
		select {
		case <-mgr.ctx.Done():
			return
		case <-tc:
		}
	}
}

// gc is the garbage collection process.
func (mgr *PaymentManagerImplV1) gc() {
	log.Debugf("gc start")
	res, err := mgr.ListInboundChs(mgr.ctx)
	if err != nil {
		log.Errorf("error listing inbound channels: %v", err.Error())
		return
	}
	for currencyID, chAddrs := range res {
		for _, chAddr := range chAddrs {
			log.Debugf("inspect in ch %v", chAddr)
			_, _, _, _, _, _, _, err = mgr.InspectInboundCh(mgr.ctx, currencyID, chAddr)
			if err != nil {
				log.Errorf("error in refreshing inbound channel %v with currency id %v", chAddr, currencyID)
			}
		}
	}
	log.Debugf("gc end")
}

// shutdownRoutine is used to safely close the routine.
func (mgr *PaymentManagerImplV1) shutdownRoutine() {
	<-mgr.ctx.Done()
	for currencyID := range mgr.roots {
		// Close outchannels
		activeStore := mgr.outChActiveStores[currencyID]
		storeLock := mgr.outChStoresLock[currencyID]
		chStore := mgr.outChStores[currencyID]
		chLocks := mgr.outChLocks[currencyID]
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
		for _, chLock := range chLocks {
			chLock.Lock()
			defer chLock.Unlock()
		}
		if err := activeStore.Close(); err != nil {
			log.Errorf("error closing storage for currency id %v: %v", currencyID, err.Error())
		}
		if err := chStore.Close(); err != nil {
			log.Errorf("error closing storage for currency id %v: %v", currencyID, err.Error())
		}
		// Close inchannels
		activeStore = mgr.inChActiveStores[currencyID]
		storeLock = mgr.inChStoresLock[currencyID]
		chStore = mgr.inChStores[currencyID]
		chLocks = mgr.inChLocks[currencyID]
		log.Debugf("Start store lock Lock")
		storeLock.Lock()
		log.Debugf("End store lock Lock")
		defer storeLock.Unlock()
		for _, chLock := range chLocks {
			chLock.Lock()
			defer chLock.Unlock()
		}
		if err := activeStore.Close(); err != nil {
			log.Errorf("error closing storage for currency id %v: %v", currencyID, err.Error())
		}
		if err := chStore.Close(); err != nil {
			log.Errorf("error closing storage for currency id %v: %v", currencyID, err.Error())
		}
		// Close recorded
		recordedStore := mgr.recordedStores[currencyID]
		recordedStoreLock := mgr.recordedStoresLock[currencyID]
		recordedLocks := mgr.recordedEntriesLock[currencyID]
		recordedStoreLock.Lock()
		defer recordedStoreLock.Unlock()
		for _, recordLock := range recordedLocks {
			recordLock.Lock()
			defer recordLock.Unlock()
		}
		if err := recordedStore.Close(); err != nil {
			log.Errorf("error closing storage for currency id %v: %v", currencyID, err.Error())
		}
	}
}

// tryRLock will try to do an RLOCK.
// It takes a lock as the argument.
// It returns a boolean indicating success.
func tryRLock(lock golock.RWMutex) bool {
	locked := false
	for i := 0; i < lockAttempt; i++ {
		if lock.RTryLock() {
			locked = true
			break
		}
		if i < lockAttempt-1 {
			time.Sleep(lockInterval)
		}
	}
	return locked
}

// tryLock will try to do an LOCK.
// It takes a lock as the argument.
// It returns a boolean indicating success.
func tryLock(lock golock.RWMutex) bool {
	locked := false
	for i := 0; i < lockAttempt; i++ {
		if lock.TryLock() {
			locked = true
			break
		}
		if i < lockAttempt-1 {
			time.Sleep(lockInterval)
		}
	}
	return locked
}
