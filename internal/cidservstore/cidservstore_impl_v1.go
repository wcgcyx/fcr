package cidservstore

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

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	badgerds "github.com/ipfs/go-ds-badger"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
	logging "github.com/ipfs/go-log"
)

const (
	activeCIDServStoreSubPath = "activecidservstore"
	retireCIDServStoreSubPath = "retirecidservstore"
	gcInterval                = 6 * time.Hour
)

var log = logging.Logger("cidservestore")

// CIDServStoreImplV1 implements CIDServStore.
type CIDServStoreImplV1 struct {
	// Parent context
	ctx context.Context
	// Serving store
	storesLock   map[uint64]*sync.RWMutex
	activeStores map[uint64]datastore.Datastore
	retireStores map[uint64]datastore.TTLDatastore
}

// recordJson is used for serialization.
type recordJson struct {
	PPB string `json:"price_per_byte"`
}

// encodeRecord encodes a record into a byte array.
func encodeRecord(ppb *big.Int) []byte {
	data, _ := json.Marshal(recordJson{
		PPB: ppb.String(),
	})
	return data
}

// decodeRecord decodes a byte array into a record.
func decodeRecord(data []byte) *big.Int {
	record := recordJson{}
	json.Unmarshal(data, &record)
	ppb, _ := big.NewInt(0).SetString(record.PPB, 10)
	return ppb
}

// NewCIDServStoreImplV1 creates a CIDServStore.
// It takes a context, a root db path, a list of currency ids as arguments.
// It returns a cidservstore and error.
func NewCIDServStoreImplV1(ctx context.Context, dbPath string, currencyIDs []uint64) (CIDServStore, error) {
	ss := &CIDServStoreImplV1{
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
		store, err := badgerds.NewDatastore(filepath.Join(dbPath, fmt.Sprintf("%v-%v", activeCIDServStoreSubPath, currencyID)), &dsopts)
		if err != nil {
			return nil, err
		}
		ss.activeStores[currencyID] = store
		store, err = badgerds.NewDatastore(filepath.Join(dbPath, fmt.Sprintf("%v-%v", retireCIDServStoreSubPath, currencyID)), &dsopts)
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
// It takes a currency id, the cid, ppb (price per byte) as arguments.
// It returns error.
func (ss *CIDServStoreImplV1) Serve(currencyID uint64, id cid.Cid, ppb *big.Int) error {
	activeStore, ok := ss.activeStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	retireStore := ss.retireStores[currencyID]
	lock := ss.storesLock[currencyID]
	lock.RLock()
	key := dshelp.MultihashToDsKey(id.Hash())
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
	if err = activeStore.Put(key, encodeRecord(ppb)); err != nil {
		return fmt.Errorf("error writing to storage: %v", err.Error())
	}
	return nil
}

// Inspect will check a given serving.
// It takes a currency id, the cid as arguments.
// It returns the ppb (price per byte), expiration and error.
func (ss *CIDServStoreImplV1) Inspect(currencyID uint64, id cid.Cid) (*big.Int, int64, error) {
	activeStore, ok := ss.activeStores[currencyID]
	if !ok {
		return nil, 0, fmt.Errorf("currency id %v is not supported", currencyID)
	}
	retireStore := ss.retireStores[currencyID]
	lock := ss.storesLock[currencyID]
	lock.RLock()
	defer lock.RUnlock()
	key := dshelp.MultihashToDsKey(id.Hash())
	// Check active store
	exists, err := activeStore.Has(key)
	if err != nil {
		return nil, 0, fmt.Errorf("error checking storage: %v", err.Error())
	}
	if exists {
		data, err := activeStore.Get(key)
		if err != nil {
			return nil, 0, fmt.Errorf("error reading storage: %v", err.Error())
		}
		ppb := decodeRecord(data)
		return ppb, 0, nil
	}
	exists, err = retireStore.Has(key)
	if err != nil {
		return nil, 0, fmt.Errorf("error checking storage: %v", err.Error())
	}
	if exists {
		data, err := retireStore.Get(key)
		if err != nil {
			return nil, 0, fmt.Errorf("error reading storage: %v", err.Error())
		}
		exp, err := retireStore.GetExpiration(key)
		if err != nil {
			return nil, 0, fmt.Errorf("error reading storage: %v", err.Error())
		}
		ppb := decodeRecord(data)
		return ppb, exp.Unix(), nil
	}
	return nil, 0, fmt.Errorf("serving not existed")
}

// Retire starts the retiring process of a serving.
// It takes a currency id, the cid, time to live (ttl) as arguments.
// It returns the error.
func (ss *CIDServStoreImplV1) Retire(currencyID uint64, id cid.Cid, ttl time.Duration) error {
	activeStore, ok := ss.activeStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	retireStore := ss.retireStores[currencyID]
	lock := ss.storesLock[currencyID]
	lock.RLock()
	key := dshelp.MultihashToDsKey(id.Hash())
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
func (ss *CIDServStoreImplV1) ListActive(ctx context.Context) (map[uint64][]cid.Cid, error) {
	res := make(map[uint64][]cid.Cid)
	for currencyID, store := range ss.activeStores {
		lock := ss.storesLock[currencyID]
		lock.RLock()
		q := dsq.Query{KeysOnly: true}
		tempRes, err := store.Query(q)
		if err != nil {
			lock.RUnlock()
			return nil, err
		}
		output := make(chan cid.Cid, dsq.KeysOnlyBufSize)

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
				mh, err := dshelp.BinaryFromDsKey(datastore.RawKey(e.Key))
				if err != nil {
					log.Warningf("error parsing key from binary: %s", err)
					continue
				}
				k := cid.NewCidV1(cid.Raw, mh)
				select {
				case <-ctx.Done():
					return
				case output <- k:
				}
			}
		}()
		for toAddr := range output {
			ids, ok := res[currencyID]
			if !ok {
				ids = make([]cid.Cid, 0)
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
func (ss *CIDServStoreImplV1) ListRetiring(ctx context.Context) (map[uint64][]cid.Cid, error) {
	res := make(map[uint64][]cid.Cid)
	for currencyID, store := range ss.retireStores {
		lock := ss.storesLock[currencyID]
		lock.RLock()
		q := dsq.Query{KeysOnly: true}
		tempRes, err := store.Query(q)
		if err != nil {
			lock.RUnlock()
			return nil, err
		}
		output := make(chan cid.Cid, dsq.KeysOnlyBufSize)

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
				mh, err := dshelp.BinaryFromDsKey(datastore.RawKey(e.Key))
				if err != nil {
					log.Warningf("error parsing key from binary: %s", err)
					continue
				}
				k := cid.NewCidV1(cid.Raw, mh)
				select {
				case <-ctx.Done():
					return
				case output <- k:
				}
			}
		}()
		for toAddr := range output {
			ids, ok := res[currencyID]
			if !ok {
				ids = make([]cid.Cid, 0)
			}
			ids = append(ids, toAddr)
			res[currencyID] = ids
		}
		lock.RUnlock()
	}
	return res, nil
}

// shutdownRoutine is used to safely close the routine.
func (ss *CIDServStoreImplV1) shutdownRoutine() {
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
