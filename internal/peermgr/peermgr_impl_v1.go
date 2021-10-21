package peermgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"

	"context"

	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	badgerds "github.com/ipfs/go-ds-badger"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	peermgrStoreSubPath = "peermgrstore"
)

var log = logging.Logger("peermgr")

// PeerManagerImplV1 implements PeerManager interface.
type PeerManagerImplV1 struct {
	// Parent context
	ctx context.Context
	// Peer storage
	storesLock map[uint64]*sync.RWMutex
	stores     map[uint64]datastore.Datastore
}

// recordJson is used for serialization.
type recordJson struct {
	PIDStr  string `json:"pid_str"`
	Blocked bool   `json:"blocked"`
	Success int    `json:"success"`
	Failure int    `json:"failure"`
}

// encodeRecord encodes a record into a byte array.
func encodeRecord(pid peer.ID, blocked bool, success int, failure int) []byte {
	data, _ := json.Marshal(recordJson{
		PIDStr:  pid.String(),
		Blocked: blocked,
		Success: success,
		Failure: failure,
	})
	return data
}

// decodeRecord decodes a byte array into a record.
func decodeRecord(data []byte) (peer.ID, bool, int, int) {
	record := recordJson{}
	json.Unmarshal(data, &record)
	pid, _ := peer.Decode(record.PIDStr)
	return pid, record.Blocked, record.Success, record.Failure
}

// NewPeerManagerImplV1 creates a PeerManager.
// It takes a context, a root db path, a targ, a list of currency ids as arguments.
// It returns a peer manager and error.
func NewPeerManagerImplV1(ctx context.Context, dbPath string, tag string, currencyIDs []uint64) (PeerManager, error) {
	mgr := &PeerManagerImplV1{
		ctx:        ctx,
		storesLock: make(map[uint64]*sync.RWMutex),
		stores:     make(map[uint64]datastore.Datastore),
	}
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	for _, currencyID := range currencyIDs {
		store, err := badgerds.NewDatastore(filepath.Join(dbPath, fmt.Sprintf("%v-%v-%v", peermgrStoreSubPath, currencyID, tag)), &dsopts)
		if err != nil {
			return nil, err
		}
		mgr.stores[currencyID] = store
		mgr.storesLock[currencyID] = &sync.RWMutex{}
	}
	go mgr.shutdownRoutine()
	return mgr, nil
}

// AddPeer adds a peer to the manager.
// It takes a currency id, peer address and peer id as arguments.
// It returns error.
func (mgr *PeerManagerImplV1) AddPeer(currencyID uint64, toAddr string, pid peer.ID) error {
	store, ok := mgr.stores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	lock := mgr.storesLock[currencyID]
	lock.RLock()
	key := datastore.NewKey(toAddr)
	exists, err := store.Has(key)
	if err != nil {
		lock.RUnlock()
		return fmt.Errorf("error checking storage: %v", err.Error())
	}
	blocked := false
	success := 0
	failure := 0
	if exists {
		// Peer existed.
		data, err := store.Get(key)
		if err != nil {
			lock.RUnlock()
			return fmt.Errorf("error reading storage: %v", err.Error())
		}
		_, blocked, success, failure = decodeRecord(data)
	}
	lock.RUnlock()
	lock.Lock()
	defer lock.Unlock()
	if err = store.Put(key, encodeRecord(pid, blocked, success, failure)); err != nil {
		return fmt.Errorf("error writing to storage: %v", err.Error())
	}
	return nil
}

