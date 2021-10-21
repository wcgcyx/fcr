package piecemgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/extern/sector-storage/partialfile"
	"github.com/filecoin-project/lotus/extern/sector-storage/storiface"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-cidutil"
	"github.com/ipfs/go-datastore"
	badgerds "github.com/ipfs/go-ds-badger"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	chunk "github.com/ipfs/go-ipfs-chunker"
	files "github.com/ipfs/go-ipfs-files"
	ipldformat "github.com/ipfs/go-ipld-format"
	logging "github.com/ipfs/go-log"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs/importer/balanced"
	"github.com/ipfs/go-unixfs/importer/helpers"
	carv2 "github.com/ipld/go-car/v2"
	blockstore2 "github.com/ipld/go-car/v2/blockstore"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"

	_ "github.com/ipld/go-codec-dagpb"
	_ "github.com/ipld/go-ipld-prime/codec/cbor"
	_ "github.com/ipld/go-ipld-prime/codec/dagcbor"
	_ "github.com/ipld/go-ipld-prime/codec/dagjson"
	_ "github.com/ipld/go-ipld-prime/codec/json"
	_ "github.com/ipld/go-ipld-prime/codec/raw"
)

var log = logging.Logger("piecemgr")

const blockStoreSubPath = "blockstore"

// pieceManagerImplV1 implements the piece manager interface.
type pieceManagerImplV1 struct {
	// Parent context
	ctx context.Context

	// Storage for all pieces.
	ps *PieceStore

	// Storage for all block references.
	brefs *BlockRefStore

	// Storage for all blocks.
	bds datastore.Batching
	bs  blockstore.Blockstore
}

// NewPieceManagerImplV1 creates a instance of the piece manager.
// It takes a context, a root db path as arguments.
// It returns a piece manager and error.
func NewPieceManagerImplV1(ctx context.Context, dbPath string) (PieceManager, error) {
	var err error
	mgr := &pieceManagerImplV1{
		ctx: ctx,
	}
	// Initialise piece store
	mgr.ps, err = NewPieceStore(dbPath)
	if err != nil {
		return nil, err
	}
	// Initialise block reference store
	mgr.brefs, err = NewBlockRefStore(dbPath)
	if err != nil {
		return nil, err
	}
	// Initialise block store
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	mgr.bds, err = badgerds.NewDatastore(filepath.Join(dbPath, blockStoreSubPath), &dsopts)
	if err != nil {
		return nil, err
	}
	mgr.bs = blockstore.NewBlockstore(mgr.bds)
	go mgr.shutdownRoutine()
	return mgr, nil
}

// Import imports a file.
// It takes a context and a path as arguments.
// It returns the cid imported and error.
func (mgr *pieceManagerImplV1) Import(ctx context.Context, path string) (cid.Cid, error) {
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

		err = bufferedDS.Commit()
		if err != nil {
			return cid.Undef, err
		}

		// Put root to piece store
		size, err := n.Size()
		if err != nil {
			return cid.Undef, err
		}
		err = mgr.ps.Put(n.Cid(), path, 0, int64(size), true)
		if err != nil {
			return cid.Undef, err
		}
		return n.Cid(), nil
	default:
		return cid.Undef, fmt.Errorf("Given path %v is not a file.", path)
	}
}

// ImportCar imports a car file.
// It takes a context and a path as arguments.
// It returns the cid imported and error.
func (mgr *pieceManagerImplV1) ImportCar(ctx context.Context, path string) (cid.Cid, error) {
	robs, err := blockstore2.OpenReadOnly(path, blockstore2.UseWholeCIDs(true), carv2.ZeroLengthSectionAsEOF(true))
	if err != nil {
		return cid.Undef, err
	}
	roots, err := robs.Roots()
	if err != nil {
		return cid.Undef, err
	}
	if len(roots) != 1 {
		return cid.Undef, fmt.Errorf("Cannot import car from %v with more than one root", path)
	}
	cids, err := robs.AllKeysChan(ctx)
	if err != nil {
		return cid.Undef, err
	}
	size := 0
	for id := range cids {
		psize, err := robs.GetSize(id)
		if err != nil {
			return cid.Undef, err
		}
		if !id.Equals(roots[0]) {
			blk, err := robs.Get(id)
			if err != nil {
				return cid.Undef, err
			}
			// Put block to block store
			err = mgr.bs.Put(blk)
			if err != nil {
				return cid.Undef, err
			}
		}
		size += psize
	}
	// Put root to piece store
	err = mgr.ps.Put(roots[0], path, 0, int64(size), true)
	if err != nil {
		return cid.Undef, err
	}
	return roots[0], nil
}

