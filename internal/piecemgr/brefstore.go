package piecemgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/json"
	"path/filepath"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	badgerds "github.com/ipfs/go-ds-badger"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
)

const blockRefStoreSubPath = "blockref"

// BlockRefStore is a wrapper over datastore that stores block references.
type BlockRefStore struct {
	lock  sync.RWMutex
	store datastore.Datastore
}

// blockRef is an entry in the BlockRefStore.
type blockRef struct {
	SectorPath string `json:"sector_path"`
	Offset     uint64 `json:"offset"`
	TrailerLen uint32 `json:"trailer_len"`
}

// NewBlockRefStore creates a new BlockRefStore.
// It takes a root db as the argument.
// It returns a block ref store and error.
func NewBlockRefStore(dbPath string) (*BlockRefStore, error) {
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	store, err := badgerds.NewDatastore(filepath.Join(dbPath, blockRefStoreSubPath), &dsopts)
	if err != nil {
		return nil, err
	}
	return &BlockRefStore{store: store, lock: sync.RWMutex{}}, nil
}

// Put puts an entry into the storage.
// It takes a cid, a path, an offset, the trailer length as arguments.
// It returns error.
func (brefs *BlockRefStore) Put(id cid.Cid, path string, offset uint64, trailerLen uint32) error {
	brefs.lock.Lock()
	defer brefs.lock.Unlock()
	key := dshelp.MultihashToDsKey(id.Hash())
	val, err := json.Marshal(blockRef{
		SectorPath: path,
		Offset:     offset,
		TrailerLen: trailerLen,
	})
	if err != nil {
		return err
	}
	return brefs.store.Put(key, val)
}

// Get gets an entry from the storage.
// It takes a cid as the argument.
// It returns a path, an offset, the trailer length and error.
func (brefs *BlockRefStore) Get(id cid.Cid) (string, uint64, uint32, error) {
	brefs.lock.RLock()
	defer brefs.lock.RUnlock()
	val, err := brefs.store.Get(dshelp.MultihashToDsKey(id.Hash()))
	if err != nil {
		return "", 0, 0, err
	}
	ref := blockRef{}
	err = json.Unmarshal(val, &ref)
	return ref.SectorPath, ref.Offset, ref.TrailerLen, err
}

// Remove removes an entry from the storage.
// It takes a cid as the argument.
// It returns error.
func (brefs *BlockRefStore) Remove(id cid.Cid) error {
	brefs.lock.Lock()
	defer brefs.lock.Unlock()
	return brefs.store.Delete(dshelp.MultihashToDsKey(id.Hash()))
}

// Close will safely close the database.
// It returns error.
func (brefs *BlockRefStore) Close() error {
	brefs.lock.Lock()
	defer brefs.lock.Unlock()
	return brefs.store.Close()
}
