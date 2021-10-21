package payservstore

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"path/filepath"
	"sync"
	"time"

	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	badgerds "github.com/ipfs/go-ds-badger"
	logging "github.com/ipfs/go-log"
)

const (
	activePayServStoreSubPath = "activepayservstore"
	retirePayServStoreSubPath = "retirepayservstore"
	gcInterval                = 6 * time.Hour
)

var log = logging.Logger("payservestore")

// PayServStoreImplV1 implements PayServStore.
type PayServStoreImplV1 struct {
	// Parent context
	ctx context.Context
	// Serving store
	storesLock   map[uint64]*sync.RWMutex
	activeStores map[uint64]datastore.Datastore
	retireStores map[uint64]datastore.TTLDatastore
}

// recordJson is used for serialization.
type recordJson struct {
	PPP    string `json:"price_per_period"`
	Period string `json:"period"`
}

// encodeRecord encodes a record into a byte array.
func encodeRecord(ppp *big.Int, period *big.Int) []byte {
	data, _ := json.Marshal(recordJson{
		PPP:    ppp.String(),
		Period: period.String(),
	})
	return data
}

// decodeRecord decodes a byte array into a record.
func decodeRecord(data []byte) (*big.Int, *big.Int) {
	record := recordJson{}
	json.Unmarshal(data, &record)
	ppp, _ := big.NewInt(0).SetString(record.PPP, 10)
	period, _ := big.NewInt(0).SetString(record.Period, 10)
	return ppp, period
}

// NewPayServStoreImplV1 creates a PayServStore.
// It takes a context, a root db path, a list of currency ids as arguments.
// It returns a payservstore and error.
func NewPayServStoreImplV1(ctx context.Context, dbPath string, currencyIDs []uint64) (PayServStore, error) {
	ss := &PayServStoreImplV1{
		ctx:          ctx,
		storesLock:   make(map[uint64]*sync.RWMutex),
		activeStores: make(map[uint64]datastore.Datastore),
		retireStores: make(map[uint64]datastore.TTLDatastore),
	}
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	dsopts.GcInterval = gcInterval
	for _, currencyID := range currencyIDs {
		store, err := badgerds.NewDatastore(filepath.Join(dbPath, fmt.Sprintf("%v-%v", activePayServStoreSubPath, currencyID)), &dsopts)
		if err != nil {
			return nil, err
		}
		ss.activeStores[currencyID] = store
		store, err = badgerds.NewDatastore(filepath.Join(dbPath, fmt.Sprintf("%v-%v", retirePayServStoreSubPath, currencyID)), &dsopts)
		if err != nil {
			return nil, err
		}
		ss.retireStores[currencyID] = store
		ss.storesLock[currencyID] = &sync.RWMutex{}
	}
	go ss.shutdownRoutine()
	return ss, nil
}

// Serve will start serving a address.
// It takes a currency id, the to address, ppp (price per period), period as arguments.
// It returns error.
func (ss *PayServStoreImplV1) Serve(currencyID uint64, toAddr string, ppp *big.Int, period *big.Int) error {
	activeStore, ok := ss.activeStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	retireStore := ss.retireStores[currencyID]
	lock := ss.storesLock[currencyID]
	lock.RLock()
	key := datastore.NewKey(toAddr)
	// Check active store
	exists, err := activeStore.Has(key)
	if err != nil {
		lock.RUnlock()
		return fmt.Errorf("error checking storage: %v", err.Error())
	}
	if exists {
		lock.RUnlock()
		return fmt.Errorf("key already exists in active store")
	}
	exists, err = retireStore.Has(key)
	if err != nil {
		lock.RUnlock()
		return fmt.Errorf("error checking storage: %v", err.Error())
	}
	if exists {
		lock.RUnlock()
		return fmt.Errorf("key already exists in retire store")
	}
	lock.RUnlock()
	lock.Lock()
	defer lock.Unlock()
	if err = activeStore.Put(key, encodeRecord(ppp, period)); err != nil {
		return fmt.Errorf("error writing to storage: %v", err.Error())
	}
	return nil
}

