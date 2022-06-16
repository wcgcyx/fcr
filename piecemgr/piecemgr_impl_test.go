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
	"context"
	"encoding/hex"
	"os"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/stretchr/testify/assert"
)

const (
	testDS             = "./test-ds"
	testFile           = "./tests/test1.txt"
	testCarFile        = "./tests/test1.car"
	testSectorFile     = "./tests/test-sector1"
	testLongSectorFile = "./tests/test-sector2"
	testCID1           = "bafkqafkunbuxgidjomqgcidumvzxiidgnfwgkibr"
	testSize1          = uint64(21)
	testCID2           = "bafk2bzacecwz4gbjugyrb3orrzqidyxjibvwxojjppbgbwdwkcfocscu2chk4"
	testSize2          = uint64(132)
	testCID3           = "bafk2bzaceawctypoilupvlidtasbqkzrfulepn4ci7mvin6vxohvvpwfywarq"
	testSize3          = uint64(396)
	testCID4           = "bafkqak2unbuxgidjomqgcidumvzxiidgnfwgkibsfyqfi2djomqgs4zameqhizltoqqgm2lmmuza"
	testSize4          = uint64(43)
	testCID5           = "bafyaa4asfyfcmakvudsaeieq6edrw7b42b3mrkobma33ysulgj6gc33qdjczt4tdpupvvnoguyjaageaqbabelqkeyavliheaiqg4dlptfjbjwzyo2djiy63ydiiqer6nvjvsxsle5ypovpxn3wongasaamkzejmbihaqaqyvsiwyieaqbacblerfq"
	testSize5          = uint64(1771804)
	testCID6           = "bafykbzacearmjwshi23oyiklke5zmnfa2rqfmzogohkvbe7uaayqysfja6yrg"
	testSize6          = uint64(5315397)
	testSubCID         = "bafk2bzacebss5wcwy2e25hrd6yqbeuccnucxfqlsnkm4tbvq7kqfljakrpfuk"
	testData1          = "54686973206973206120746573742066696c652031"
)

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	m.Run()
}

func TestNewPieceManagerImpl(t *testing.T) {
	ctx := context.Background()

	_, err := NewPieceManagerImpl(ctx, Opts{})
	assert.NotNil(t, err)

	mgr, err := NewPieceManagerImpl(ctx, Opts{PsPath: testDS + "/ps", BsPath: testDS + "/bs", BrefsPath: testDS + "/brefs"})
	assert.Nil(t, err)
	defer mgr.Shutdown()
}

func TestImportFile(t *testing.T) {
	ctx := context.Background()

	mgr, err := NewPieceManagerImpl(ctx, Opts{PsPath: testDS + "/ps", BsPath: testDS + "/bs", BrefsPath: testDS + "/brefs"})
	assert.Nil(t, err)
	defer mgr.Shutdown()

	_, err = mgr.Import(ctx, ".")
	assert.NotNil(t, err)

	id, err := mgr.Import(ctx, testFile)
	assert.Nil(t, err)
	assert.Equal(t, testCID1, id.String())

	exists, path, index, size, copy, err := mgr.Inspect(ctx, id)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, testFile, path)
	assert.Equal(t, 0, index)
	assert.Equal(t, testSize1, size)
	assert.True(t, copy)

	idOut, errOut := mgr.ListImported(ctx)
	ids := make([]cid.Cid, 0)
	for id := range idOut {
		ids = append(ids, id)
	}
	err = <-errOut
	assert.Nil(t, err)
	assert.Equal(t, 1, len(ids))

	err = mgr.Remove(ctx, id)
	assert.Nil(t, err)

	exists, _, _, _, _, err = mgr.Inspect(ctx, id)
	assert.Nil(t, err)
	assert.False(t, exists)

	idOut, errOut = mgr.ListImported(ctx)
	ids = make([]cid.Cid, 0)
	for id := range idOut {
		ids = append(ids, id)
	}
	err = <-errOut
	assert.Nil(t, err)
	assert.Equal(t, 0, len(ids))

	err = mgr.Remove(ctx, id)
	assert.NotNil(t, err)
}

