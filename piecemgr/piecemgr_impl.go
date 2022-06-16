package piecemgr

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
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-cidutil"
	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	badgerds "github.com/ipfs/go-ds-badger"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	chunk "github.com/ipfs/go-ipfs-chunker"
	files "github.com/ipfs/go-ipfs-files"
	ipldformat "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs/importer/balanced"
	"github.com/ipfs/go-unixfs/importer/helpers"
	carv2 "github.com/ipld/go-car/v2"
	blockstore2 "github.com/ipld/go-car/v2/blockstore"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/mr-tron/base58"
	"github.com/multiformats/go-multihash"
	golock "github.com/viney-shih/go-lock"
	"github.com/wcgcyx/fcr/piecemgr/brefstore"
	"github.com/wcgcyx/fcr/piecemgr/sreader"
)

// PieceManagerImpl implements the PieceManager interface.
type PieceManagerImpl struct {
	// Piece info store.
	ps   datastore.Datastore
	lock golock.RWMutex

	// Block store.
	bs    blockstore.Blockstore
	brefs *brefstore.BlockRefStore

	// Shutdown function.
	shutdown func()
}

// NewPieceManagerImpl creates a new piece manager.
//
// @input - context, options.
//
// @output - piece manager, error.
func NewPieceManagerImpl(ctx context.Context, opts Opts) (*PieceManagerImpl, error) {
	log.Infof("Start piece manager...")
	// Parse options.
	if opts.PsPath == "" || opts.BsPath == "" || opts.BrefsPath == "" {
		return nil, fmt.Errorf("empty path provided")
	}
	// Open block store
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	bds, err := badgerds.NewDatastore(opts.BsPath, &dsopts)
	if err != nil {
		log.Errorf("Fail to open block store: %v", err.Error())
		return nil, err
	}
	bs := blockstore.NewBlockstore(bds)
	defer func() {
		if err != nil {
			log.Infof("Fail to start piece manager, close block store...")
			err0 := bds.Close()
			if err0 != nil {
				log.Errorf("Fail to close block store after failing to start piece manager: %v", err0.Error())
			}
		}
	}()
	// Open block reference store
	brefs, err := brefstore.NewBlockRefStore(ctx, brefstore.Opts{Path: opts.BrefsPath})
	if err != nil {
		log.Errorf("Fail to open block reference store: %v", err.Error())
		return nil, err
	}
	defer func() {
		if err != nil {
			log.Infof("Fail to start piece manager, close block reference store...")
			err0 := brefs.Shutdown(context.Background())
			if err0 != nil {
				log.Errorf("Fail to close block reference store: %v", err0.Error())
			}
		}
	}()
	// Open piece store
	ps, err := badgerds.NewDatastore(opts.PsPath, &dsopts)
	if err != nil {
		log.Errorf("Fail to open piece store: %v", err.Error())
		return nil, err
	}
	mgr := &PieceManagerImpl{
		ps:    ps,
		lock:  golock.NewCASMutex(),
		bs:    bs,
		brefs: brefs,
	}
	mgr.shutdown = func() {
		log.Infof("Stop piece store...")
		err := ps.Close()
		if err != nil {
			log.Errorf("Fail to stop piece store: %v", err.Error())
		}
		log.Infof("Stop block reference store...")
		err = brefs.Shutdown(context.Background())
		if err != nil {
			log.Errorf("Fail to stop block reference store: %v", err.Error())
		}
		log.Infof("Stop block store...")
		err = bds.Close()
		if err != nil {
			log.Errorf("Fail to stop block store: %v", err.Error())
		}
	}
	return mgr, nil
}

// Shutdown safely shuts down the component.
func (mgr *PieceManagerImpl) Shutdown() {
	log.Infof("Start shutdown...")
	mgr.shutdown()
}

