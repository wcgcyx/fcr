package pubsubstore

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	badgerds "github.com/ipfs/go-ds-badger"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	subStoreSubPath = "substore"
	pubStoreSubPath = "pubstore"
	gcInterval      = 6 * time.Hour
)

var log = logging.Logger("pubsubstore")

// PubSubStoreImplV1 implements PubSubStore interface.
type PubSubStoreImplV1 struct {
	// Parent context
	ctx context.Context
	// sub store
	subStoresLock map[uint64]*sync.RWMutex
	subStores     map[uint64]datastore.Datastore
	// pub store
	pubStoresLock map[uint64]*sync.RWMutex
	pubStores     map[uint64]datastore.TTLDatastore
}

// NewPubSubStoreImplV1 creates a PubSubStore.
// It takes a context, a root db path, a list of currency ids as arguments.
// It returns a pubsubstore and error.
func NewPubSubStoreImplV1(ctx context.Context, dbPath string, currencyIDs []uint64) (PubSubStore, error) {
	pss := &PubSubStoreImplV1{
		ctx:           ctx,
		subStoresLock: make(map[uint64]*sync.RWMutex),
		subStores:     make(map[uint64]datastore.Datastore),
		pubStoresLock: make(map[uint64]*sync.RWMutex),
		pubStores:     make(map[uint64]datastore.TTLDatastore),
	}
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	dsopts.GcInterval = gcInterval
	for _, currencyID := range currencyIDs {
		store, err := badgerds.NewDatastore(filepath.Join(dbPath, fmt.Sprintf("%v-%v", subStoreSubPath, currencyID)), &dsopts)
		if err != nil {
			return nil, err
		}
		pss.subStores[currencyID] = store
		pss.subStoresLock[currencyID] = &sync.RWMutex{}
		store, err = badgerds.NewDatastore(filepath.Join(dbPath, fmt.Sprintf("%v-%v", pubStoreSubPath, currencyID)), &dsopts)
		if err != nil {
			return nil, err
		}
		pss.pubStores[currencyID] = store
		pss.pubStoresLock[currencyID] = &sync.RWMutex{}
	}
	go pss.shutdownRoutine()
	return pss, nil
}

// AddSubscribing adds a subscribing to the store.
// It takes a currency id, the peer id as arguments.
// It returns error.
func (pss *PubSubStoreImplV1) AddSubscribing(currencyID uint64, pid peer.ID) error {
	store, ok := pss.subStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	lock := pss.subStoresLock[currencyID]
	lock.Lock()
	defer lock.Unlock()
	return store.Put(datastore.NewKey(pid.String()), []byte{0})
}

// RemoveSubscribing removes a subscribing from the store.
// It takes a currency id, the peer id as arguments.
// It returns error.
func (pss *PubSubStoreImplV1) RemoveSubscribing(currencyID uint64, pid peer.ID) error {
	store, ok := pss.subStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	lock := pss.subStoresLock[currencyID]
	lock.Lock()
	defer lock.Unlock()
	return store.Delete(datastore.NewKey(pid.String()))
}

// AddSubscriber adds a subscriber to the store.
// It takes a currency id, the peer id as arguments.
// It returns error.
func (pss *PubSubStoreImplV1) AddSubscriber(currencyID uint64, pid peer.ID, ttl time.Duration) error {
	store, ok := pss.pubStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	lock := pss.pubStoresLock[currencyID]
	lock.Lock()
	defer lock.Unlock()
	return store.PutWithTTL(datastore.NewKey(pid.String()), []byte{0}, ttl)
}

// ListSubscribings lists all the subscribings.
// It returns a map from currency id to a list of peer ids and error.
func (pss *PubSubStoreImplV1) ListSubscribings(ctx context.Context) (map[uint64][]peer.ID, error) {
	res := make(map[uint64][]peer.ID)
	for currencyID, store := range pss.subStores {
		lock := pss.subStoresLock[currencyID]
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
		for idStr := range output {
			ids, ok := res[currencyID]
			if !ok {
				ids = make([]peer.ID, 0)
			}
			pid, _ := peer.Decode(idStr)
			ids = append(ids, pid)
			res[currencyID] = ids
		}
		lock.RUnlock()
	}
	return res, nil
}

// ListSubscribers lists all the subscribers.
// It returns a map from currency id to a list of peer ids and error.
func (pss *PubSubStoreImplV1) ListSubscribers(ctx context.Context) (map[uint64][]peer.ID, error) {
	res := make(map[uint64][]peer.ID)
	for currencyID, store := range pss.pubStores {
		lock := pss.pubStoresLock[currencyID]
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
		for idStr := range output {
			ids, ok := res[currencyID]
			if !ok {
				ids = make([]peer.ID, 0)
			}
			pid, _ := peer.Decode(idStr)
			ids = append(ids, pid)
			res[currencyID] = ids
		}
		lock.RUnlock()
	}
	return res, nil
}

// shutdownRoutine is used to safely close the routine.
func (pss *PubSubStoreImplV1) shutdownRoutine() {
	<-pss.ctx.Done()
	for currencyID, store := range pss.subStores {
		lock := pss.subStoresLock[currencyID]
		lock.Lock()
		defer lock.Unlock()
		if err := store.Close(); err != nil {
			log.Errorf("error closing storage for currency id %v: %v", currencyID, err.Error())
		}
	}
	for currencyID, store := range pss.pubStores {
		lock := pss.pubStoresLock[currencyID]
		lock.Lock()
		defer lock.Unlock()
		if err := store.Close(); err != nil {
			log.Errorf("error closing storage for currency id %v: %v", currencyID, err.Error())
		}
	}
}
