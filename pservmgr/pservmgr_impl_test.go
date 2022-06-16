package pservmgr

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

	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/crypto"
)

const (
	testDS = "./test-ds"
)

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	m.Run()
}

func TestNewPaychServingManagerImpl(t *testing.T) {
	ctx := context.Background()

	_, err := NewPaychServingManagerImpl(ctx, Opts{})
	assert.NotNil(t, err)

	pservMgr, err := NewPaychServingManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer pservMgr.Shutdown()
}

func TestServe(t *testing.T) {
	ctx := context.Background()

	pservMgr, err := NewPaychServingManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer pservMgr.Shutdown()

	exists, _, _, err := pservMgr.Inspect(ctx, crypto.FIL, "peer1", "paych1")
	assert.Nil(t, err)
	assert.False(t, exists)

	err = pservMgr.Serve(ctx, crypto.FIL, "peer1", "paych1", big.NewInt(1), big.NewInt(100))
	assert.Nil(t, err)

	err = pservMgr.Serve(ctx, crypto.FIL, "peer1", "paych1", big.NewInt(1), big.NewInt(100))
	assert.NotNil(t, err)

	exists, ppp, period, err := pservMgr.Inspect(ctx, crypto.FIL, "peer1", "paych1")
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, big.NewInt(1), ppp)
	assert.Equal(t, big.NewInt(100), period)
}

func TestStop(t *testing.T) {
	ctx := context.Background()

	pservMgr, err := NewPaychServingManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer pservMgr.Shutdown()

	exists, _, _, err := pservMgr.Inspect(ctx, crypto.FIL, "peer1", "paych1")
	assert.Nil(t, err)
	assert.True(t, exists)

	err = pservMgr.Stop(ctx, 0, "peer1", "paych1")
	assert.NotNil(t, err)

	err = pservMgr.Stop(ctx, crypto.FIL, "peer2", "paych1")
	assert.NotNil(t, err)

	err = pservMgr.Stop(ctx, crypto.FIL, "peer1", "paych2")
	assert.NotNil(t, err)

	err = pservMgr.Stop(ctx, crypto.FIL, "peer1", "paych1")
	assert.Nil(t, err)

	exists, _, _, err = pservMgr.Inspect(ctx, crypto.FIL, "peer1", "paych1")
	assert.Nil(t, err)
	assert.False(t, exists)
}

func TestList(t *testing.T) {
	ctx := context.Background()

	pservMgr, err := NewPaychServingManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer pservMgr.Shutdown()

	err = pservMgr.Serve(ctx, crypto.FIL, "peer1", "paych1", big.NewInt(1), big.NewInt(100))
	assert.Nil(t, err)

	err = pservMgr.Serve(ctx, crypto.FIL, "peer1", "paych2", big.NewInt(1), big.NewInt(100))
	assert.Nil(t, err)

	err = pservMgr.Serve(ctx, crypto.FIL, "peer1", "paych3", big.NewInt(1), big.NewInt(100))
	assert.Nil(t, err)

	err = pservMgr.Serve(ctx, crypto.FIL, "peer2", "paych1", big.NewInt(1), big.NewInt(100))
	assert.Nil(t, err)

	curChan, errChan := pservMgr.ListCurrencyIDs(ctx)
	currencies := make([]byte, 0)
	for currencyID := range curChan {
		currencies = append(currencies, currencyID)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(currencies))

	senderChan, errChan := pservMgr.ListRecipients(ctx, crypto.FIL)
	senders := make([]string, 0)
	for sender := range senderChan {
		senders = append(senders, sender)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 2, len(senders))

	servingChan, errChan := pservMgr.ListServings(ctx, crypto.FIL, "peer1")
	servings := make([]string, 0)
	for serving := range servingChan {
		servings = append(servings, serving)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 3, len(servings))

	servingChan, errChan = pservMgr.ListServings(ctx, crypto.FIL, "peer2")
	servings = make([]string, 0)
	for serving := range servingChan {
		servings = append(servings, serving)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(servings))
}
