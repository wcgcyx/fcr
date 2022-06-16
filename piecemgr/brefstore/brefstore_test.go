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
	"encoding/binary"
	"io"
	"os"
	"testing"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	carv2 "github.com/ipld/go-car/v2"
	blockstore2 "github.com/ipld/go-car/v2/blockstore"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/piecemgr/sreader"
)

const (
	testDS     = "./test-ds"
	testSector = "./tests/test-sector"
	testCID1   = "bafk2bzacecwz4gbjugyrb3orrzqidyxjibvwxojjppbgbwdwkcfocscu2chk4"
	testCID2   = "bafk2bzaceawctypoilupvlidtasbqkzrfulepn4ci7mvin6vxohvvpwfywarq"
)

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	m.Run()
}

func TestNewBlockRefStore(t *testing.T) {
	ctx := context.Background()

	_, err := NewBlockRefStore(ctx, Opts{})
	assert.NotNil(t, err)

	s, err := NewBlockRefStore(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer s.Shutdown(ctx)
}

func TestPutBlock(t *testing.T) {
	ctx := context.Background()

	s, err := NewBlockRefStore(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer s.Shutdown(ctx)

	// No effect.
	s.HashOnRead(true)
	s.HashOnRead(false)

	// Get the first block from sector.
	f, err := os.Open(testSector)
	assert.Nil(t, err)
	defer f.Close()
	_, err = f.Seek(-4, io.SeekEnd)
	assert.Nil(t, err)
	tlen := make([]byte, 4)
	_, err = f.Read(tlen)
	assert.Nil(t, err)
	trailerLen := binary.LittleEndian.Uint32(tlen) + 4
	_, err = f.Seek(0, io.SeekStart)
	assert.Nil(t, err)
	sr, err := sreader.NewSectorReader(f)
	assert.Nil(t, err)
	blkr, err := carv2.NewBlockReader(sr, blockstore2.UseWholeCIDs(true), carv2.ZeroLengthSectionAsEOF(true))
	assert.Nil(t, err)
	pos := sr.GetPos()
	blk1, err := blkr.Next()
	assert.Nil(t, err)

	// Put the first block.
	err = s.Put(ctx, blk1)
	assert.NotNil(t, err)

	err = s.PutMany(ctx, []blocks.Block{blk1})
	assert.NotNil(t, err)

	exists, err := s.Has(ctx, blk1.Cid())
	assert.Nil(t, err)
	assert.False(t, exists)

	err = s.PutRef(ctx, blk1, testSector, pos, trailerLen)
	assert.Nil(t, err)

	exists, err = s.Has(ctx, blk1.Cid())
	assert.Nil(t, err)
	assert.True(t, exists)

	// Get the second block.
	err = sr.Advance()
	assert.Nil(t, err)
	blkr, err = carv2.NewBlockReader(sr, blockstore2.UseWholeCIDs(true), carv2.ZeroLengthSectionAsEOF(true))
	assert.Nil(t, err)
	pos = sr.GetPos()
	blk2, err := blkr.Next()
	assert.Nil(t, err)

	// Put the second block.
	err = s.PutRef(ctx, blk2, testSector, pos, trailerLen)
	assert.Nil(t, err)

	id, err := cid.Parse(testCID1)
	assert.Nil(t, err)

	blk, err := s.Get(ctx, id)
	assert.Nil(t, err)
	assert.Equal(t, blk1, blk)

	size, err := s.GetSize(ctx, id)
	assert.Nil(t, err)
	assert.Equal(t, len(blk1.RawData()), size)

	id, err = cid.Parse(testCID2)
	assert.Nil(t, err)
	blk, err = s.Get(ctx, id)
	assert.Nil(t, err)
	assert.Equal(t, blk2, blk)

	size, err = s.GetSize(ctx, id)
	assert.Nil(t, err)
	assert.Equal(t, len(blk2.RawData()), size)
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	s, err := NewBlockRefStore(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer s.Shutdown(ctx)

	idOut, err := s.AllKeysChan(ctx)
	assert.Nil(t, err)
	ids := make([]cid.Cid, 0)
	for id := range idOut {
		ids = append(ids, id)
	}
	assert.Equal(t, 2, len(ids))

	id, err := cid.Parse(testCID1)
	assert.Nil(t, err)
	err = s.DeleteBlock(ctx, id)
	assert.Nil(t, err)

	idOut, err = s.AllKeysChan(ctx)
	assert.Nil(t, err)
	ids = make([]cid.Cid, 0)
	for id := range idOut {
		ids = append(ids, id)
	}
	assert.Equal(t, 1, len(ids))
}