// Import imports a file.
//
// @input - context, file path.
//
// @output - cid imported, error.
func (mgr *PieceManagerImpl) Import(ctx context.Context, path string) (cid.Cid, error) {
	log.Debugf("Import %v", path)
	fstat, err := os.Stat(path)
	if err != nil {
		return cid.Undef, err
	}
	fnd, err := files.NewSerialFile(path, false, fstat)
	if err != nil {
		return cid.Undef, err
	}
	defer fnd.Close()
	switch f := fnd.(type) {
	case files.File:
		// Get piece cid.
		dag := merkledag.NewDAGService(blockservice.New(mgr.bs, nil))
		bufferedDS := ipldformat.NewBufferedDAG(ctx, dag)
		prefix, err := merkledag.PrefixForCidVersion(1)
		if err != nil {
			return cid.Undef, err
		}
		prefix.MhType = uint64(multihash.BLAKE2B_MIN + 31)
		params := helpers.DagBuilderParams{
			Maxlinks:  1024,
			RawLeaves: true,
			CidBuilder: cidutil.InlineBuilder{
				Builder: prefix,
				Limit:   126,
			},
			Dagserv: bufferedDS,
		}
		db, err := params.New(chunk.NewSizeSplitter(f, int64(1<<20)))
		if err != nil {
			return cid.Undef, err
		}
		n, err := balanced.Layout(db)
		if err != nil {
			return cid.Undef, err
		}
		size, err := n.Size()
		if err != nil {
			return cid.Cid{}, err
		}
		// Check if piece exists.
		if !mgr.lock.RTryLockWithContext(ctx) {
			log.Debugf("Fail to obtain read lock over piece store")
			return cid.Undef, fmt.Errorf("fail to RLock")
		}
		release := mgr.lock.RUnlock

		key := datastore.NewKey(base58.Encode(n.Cid().Bytes()))
		exists, err := mgr.ps.Has(ctx, key)
		if err != nil {
			release()
			log.Warnf("Fail to check if piece store contains %v: %v", key, err.Error())
			return cid.Undef, err
		}
		if exists {
			release()
			return cid.Undef, fmt.Errorf("piece %v exists", n.Cid())
		}
		release()

		if !mgr.lock.TryLockWithContext(ctx) {
			log.Debugf("Fail to obtain write lock over piece store")
			return cid.Undef, fmt.Errorf("fail to Lock")
		}
		release = mgr.lock.Unlock
		defer release()

		// Check if piece exists again.
		exists, err = mgr.ps.Has(ctx, key)
		if err != nil {
			log.Warnf("Fail to check if piece store contains %v: %v", key, err.Error())
			return cid.Undef, err
		}
		if exists {
			return cid.Undef, fmt.Errorf("piece %v exists", n.Cid())
		}
		// Commit into block store
		err = bufferedDS.Commit()
		if err != nil {
			log.Warnf("Fail to commit piece %v into block store: %v", key, err.Error())
			return cid.Undef, err
		}
		// Put new piece
		val, err := encDSVal(path, 0, size, true)
		if err != nil {
			log.Errorf("Fail to encode ds value for %v-0-%v-true: %v", path, size, err.Error())
			return cid.Undef, err
		}
		err = mgr.ps.Put(ctx, key, val)
		if err != nil {
			log.Warnf("Fail to put ds value %v for %v into piece store: %v", val, key, err.Error())
			return cid.Undef, err
		}
		return n.Cid(), nil
	default:
		return cid.Undef, fmt.Errorf("given path %v is not a file.", path)
	}
}

