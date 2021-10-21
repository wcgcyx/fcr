package cidservstore

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
)

const (
	testDB = "./testdb"
	cid1   = "bafk2bzacecwz4gbjugyrb3orrzqidyxjibvwxojjppbgbwdwkcfocscu2chk4"
	cid2   = "bafk2bzaceawctypoilupvlidtasbqkzrfulepn4ci7mvin6vxohvvpwfywarq"
	cid3   = "bafk2bzacebss5wcwy2e25hrd6yqbeuccnucxfqlsnkm4tbvq7kqfljakrpfuk"
)

func TestNewPayServStore(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	ss, err := NewCIDServStoreImplV1(ctx, testDB, []uint64{0, 1, 2})
	assert.Empty(t, err)
	assert.NotEmpty(t, ss)
	cancel()
	time.Sleep(1 * time.Second)
}

func TestAddServing(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	ss, err := NewCIDServStoreImplV1(ctx, testDB, []uint64{0, 1, 2})
	assert.Empty(t, err)
	assert.NotEmpty(t, ss)

	id1, err := cid.Parse(cid1)
	assert.Empty(t, err)
	id2, err := cid.Parse(cid2)
	assert.Empty(t, err)
	id3, err := cid.Parse(cid3)
	assert.Empty(t, err)

	err = ss.Serve(3, id1, big.NewInt(1))
	assert.NotEmpty(t, err)

	err = ss.Serve(0, id1, big.NewInt(1))
	assert.Empty(t, err)

	err = ss.Serve(1, id1, big.NewInt(1))
	assert.Empty(t, err)

	err = ss.Serve(1, id2, big.NewInt(2))
	assert.Empty(t, err)

	err = ss.Serve(2, id3, big.NewInt(3))
	assert.Empty(t, err)

	err = ss.Serve(0, id1, big.NewInt(1))
	assert.NotEmpty(t, err)

	err = ss.Retire(3, id1, time.Minute)
	assert.NotEmpty(t, err)

	err = ss.Retire(0, id3, time.Minute)
	assert.NotEmpty(t, err)

	err = ss.Retire(0, id1, time.Minute)
	assert.Empty(t, err)

	err = ss.Serve(0, id1, big.NewInt(1))
	assert.NotEmpty(t, err)

	_, _, err = ss.Inspect(3, id1)
	assert.NotEmpty(t, err)

	_, _, err = ss.Inspect(0, id3)
	assert.NotEmpty(t, err)

	ppb, exp, err := ss.Inspect(0, id1)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1), ppb)
	assert.NotEmpty(t, exp)

	ppb, exp, err = ss.Inspect(1, id1)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1), ppb)
	assert.Empty(t, exp)

	res, err := ss.ListActive(ctx)
	assert.Empty(t, err)
	assert.Empty(t, res[0])
	assert.ElementsMatch(t, []cid.Cid{id1, id2}, res[1])
	assert.ElementsMatch(t, []cid.Cid{id3}, res[2])

	res, err = ss.ListRetiring(ctx)
	assert.Empty(t, err)
	assert.ElementsMatch(t, []cid.Cid{id1}, res[0])
	assert.Empty(t, res[1])
	assert.Empty(t, res[2])

	err = ss.Retire(2, id3, time.Second)
	assert.Empty(t, err)
	_, _, err = ss.Inspect(2, id3)
	assert.Empty(t, err)
	time.Sleep(2 * time.Second)
	_, _, err = ss.Inspect(2, id3)
	assert.NotEmpty(t, err)

	cancel()
	time.Sleep(1 * time.Second)
}