func TestImportCarFile(t *testing.T) {
	ctx := context.Background()

	mgr, err := NewPieceManagerImpl(ctx, Opts{PsPath: testDS + "/ps", BsPath: testDS + "/bs", BrefsPath: testDS + "/brefs"})
	assert.Nil(t, err)
	defer mgr.Shutdown()

	_, err = mgr.ImportCar(ctx, testFile)
	assert.NotNil(t, err)

	_, err = mgr.ImportCar(ctx, ".")
	assert.NotNil(t, err)

	id, err := mgr.ImportCar(ctx, testCarFile)
	assert.Nil(t, err)
	assert.Equal(t, testCID1, id.String())

	exists, path, index, size, copy, err := mgr.Inspect(ctx, id)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, testCarFile, path)
	assert.Equal(t, 0, index)
	assert.Equal(t, testSize1, size)
	assert.True(t, copy)
}

func TestImportSector(t *testing.T) {
	ctx := context.Background()

	mgr, err := NewPieceManagerImpl(ctx, Opts{PsPath: testDS + "/ps", BsPath: testDS + "/bs", BrefsPath: testDS + "/brefs"})
	assert.Nil(t, err)
	defer mgr.Shutdown()

	ids, err := mgr.ImportSector(ctx, testSectorFile, false)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(ids))
	assert.Equal(t, testCID2, ids[0].String())
	assert.Equal(t, testCID3, ids[1].String())

	exists, path, index, size, copy, err := mgr.Inspect(ctx, ids[0])
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, testSectorFile, path)
	assert.Equal(t, 0, index)
	assert.Equal(t, testSize2, size)
	assert.False(t, copy)

	exists, path, index, size, copy, err = mgr.Inspect(ctx, ids[1])
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, testSectorFile, path)
	assert.Equal(t, 1, index)
	assert.Equal(t, testSize3, size)
	assert.False(t, copy)

	ids, err = mgr.ImportSector(ctx, testSectorFile, false)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(ids))

	ids, err = mgr.ImportSector(ctx, testLongSectorFile, false)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(ids))
	assert.Equal(t, testCID4, ids[0].String())
	assert.Equal(t, testCID5, ids[1].String())
	assert.Equal(t, testCID6, ids[2].String())

	exists, path, index, size, copy, err = mgr.Inspect(ctx, ids[0])
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, testLongSectorFile, path)
	assert.Equal(t, 0, index)
	assert.Equal(t, testSize4, size)
	assert.False(t, copy)

	exists, path, index, size, copy, err = mgr.Inspect(ctx, ids[1])
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, testLongSectorFile, path)
	assert.Equal(t, 1, index)
	assert.Equal(t, testSize5, size)
	assert.False(t, copy)

	exists, path, index, size, copy, err = mgr.Inspect(ctx, ids[2])
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, testLongSectorFile, path)
	assert.Equal(t, 2, index)
	assert.Equal(t, testSize6, size)
	assert.False(t, copy)

	err = mgr.Remove(ctx, ids[2])
	assert.Nil(t, err)
	exists, _, _, _, _, err = mgr.Inspect(ctx, ids[2])
	assert.Nil(t, err)
	assert.False(t, exists)

	ids, err = mgr.ImportSector(ctx, testLongSectorFile, true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(ids))
	assert.Equal(t, testCID6, ids[0].String())

	exists, path, index, size, copy, err = mgr.Inspect(ctx, ids[0])
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, testLongSectorFile, path)
	assert.Equal(t, 2, index)
	assert.Equal(t, testSize6, size)
	assert.True(t, copy)
}

func TestLinkSystem(t *testing.T) {
	ctx := context.Background()

	mgr, err := NewPieceManagerImpl(ctx, Opts{PsPath: testDS + "/ps", BsPath: testDS + "/bs", BrefsPath: testDS + "/brefs"})
	assert.Nil(t, err)
	defer mgr.Shutdown()

	lsys := mgr.LinkSystem()

	id, err := cid.Parse(testCID1)
	assert.Nil(t, err)

	node, err := lsys.Load(ipld.LinkContext{}, cidlink.Link{Cid: id}, basicnode.Prototype.Any)
	assert.Nil(t, err)

	assert.Equal(t, "bytes", node.Kind().String())
	data, err := node.AsBytes()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, testData1, hex.EncodeToString(data))

	id, err = cid.Parse(testCID5)
	assert.Nil(t, err)

	node, err = lsys.Load(ipld.LinkContext{}, cidlink.Link{Cid: id}, basicnode.Prototype.Any)
	assert.Nil(t, err)
	assert.Equal(t, "map", node.Kind().String())

	id, err = cid.Parse(testCID6)
	assert.Nil(t, err)

	node, err = lsys.Load(ipld.LinkContext{}, cidlink.Link{Cid: id}, basicnode.Prototype.Any)
	assert.Nil(t, err)
	assert.Equal(t, "map", node.Kind().String())
}