// Record record a success or failure to a peer.
// It takes a currency id, peer address, boolean indicates whether succeed as arguments.
func (mgr *PeerManagerImplV1) Record(currencyID uint64, toAddr string, succeed bool) {
	store, ok := mgr.stores[currencyID]
	if !ok {
		log.Errorf("error in record success: currency id %v not supported", currencyID)
		return
	}
	lock := mgr.storesLock[currencyID]
	lock.RLock()
	key := datastore.NewKey(toAddr)
	data, err := store.Get(key)
	if err != nil {
		lock.RUnlock()
		log.Errorf("error reading storage: %v", err.Error())
		return
	}
	lock.RUnlock()
	lock.Lock()
	defer lock.Unlock()
	pid, blocked, success, failure := decodeRecord(data)
	if succeed {
		success += 1
	} else {
		failure += 1
	}
	if err = store.Put(key, encodeRecord(pid, blocked, success, failure)); err != nil {
		log.Errorf("error writing to storage: %v", err.Error())
	}
}

// GetPeerInfo gets the information of a peer.
// It takes a currency id, peer address as arguments.
// It returns the peer id, a boolean indicate whether it is blocked, success count, failure count and error.
func (mgr *PeerManagerImplV1) GetPeerInfo(currencyID uint64, toAddr string) (peer.ID, bool, int, int, error) {
	store, ok := mgr.stores[currencyID]
	if !ok {
		return "", false, 0, 0, fmt.Errorf("error in record success: currency id %v not supported", currencyID)
	}
	lock := mgr.storesLock[currencyID]
	lock.RLock()
	defer lock.RUnlock()
	key := datastore.NewKey(toAddr)
	data, err := store.Get(key)
	if err != nil {
		return "", false, 0, 0, err
	}
	pid, blocked, success, failure := decodeRecord(data)
	return pid, blocked, success, failure, nil
}

// BlockPeer is used to block a peer.
// It takes a currency id, peer address to block as arguments.
// It returns error.
func (mgr *PeerManagerImplV1) BlockPeer(currencyID uint64, toAddr string) error {
	store, ok := mgr.stores[currencyID]
	if !ok {
		return fmt.Errorf("error in record success: currency id %v not supported", currencyID)
	}
	lock := mgr.storesLock[currencyID]
	lock.RLock()
	key := datastore.NewKey(toAddr)
	data, err := store.Get(key)
	if err != nil {
		lock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	lock.RUnlock()
	lock.Lock()
	defer lock.Unlock()
	pid, _, success, failure := decodeRecord(data)
	if err = store.Put(key, encodeRecord(pid, true, success, failure)); err != nil {
		return fmt.Errorf("error writing to storage: %v", err.Error())
	}
	return nil
}

// UnblockPeer is used to unblock a peer.
// It takes a currency id, peer address to unblock as arguments.
// It returns error.
func (mgr *PeerManagerImplV1) UnblockPeer(currencyID uint64, toAddr string) error {
	store, ok := mgr.stores[currencyID]
	if !ok {
		return fmt.Errorf("error in record success: currency id %v not supported", currencyID)
	}
	lock := mgr.storesLock[currencyID]
	lock.RLock()
	key := datastore.NewKey(toAddr)
	data, err := store.Get(key)
	if err != nil {
		lock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	lock.RUnlock()
	lock.Lock()
	defer lock.Unlock()
	pid, _, success, failure := decodeRecord(data)
	if err = store.Put(key, encodeRecord(pid, false, success, failure)); err != nil {
		return fmt.Errorf("error writing to storage: %v", err.Error())
	}
	return nil
}

// ListPeers is used to list all the peers.
// It takes a context as the argument.
// It returns a map from currency id to list of peers and error.
func (mgr *PeerManagerImplV1) ListPeers(ctx context.Context) (map[uint64][]string, error) {
	res := make(map[uint64][]string)
	for currencyID, store := range mgr.stores {
		lock := mgr.storesLock[currencyID]
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
func (mgr *PeerManagerImplV1) shutdownRoutine() {
	<-mgr.ctx.Done()
	for currencyID, store := range mgr.stores {
		lock := mgr.storesLock[currencyID]
		lock.Lock()
		defer lock.Unlock()
		if err := store.Close(); err != nil {
			log.Errorf("error closing storage for currency id %v: %v", currencyID, err.Error())
		}
	}
}