// ImportSector imports an filecoin lotus unsealed sector copy.
// It takes a context and a path, a boolean indicating whether to keep a copy as arguments.
// It returns the a list of cids imported and error.
func (mgr *pieceManagerImplV1) ImportSector(ctx context.Context, path string, copy bool) ([]cid.Cid, error) {
	// TODO, Handle ctx
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
	trailerLen := binary.LittleEndian.Uint32(tlen) + 4
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	sr, err := NewSectorReader(f)
	if err != nil {
		return nil, err
	}
	pos := sr.GetPos()
	blkr, err := carv2.NewBlockReader(sr, blockstore2.UseWholeCIDs(true), carv2.ZeroLengthSectionAsEOF(true))
	if err != nil {
		return nil, err
	}
	res := make([]cid.Cid, 0)
	var root *blocks.Block
	for {
		blk, err := blkr.Next()
		if err != nil {
			if root == nil {
				return nil, err
			}
			// Root has finished, put it to piece store
			size := sr.GetPos() + sr.GetBufPos() - 1
			err = mgr.ps.Put((*root).Cid(), path, len(res), int64(size), copy)
			if err != nil {
				return nil, err
			}
			res = append(res, (*root).Cid())
			err = sr.Advance()
			if err != nil {
				break
			}
			pos = sr.GetPos()
			blkr, err = carv2.NewBlockReader(sr, blockstore2.UseWholeCIDs(true), carv2.ZeroLengthSectionAsEOF(true))
			if err != nil {
				return nil, err
			}
			root = nil
		} else {
			if root == nil {
				root = &blk
			}
			// Store block
			if copy {
				// Put block to block store
				err = mgr.bs.Put(blk)
				if err != nil {
					return nil, err
				}
			} else {
				// Put block to block ref store
				err = mgr.brefs.Put(blk.Cid(), path, pos, trailerLen)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return res, nil
}

// ListImported lists all imported pieces.
// It takes a context as the argument.
// It returns a list of cids imported and error.
func (mgr *pieceManagerImplV1) ListImported(ctx context.Context) ([]cid.Cid, error) {
	return mgr.ps.List(ctx)
}

// Inspect inspects a piece.
// It takes a cid as the argument.
// It returns the path, the index, the size, a boolean indicating whether a copy is kept and error.
func (mgr *pieceManagerImplV1) Inspect(id cid.Cid) (string, int, int64, bool, error) {
	return mgr.ps.Get(id)
}

// Remove removes a piece.
// It takes a cid as the argument.
// It returns error.
func (mgr *pieceManagerImplV1) Remove(id cid.Cid) error {
	_, _, _, copy, err := mgr.Inspect(id)
	if err != nil {
		return err
	}
	if copy {
		return mgr.ps.Remove(id)
	} else {
		return mgr.brefs.Remove(id)
	}
}

// LinkSystem returns the linksystem.
// It returns the linksystem.
func (mgr *pieceManagerImplV1) LinkSystem() ipld.LinkSystem {
	lsys := cidlink.DefaultLinkSystem()
	lsys.TrustedStorage = true
	lsys.StorageReadOpener = func(lnkCtx ipld.LinkContext, lnk ipld.Link) (io.Reader, error) {
		asCidLink, ok := lnk.(cidlink.Link)
		if !ok {
			return nil, fmt.Errorf("unsupported link type")
		}
		id := asCidLink.Cid
		blk, err := mgr.bs.Get(id)
		if err == nil {
			return bytes.NewBuffer(blk.RawData()), nil
		}
		// Try to fetch from block reference
		sectorPath, offset, trailerLen, err := mgr.brefs.Get(id)
		if err != nil {
			return nil, err
		}
		fs, err := os.Stat(sectorPath)
		if err != nil {
			return nil, err
		}
		pf, err := partialfile.OpenPartialFile(abi.PaddedPieceSize(fs.Size()-int64(trailerLen)), sectorPath)
		if err != nil {
			return nil, err
		}
		defer pf.Close()
		f, err := pf.Reader(storiface.PaddedByteIndex(offset), 1)
		if err != nil {
			return nil, err
		}
		sr, err := NewSectorReader(f)
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
				break
			}
			if blk.Cid().Equals(id) {
				return bytes.NewBuffer(blk.RawData()), nil
			}
		}
		return nil, fmt.Errorf("link not found")
	}
	return lsys
}

// shutdownRoutine is used to safely close the routine.
func (mgr *pieceManagerImplV1) shutdownRoutine() {
	<-mgr.ctx.Done()
	if err := mgr.ps.Close(); err != nil {
		log.Errorf("error closing piecestore database: %s", err)
	}
	if err := mgr.brefs.Close(); err != nil {
		log.Errorf("error closing blockref database: %s", err)
	}
	if err := mgr.bds.Close(); err != nil {
		log.Errorf("error closing blockstore database: %s", err)
	}
}