// Inspect will check a given serving.
// It takes a currency id, the to address as arguments.
// It returns the ppp (price per period), period, expiration and error.
func (ss *PayServStoreImplV1) Inspect(currencyID uint64, toAddr string) (*big.Int, *big.Int, int64, error) {
	activeStore, ok := ss.activeStores[currencyID]
	if !ok {
		return nil, nil, 0, fmt.Errorf("currency id %v is not supported", currencyID)
	}
	retireStore := ss.retireStores[currencyID]
	lock := ss.storesLock[currencyID]
	lock.RLock()
	defer lock.RUnlock()
	key := datastore.NewKey(toAddr)
	// Check active store
	exists, err := activeStore.Has(key)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("error checking storage: %v", err.Error())
	}
	if exists {
		data, err := activeStore.Get(key)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("error reading storage: %v", err.Error())
		}
		ppp, period := decodeRecord(data)
		return ppp, period, 0, nil
	}
	exists, err = retireStore.Has(key)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("error checking storage: %v", err.Error())
	}
	if exists {
		data, err := retireStore.Get(key)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("error reading storage: %v", err.Error())
		}
		exp, err := retireStore.GetExpiration(key)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("error reading storage: %v", err.Error())
		}
		ppp, period := decodeRecord(data)
		return ppp, period, exp.Unix(), nil
	}
	return nil, nil, 0, fmt.Errorf("serving not existed")
}

// Retire starts the retiring process of a serving.
// It takes a currency id, the to address, time to live (ttl) as arguments.
// It returns the error.
func (ss *PayServStoreImplV1) Retire(currencyID uint64, toAddr string, ttl time.Duration) error {
	activeStore, ok := ss.activeStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	retireStore := ss.retireStores[currencyID]
	lock := ss.storesLock[currencyID]
	lock.RLock()
	key := datastore.NewKey(toAddr)
	// Check active store
	exists, err := activeStore.Has(key)
	if err != nil {
		lock.RUnlock()
		return fmt.Errorf("error checking storage: %v", err.Error())
	}
	if !exists {
		lock.RUnlock()
		return fmt.Errorf("key not exists in active store")
	}
	data, err := activeStore.Get(key)
	if err != nil {
		lock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	lock.RUnlock()
	lock.Lock()
	defer lock.Unlock()
	// Remove from active store
	if err = activeStore.Delete(key); err != nil {
		return fmt.Errorf("error removing from storage: %v", err.Error())
	}
	// Add to retire storage
	if err = retireStore.PutWithTTL(key, data, ttl); err != nil {
		return fmt.Errorf("error writing to storage: %v", err.Error())
	}
	return nil
}

// ListActive will lists all the active servings.
// It takes a context as the argument.
// It returns a map from currency id to list of active servings and error.
func (ss *PayServStoreImplV1) ListActive(ctx context.Context) (map[uint64][]string, error) {
	res := make(map[uint64][]string)
	for currencyID, store := range ss.activeStores {
		lock := ss.storesLock[currencyID]
		lock.RLock()
		q := dsq.Query{KeysOnly: true}
		tempRes, err := store.Query(q)
		if err != nil {
			lock.RUnlock()
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
		lock.RUnlock()
	}
	return res, nil
}

// ListRetiring will lists all the retiring servings.
// It takes a context as the argument.
// It returns a map from currency id to list of retiring servings and error.
func (ss *PayServStoreImplV1) ListRetiring(ctx context.Context) (map[uint64][]string, error) {
	res := make(map[uint64][]string)
	for currencyID, store := range ss.retireStores {
		lock := ss.storesLock[currencyID]
		lock.RLock()
		q := dsq.Query{KeysOnly: true}
		tempRes, err := store.Query(q)
		if err != nil {
			lock.RUnlock()
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
		lock.RUnlock()
	}
	return res, nil
}

// shutdownRoutine is used to safely close the routine.
func (ss *PayServStoreImplV1) shutdownRoutine() {
	<-ss.ctx.Done()
	for currencyID, store := range ss.activeStores {
		lock := ss.storesLock[currencyID]
		lock.Lock()
		defer lock.Unlock()
		if err := store.Close(); err != nil {
			log.Errorf("error closing storage for currency id %v: %v", currencyID, err.Error())
		}
		store = ss.retireStores[currencyID]
		if err := store.Close(); err != nil {
			log.Errorf("error closing storage for currency id %v: %v", currencyID, err.Error())
		}
	}
}