// ImportCar imports a car file.
//
// @input - context, file path.
//
// @output - cid imported, error.
func (mgr *PieceManagerImpl) ImportCar(ctx context.Context, path string) (cid.Cid, error) {
	log.Debugf("Import car %v", path)
	robs, err := blockstore2.OpenReadOnly(path, blockstore2.UseWholeCIDs(true), carv2.ZeroLengthSectionAsEOF(true))
	if err != nil {
		return cid.Undef, err
	}
	roots, err := robs.Roots()
	if err != nil {
		return cid.Undef, err
	}
	if len(roots) != 1 {
		return cid.Undef, fmt.Errorf("cannot import car from %v with more than one root", path)
	}
	// Check if piece exists.
	if !mgr.lock.RTryLockWithContext(ctx) {
		log.Debugf("Fail to obtain read lock over piece store")
		return cid.Undef, fmt.Errorf("fail to RLock")
	}
	release := mgr.lock.RUnlock

	key := datastore.NewKey(base58.Encode(roots[0].Bytes()))
	exists, err := mgr.ps.Has(ctx, key)
	if err != nil {
		release()
		log.Warnf("Fail to check if piece store contains %v: %v", key, err.Error())
		return cid.Undef, err
	}
	if exists {
		release()
		return cid.Undef, fmt.Errorf("piece %v exists", roots[0])
	}
	release()

	if !mgr.lock.TryLockWithContext(ctx) {
		log.Debugf("Fail to obtain write lock over piece store")
		return cid.Undef, fmt.Errorf("fail to Lock")
	}
	release = mgr.lock.Unlock
	defer release()

	// Check if piece exists again.
	exists, err = mgr.ps.Has(ctx, key)
	if err != nil {
		log.Warnf("Fail to check if piece store contains %v: %v", key, err.Error())
		return cid.Undef, err
	}
	if exists {
		return cid.Undef, fmt.Errorf("piece %v exists", roots[0])
	}
	cids, err := robs.AllKeysChan(ctx)
	if err != nil {
		log.Warnf("Fail to list all keys: %v", err.Error())
		return cid.Undef, err
	}
	size := uint64(0)
	for id := range cids {
		psize, err := robs.GetSize(ctx, id)
		if err != nil {
			log.Warnf("Fail to get size for %v: %v", id.String(), err.Error())
			return cid.Undef, err
		}
		if !id.Equals(roots[0]) {
			blk, err := robs.Get(ctx, id)
			if err != nil {
				log.Warnf("Fail to get block for %v: %v", id.String(), err.Error())
				return cid.Undef, err
			}
			// Put block to block store
			err = mgr.bs.Put(ctx, blk)
			if err != nil {
				log.Warnf("Fail to put block %v into block store: %v", id.String(), err.Error())
				return cid.Undef, err
			}
		}
		size += uint64(psize)
	}
	// Put new piece
	val, err := encDSVal(path, 0, size, true)
	if err != nil {
		log.Errorf("Fail to encode ds value for %v-0-%v-true: %v", path, size, err.Error())
		return cid.Undef, err
	}
	err = mgr.ps.Put(ctx, key, val)
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v into piece store: %v", val, key, err.Error())
		return cid.Undef, err
	}
	return roots[0], nil
}

