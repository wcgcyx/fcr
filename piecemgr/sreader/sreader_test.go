package sreader

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
	"encoding/binary"
	"io"
	"os"
	"testing"

	carv2 "github.com/ipld/go-car/v2"
	blockstore2 "github.com/ipld/go-car/v2/blockstore"
	"github.com/stretchr/testify/assert"
)

const (
	testSector     = "./tests/test-sector"
	testTrailerLen = uint32(7)
	testOffset1    = uint64(0)
	testOffset2    = uint64(512)
	testCID1       = "bafk2bzacecwz4gbjugyrb3orrzqidyxjibvwxojjppbgbwdwkcfocscu2chk4"
	testCID2       = "bafk2bzaceawctypoilupvlidtasbqkzrfulepn4ci7mvin6vxohvvpwfywarq"
)

func TestReadSector(t *testing.T) {
	// Open sector file
	f, err := os.Open(testSector)
	assert.Nil(t, err)
	defer f.Close()
	_, err = f.Seek(-4, io.SeekEnd)
	assert.Nil(t, err)
	tlen := make([]byte, 4)
	_, err = f.Read(tlen)
	assert.Nil(t, err)

	// Get trailer length
	trailerLen := binary.LittleEndian.Uint32(tlen) + 4
	assert.Equal(t, uint32(trailerLen), trailerLen)

	// Seek to start.
	_, err = f.Seek(0, io.SeekStart)
	assert.Nil(t, err)

	// New sector reader
	sr, err := NewSectorReader(f)
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), sr.GetPos())
	assert.Equal(t, uint64(0), sr.GetBufPos())

	blkr, err := carv2.NewBlockReader(sr, blockstore2.UseWholeCIDs(true), carv2.ZeroLengthSectionAsEOF(true))
	assert.Nil(t, err)

	pos := sr.GetPos()
	assert.Equal(t, testOffset1, pos)

	blk1, err := blkr.Next()
	assert.Nil(t, err)
	assert.Equal(t, testCID1, blk1.Cid().String())

	// Next block
	err = sr.Advance()
	assert.Nil(t, err)

	blkr, err = carv2.NewBlockReader(sr, blockstore2.UseWholeCIDs(true), carv2.ZeroLengthSectionAsEOF(true))
	assert.Nil(t, err)

	pos = sr.GetPos()
	assert.Equal(t, testOffset2, pos)

	blk2, err := blkr.Next()
	assert.Nil(t, err)
	assert.Equal(t, testCID2, blk2.Cid().String())
}
