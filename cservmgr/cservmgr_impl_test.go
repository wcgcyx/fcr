package cservmgr

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
	"math/big"
	"os"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
)

const (
	testDS   = "./test-ds"
	testCID1 = "bafkqafkunbuxgidjomqgcidumvzxiidgnfwgkibr"
	testCID2 = "bafk2bzacecwz4gbjugyrb3orrzqidyxjibvwxojjppbgbwdwkcfocscu2chk4"
)

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	m.Run()
}

func TestNewPieceServingManagerImpl(t *testing.T) {
	ctx := context.Background()

	_, err := NewPieceServingManager(ctx, Opts{})
	assert.NotNil(t, err)

	cServMgr, err := NewPieceServingManager(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer cServMgr.Shutdown()
}

func TestServe(t *testing.T) {
	ctx := context.Background()

	cServMgr, err := NewPieceServingManager(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer cServMgr.Shutdown()

	id, err := cid.Parse(testCID1)
	assert.Nil(t, err)

	exists, _, err := cServMgr.Inspect(ctx, id, 0)
	assert.Nil(t, err)
	assert.False(t, exists)

	err = cServMgr.Serve(ctx, id, 0, big.NewInt(100))
	assert.Nil(t, err)

	err = cServMgr.Serve(ctx, id, 0, big.NewInt(101))
	assert.NotNil(t, err)

	err = cServMgr.Serve(ctx, id, 1, big.NewInt(102))
	assert.Nil(t, err)

	exists, ppb, err := cServMgr.Inspect(ctx, id, 0)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, big.NewInt(100), ppb)

	exists, ppb, err = cServMgr.Inspect(ctx, id, 1)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, big.NewInt(102), ppb)
}

func TestStop(t *testing.T) {
	ctx := context.Background()

	cServMgr, err := NewPieceServingManager(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer cServMgr.Shutdown()

	id1, err := cid.Parse(testCID1)
	assert.Nil(t, err)
	id2, err := cid.Parse(testCID2)
	assert.Nil(t, err)

	err = cServMgr.Stop(ctx, id2, 0)
	assert.NotNil(t, err)

	err = cServMgr.Stop(ctx, id1, 2)
	assert.NotNil(t, err)

	exists, _, err := cServMgr.Inspect(ctx, id1, 0)
	assert.Nil(t, err)
	assert.True(t, exists)

	err = cServMgr.Stop(ctx, id1, 0)
	assert.Nil(t, err)

	exists, _, err = cServMgr.Inspect(ctx, id1, 0)
	assert.Nil(t, err)
	assert.False(t, exists)
}

func TestList(t *testing.T) {
	ctx := context.Background()

	cServMgr, err := NewPieceServingManager(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer cServMgr.Shutdown()

	id1, err := cid.Parse(testCID1)
	assert.Nil(t, err)
	id2, err := cid.Parse(testCID2)
	assert.Nil(t, err)

	err = cServMgr.Serve(ctx, id2, 0, big.NewInt(100))
	assert.Nil(t, err)

	err = cServMgr.Serve(ctx, id2, 1, big.NewInt(101))
	assert.Nil(t, err)

	err = cServMgr.Serve(ctx, id2, 2, big.NewInt(102))
	assert.Nil(t, err)

	pieceChan, errChan := cServMgr.ListPieceIDs(ctx)
	pieces := make([]cid.Cid, 0)
	for piece := range pieceChan {
		pieces = append(pieces, piece)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 2, len(pieces))

	curChan, errChan := cServMgr.ListCurrencyIDs(ctx, id1)
	currencyIDs := make([]byte, 0)
	for currencyID := range curChan {
		currencyIDs = append(currencyIDs, currencyID)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(currencyIDs))

	curChan, errChan = cServMgr.ListCurrencyIDs(ctx, id2)
	currencyIDs = make([]byte, 0)
	for currencyID := range curChan {
		currencyIDs = append(currencyIDs, currencyID)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 3, len(currencyIDs))
}