// ImportSector imports an unsealed sector copy.
//
// @input - context, file path, whether to keep a copy in datastore.
//
// @output - list of cid imported, error.
func (mgr *PieceManagerImpl) ImportSector(ctx context.Context, path string, copy bool) ([]cid.Cid, error) {
	log.Debugf("Import sector %v with copy %v", path, copy)
	// Open sector file
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	_, err = f.Seek(-4, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	tlen := make([]byte, 4)
	_, err = f.Read(tlen)
	if err != nil {
		return nil, err
	}
	// Get trailer length
	trailerLen := binary.LittleEndian.Uint32(tlen) + 4
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	// New sector reader
	sr, err := sreader.NewSectorReader(f)
	if err != nil {
		return nil, err
	}
	imported := make([]cid.Cid, 0)
	index := 0
	for {
		// Go through all pieces
		pos := sr.GetPos()
		blkr, err := carv2.NewBlockReader(sr, blockstore2.UseWholeCIDs(true), carv2.ZeroLengthSectionAsEOF(true))
		if err != nil {
			return nil, err
		}
		if len(blkr.Roots) != 1 {
			return nil, fmt.Errorf("cannot import piece with multiple roots at %v - %v", path, pos)
		}
		root := blkr.Roots[0]

		// Check if piece exists.
		if !mgr.lock.RTryLockWithContext(ctx) {
			log.Debugf("Fail to obtain read lock over piece store")
			return nil, fmt.Errorf("fail to RLock")
		}
		release := mgr.lock.RUnlock

		key := datastore.NewKey(base58.Encode(root.Bytes()))
		exists, err := mgr.ps.Has(ctx, key)
		if err != nil {
			release()
			return nil, err
		}
		release()

		if !exists {
			if !mgr.lock.TryLockWithContext(ctx) {
				log.Debugf("Fail to obtain write lock over piece store")
				return nil, fmt.Errorf("fail to Lock")
			}
			release = mgr.lock.Unlock

			// Check again
			exists, err := mgr.ps.Has(ctx, key)
			if err != nil {
				release()
				log.Warnf("Fail to check if piece store contains %v: %v", key, err.Error())
				return nil, err
			}
			if exists {
				release()
			}
		}
		// Go through all blocks
		size := uint64(0)
		for {
			blk, err := blkr.Next()
			if err != nil {
				break
			}
			// If exists, skip
			if !exists {
				if copy {
					err = mgr.bs.Put(ctx, blk)
					if err != nil {
						release()
						log.Warnf("Fail to put block into block store: %v: %v", blk.Cid().String(), err.Error())
						return nil, err
					}
				} else {
					err = mgr.brefs.PutRef(ctx, blk, path, pos, trailerLen)
					if err != nil {
						release()
						log.Warnf("Fail to put block into block reference store: %v: %v", blk.Cid().String(), err.Error())
						return nil, err
					}
				}
			}
			// TODO: Check if size calculation is correct, in car importing, the root block is not calculated.
			size += uint64(len(blk.RawData()))
		}
		// Root has finished, put it to piece store if not exists
		if !exists {
			val, err := encDSVal(path, index, size, copy)
			if err != nil {
				release()
				log.Errorf("Fail to encode value for %v-%v-%v-%v: %v", path, index, size, copy, err.Error())
				return nil, err
			}
			err = mgr.ps.Put(ctx, key, val)
			if err != nil {
				release()
				log.Warnf("Fail to put %v into piece store for %v: %v", val, key, err.Error())
				return nil, err
			}
			imported = append(imported, root)
			release()
		}
		index++
		// Advance to the next piece.
		err = sr.Advance()
		if err != nil {
			break
		}
	}
	return imported, nil
}

// ListImported lists all imported pieces.
//
// @input - context.
//
// @output - cid chan out, error chan out.
func (mgr *PieceManagerImpl) ListImported(ctx context.Context) (<-chan cid.Cid, <-chan error) {
	out := make(chan cid.Cid, dsq.KeysOnlyBufSize)
	errChan := make(chan error)
	go func() {
		log.Debugf("Listing imported")
		defer func() {
			close(out)
			close(errChan)
		}()
		if !mgr.lock.RTryLockWithContext(ctx) {
			errChan <- fmt.Errorf("fail to RLock")
			log.Debugf("Fail to obtain read lock over piece store")
			return
		}
		release := mgr.lock.RUnlock
		defer release()

		q := dsq.Query{KeysOnly: true}
		tempRes, err := mgr.ps.Query(ctx, q)
		if err != nil {
			errChan <- err
			log.Warnf("Fail to query piece store: %v", err.Error())
			return
		}
		for {
			e, ok := tempRes.NextSync()
			if !ok {
				return
			}
			if e.Error != nil {
				errChan <- e.Error
				log.Warnf("Fail to get next entry: %v", err.Error())
				return
			}
			mh, err := base58.Decode(e.Key[1:])
			if err != nil {
				errChan <- err
				log.Errorf("Fail to decode child %v, this should never happen: %v", e.Key[1:], err.Error())
				return
			}
			_, id, err := cid.CidFromBytes(mh)
			if err != nil {
				errChan <- err
				log.Errorf("Fail to create cid for %v, this should never happen: %v", e.Key[1:], err.Error())
				return
			}
			select {
			case <-ctx.Done():
				return
			case out <- id:
			}
		}
	}()
	return out, errChan
}

// Inspect inspects a piece.
//
// @input - context, cid.
//
// @output - if exists, path, index (applicable in sector), the size, whether a copy is kept, error.
func (mgr *PieceManagerImpl) Inspect(ctx context.Context, id cid.Cid) (bool, string, int, uint64, bool, error) {
	log.Debugf("Inspect piece %v", id)
	if !mgr.lock.RTryLockWithContext(ctx) {
		log.Debugf("Fail to obtain read lock over piece store")
		return false, "", 0, 0, false, fmt.Errorf("fail to RLock")
	}
	release := mgr.lock.RUnlock
	defer release()

	key := datastore.NewKey(base58.Encode(id.Bytes()))
	exists, err := mgr.ps.Has(ctx, key)
	if err != nil {
		log.Warnf("Fail to check if piece store contains %v: %v", key, err.Error())
		return false, "", 0, 0, false, err
	}
	if !exists {
		return false, "", 0, 0, false, nil
	}
	val, err := mgr.ps.Get(ctx, key)
	if err != nil {
		log.Warnf("Fail to read ds value for %v: %v", key, err.Error())
		return false, "", 0, 0, false, err
	}
	path, index, size, copy, err := decDSVal(val)
	if err != nil {
		log.Errorf("Fail to decode ds value for %v: %v", val, err.Error())
		return false, "", 0, 0, false, err
	}
	return true, path, index, size, copy, nil
}

// Remove removes a piece.
//
// @input - context, cid.
//
// @output - error.
func (mgr *PieceManagerImpl) Remove(ctx context.Context, id cid.Cid) error {
	log.Debugf("Remove piece %v", id)
	if !mgr.lock.RTryLockWithContext(ctx) {
		log.Debugf("Fail to obtain read lock over piece store")
		return fmt.Errorf("fail to RLock")
	}
	release := mgr.lock.RUnlock

	key := datastore.NewKey(base58.Encode(id.Bytes()))
	exists, err := mgr.ps.Has(ctx, key)
	if err != nil {
		release()
		log.Warnf("Fail to check if piece store contains %v: %v", key, err.Error())
		return err
	}
	if !exists {
		release()
		return fmt.Errorf("piece %v does not exist", id)
	}
	release()

	if !mgr.lock.TryLockWithContext(ctx) {
		log.Debugf("Fail to obtain write lock over piece store")
		return fmt.Errorf("fail to Lock")
	}
	release = mgr.lock.Unlock
	defer release()

	exists, err = mgr.ps.Has(ctx, key)
	if err != nil {
		log.Warnf("Fail to check if piece store contains %v: %v", key, err.Error())
		return err
	}
	if !exists {
		return fmt.Errorf("piece %v does not exist", id)
	}
	err = mgr.ps.Delete(ctx, key)
	if err != nil {
		log.Warnf("Fail to remove %v: %v", key, err.Error())
		return err
	}
	return nil
}

// LinkSystem returns the linksystem for retrieval.
//
// @output - linksystem.
func (mgr *PieceManagerImpl) LinkSystem() ipld.LinkSystem {
	lsys := cidlink.DefaultLinkSystem()
	lsys.TrustedStorage = true
	lsys.StorageReadOpener = func(lnkCtx ipld.LinkContext, lnk ipld.Link) (io.Reader, error) {
		asCidLink, ok := lnk.(cidlink.Link)
		if !ok {
			return nil, fmt.Errorf("unsupported link type")
		}
		id := asCidLink.Cid
		// First try to fetch from block store.
		blk, err := mgr.bs.Get(lnkCtx.Ctx, id)
		if err != nil {
			// Second try to fetch from block reference store.
			blk, err = mgr.brefs.Get(lnkCtx.Ctx, id)
			if err != nil {
				return nil, err
			}
		}
		return bytes.NewBuffer(blk.RawData()), nil
	}
	return lsys
}
