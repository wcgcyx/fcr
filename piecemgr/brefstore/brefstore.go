package brefstore

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
	"os"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/storage/sealer/partialfile"
	"github.com/filecoin-project/lotus/storage/sealer/storiface"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	badgerds "github.com/ipfs/go-ds-badger"
	logging "github.com/ipfs/go-log"
	carv2 "github.com/ipld/go-car/v2"
	blockstore2 "github.com/ipld/go-car/v2/blockstore"
	"github.com/mr-tron/base58"
	golock "github.com/viney-shih/go-lock"
	"github.com/wcgcyx/fcr/piecemgr/sreader"
)

// Logger
var log = logging.Logger("blockrefstore")

// BlockRefStore is a wrapper over datastore that stores block reference.
// It implements block store interface.
type BlockRefStore struct {
	// The datastore.
	ds   datastore.Datastore
	lock golock.RWMutex
}

// NewBlockRefStore creates a new block reference store.
//
// @input - context, options.
//
// @output - block reference store, error.
func NewBlockRefStore(ctx context.Context, opts Opts) (*BlockRefStore, error) {
	// Parse options.
	if opts.Path == "" {
		return nil, fmt.Errorf("empty path provided")
	}
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	ds, err := badgerds.NewDatastore(opts.Path, &dsopts)
	if err != nil {
		return nil, err
	}
	return &BlockRefStore{ds: ds, lock: golock.NewCASMutex()}, nil
}

// Shutdown safely shuts down the store.
//
// @input - context.
//
// @output - error.
func (s *BlockRefStore) Shutdown(ctx context.Context) error {
	if !s.lock.TryLockWithContext(ctx) {
		return fmt.Errorf("fail to Lock")
	}
	return s.ds.Close()
}

// DeleteBlock implements a blockstore.Blockstore interface.
func (s *BlockRefStore) DeleteBlock(ctx context.Context, id cid.Cid) error {
	if !s.lock.RTryLockWithContext(ctx) {
		return fmt.Errorf("fail to RLock")
	}
	defer s.lock.RUnlock()
	key := datastore.NewKey(base58.Encode(id.Bytes()))
	return s.ds.Delete(ctx, key)
}

// Has implements blockstore.Blockstore interface.
func (s *BlockRefStore) Has(ctx context.Context, id cid.Cid) (bool, error) {
	if !s.lock.RTryLockWithContext(ctx) {
		return false, fmt.Errorf("fail to RLock")
	}
	defer s.lock.RUnlock()
	key := datastore.NewKey(base58.Encode(id.Bytes()))
	return s.ds.Has(ctx, key)
}

// Get implements blockstore.Blockstore interface.
func (s *BlockRefStore) Get(ctx context.Context, id cid.Cid) (blocks.Block, error) {
	if !s.lock.RTryLockWithContext(ctx) {
		return nil, fmt.Errorf("fail to RLock")
	}
	defer s.lock.RUnlock()
	// Get data
	key := datastore.NewKey(base58.Encode(id.Bytes()))
	val, err := s.ds.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	_, path, offset, trailerLen, err := decDSVal(val)
	if err != nil {
		return nil, err
	}
	fs, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	pf, err := partialfile.OpenPartialFile(abi.PaddedPieceSize(fs.Size()-int64(trailerLen)), path)
	if err != nil {
		return nil, err
	}
	defer pf.Close()
	f, err := pf.Reader(storiface.PaddedByteIndex(offset), 1)
	if err != nil {
		return nil, err
	}
	sr, err := sreader.NewSectorReader(f)
	if err != nil {
		return nil, err
	}
	blkr, err := carv2.NewBlockReader(sr, blockstore2.UseWholeCIDs(true), carv2.ZeroLengthSectionAsEOF(true))
	if err != nil {
		return nil, err
	}
	for {
		blk, err := blkr.Next()
		if err != nil {
			return nil, err
		}
		if blk.Cid().Equals(id) {
			return blk, nil
		}
	}
}

// GetSize implements blockstore.Blockstore interface.
func (s *BlockRefStore) GetSize(ctx context.Context, id cid.Cid) (int, error) {
	if !s.lock.RTryLockWithContext(ctx) {
		return 0, fmt.Errorf("fail to RLock")
	}
	defer s.lock.RUnlock()
	// Get data
	key := datastore.NewKey(base58.Encode(id.Bytes()))
	val, err := s.ds.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	size, _, _, _, err := decDSVal(val)
	if err != nil {
		return 0, err
	}
	return int(size), nil
}

// PutRef puts a block reference into the datastore.
func (s *BlockRefStore) PutRef(ctx context.Context, blk blocks.Block, path string, offset uint64, trailerLen uint32) error {
	if !s.lock.RTryLockWithContext(ctx) {
		return fmt.Errorf("fail to RLock")
	}
	defer s.lock.RUnlock()
	key := datastore.NewKey(base58.Encode(blk.Cid().Bytes()))
	val, err := encDSVal(uint64(len(blk.RawData())), path, offset, trailerLen)
	if err != nil {
		return err
	}
	return s.ds.Put(ctx, key, val)
}

// Put implements blockstore.Blockstore interface.
func (s *BlockRefStore) Put(ctx context.Context, blk blocks.Block) error {
	// BlockRefStore can only store reference.
	return fmt.Errorf("ref store can only store reference")
}

// PutMany implements blockstore.Blockstore interface.
func (s *BlockRefStore) PutMany(ctx context.Context, blks []blocks.Block) error {
	// BlockRefStore can only store reference.
	return fmt.Errorf("ref store can only store reference")
}

// AllKeysChan implements blockstore.Blockstore interface.
func (s *BlockRefStore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	if !s.lock.RTryLockWithContext(ctx) {
		return nil, fmt.Errorf("fail to RLock")
	}
	q := dsq.Query{KeysOnly: true}
	tempRes, err := s.ds.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	output := make(chan cid.Cid, dsq.KeysOnlyBufSize)
	// Start listings.
	go func() {
		defer func() {
			s.lock.RUnlock()
			tempRes.Close()
			close(output)
		}()
		for {
			e, ok := tempRes.NextSync()
			if !ok {
				return
			}
			if e.Error != nil {
				log.Errorf("Error querying from database: %v", e.Error.Error())
				return
			}
			mh, err := base58.Decode(e.Key[1:])
			if err != nil {
				log.Errorf("Fail to decode child %v, this should never happen: %v", e.Key[1:], err.Error())
				return
			}
			_, id, err := cid.CidFromBytes(mh)
			if err != nil {
				log.Errorf("Fail to create cid for %v, this should never happen: %v", e.Key[1:], err.Error())
				return
			}
			select {
			case <-ctx.Done():
				return
			case output <- id:
			}
		}
	}()
	return output, nil
}

// HashOnRead implements blockstore.Blockstore interface.
func (s *BlockRefStore) HashOnRead(enabled bool) {
}
