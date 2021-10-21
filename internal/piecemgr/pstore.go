package piecemgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	badgerds "github.com/ipfs/go-ds-badger"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
)

const pieceStoreSubPath = "piecestore"

// PieceStore is a wrapper over datastore that stores pieces information.
type PieceStore struct {
	lock  sync.RWMutex
	store datastore.Datastore
}

// piece is an entry in the PieceStore.
type piece struct {
	Path  string `json:"path"`
	Index int    `json:"index"`
	Size  int64  `json:"size"`
	Copy  bool   `json:"copy"`
}

// NewPieceStore creates a new PieceStore.
// It takes a root db path as the argument.
// It returns a piece store and error.
func NewPieceStore(dbPath string) (*PieceStore, error) {
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	store, err := badgerds.NewDatastore(filepath.Join(dbPath, pieceStoreSubPath), &dsopts)
	if err != nil {
		return nil, err
	}
	return &PieceStore{store: store, lock: sync.RWMutex{}}, nil
}

// Put puts an entry into the storage.
// It takes a cid, a path, an index, a size, a boolean indicating whether a copy is kept as arguments.
// It returns error.
func (ps *PieceStore) Put(id cid.Cid, path string, index int, size int64, copy bool) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	key := dshelp.MultihashToDsKey(id.Hash())
	val, err := json.Marshal(piece{
		Path:  path,
		Size:  size,
		Index: index,
		Copy:  copy,
	})
	if err != nil {
		return err
	}
	return ps.store.Put(key, val)
}

// Get gets an entry from the storage.
// It takes a cid as the argument.
// It returns the path, the index, the size, a boolean indicating whether a copy is kept and error.
func (ps *PieceStore) Get(id cid.Cid) (string, int, int64, bool, error) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	val, err := ps.store.Get(dshelp.MultihashToDsKey(id.Hash()))
	if err != nil {
		return "", 0, 0, false, err
	}
	p := piece{}
	err = json.Unmarshal(val, &p)
	return p.Path, p.Index, p.Size, p.Copy, err
}

// Remove removes an entry from the storage.
// It takes a cid as the argument.
// It returns error.
func (ps *PieceStore) Remove(id cid.Cid) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	return ps.store.Delete(dshelp.MultihashToDsKey(id.Hash()))
}

// List lists all the entries in the storage.
// It takes a context as the argument.
// It returns a list of cids imported and error.
func (ps *PieceStore) List(ctx context.Context) ([]cid.Cid, error) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	q := dsq.Query{KeysOnly: true}
	tempRes, err := ps.store.Query(q)
	if err != nil {
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
				return
			}
			if e.Error != nil {
				log.Errorf("error querying from database: %s", e.Error.Error())
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

	res := make([]cid.Cid, 0)
	for id := range output {
		res = append(res, id)
	}

	return res, nil
}

// Close will safely close the database.
// It returns error.
func (ps *PieceStore) Close() error {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	return ps.store.Close()
}
