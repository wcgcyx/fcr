package piecemgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"
	"github.com/stretchr/testify/assert"
)

const (
	testDB             = "./tests/testdb"
	testFile           = "./tests/test1.txt"
	testCarFile        = "./tests/test1.car"
	testSectorFile     = "./tests/test-sector1"
	testLongSectorFile = "./tests/test-sector2"
)

func TestImportFile(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	mgr, err := NewPieceManagerImplV1(ctx, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr)

	id, err := mgr.Import(context.Background(), testFile)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "bafkqafkunbuxgidjomqgcidumvzxiidgnfwgkibr", id.String())

	imported, err := mgr.ListImported(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(imported))

	path, index, size, copy, err := mgr.Inspect(imported[0])
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, testFile, path)
	assert.Equal(t, 0, index)
	assert.Equal(t, int64(21), size)
	assert.Equal(t, true, copy)

	err = mgr.Remove(imported[0])
	if err != nil {
		t.Fatal(err)
	}
	imported, err = mgr.ListImported(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, len(imported))

	cancel()
	time.Sleep(1 * time.Second)
}

func TestImportCarFile(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	mgr, err := NewPieceManagerImplV1(ctx, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr)

	id, err := mgr.ImportCar(context.Background(), testCarFile)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "bafkqafkunbuxgidjomqgcidumvzxiidgnfwgkibr", id.String())

	cancel()
	time.Sleep(1 * time.Second)
}

func TestImportSector(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	mgr, err := NewPieceManagerImplV1(ctx, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr)

	ids, err := mgr.ImportSector(context.Background(), testSectorFile, false)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(ids))
	assert.Equal(t, "bafk2bzacecwz4gbjugyrb3orrzqidyxjibvwxojjppbgbwdwkcfocscu2chk4", ids[0].String())
	assert.Equal(t, "bafk2bzaceawctypoilupvlidtasbqkzrfulepn4ci7mvin6vxohvvpwfywarq", ids[1].String())

	err = mgr.Remove(ids[0])
	assert.Empty(t, err)

	ids, err = mgr.ImportSector(context.Background(), testSectorFile, true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(ids))
	assert.Equal(t, "bafk2bzacecwz4gbjugyrb3orrzqidyxjibvwxojjppbgbwdwkcfocscu2chk4", ids[0].String())
	assert.Equal(t, "bafk2bzaceawctypoilupvlidtasbqkzrfulepn4ci7mvin6vxohvvpwfywarq", ids[1].String())

	cancel()
	time.Sleep(1 * time.Second)
}

func TestLinkSystem(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	mgr, err := NewPieceManagerImplV1(ctx, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr)

	id, err := mgr.Import(context.Background(), testFile)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := mgr.ImportSector(context.Background(), testLongSectorFile, false)
	if err != nil {
		t.Fatal(err)
	}

	lsys := mgr.LinkSystem()
	node, err := lsys.Load(ipld.LinkContext{}, cidlink.Link{Cid: id}, basicnode.Prototype.Any)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "bytes", node.Kind().String())
	data, err := node.AsBytes()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "54686973206973206120746573742066696c652031", hex.EncodeToString(data))

	node, err = lsys.Load(ipld.LinkContext{}, cidlink.Link{Cid: ids[2]}, basicnode.Prototype.Any)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "map", node.Kind().String())

	subCid, err := cid.Parse("bafk2bzacebss5wcwy2e25hrd6yqbeuccnucxfqlsnkm4tbvq7kqfljakrpfuk")
	if err != nil {
		t.Fatal(err)
	}
	node, err = lsys.Load(ipld.LinkContext{}, cidlink.Link{Cid: subCid}, basicnode.Prototype.Any)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "bytes", node.Kind().String())

	cancel()
	time.Sleep(1 * time.Second)
}
