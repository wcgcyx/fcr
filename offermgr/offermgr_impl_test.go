package offermgr

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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestNewOfferManager(t *testing.T) {
	ctx := context.Background()

	_, err := NewOfferManagerImpl(ctx, Opts{})
	assert.NotNil(t, err)

	offerMgr, err := NewOfferManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer offerMgr.Shutdown()

	nonce, err := offerMgr.GetNonce(ctx)
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), nonce)
}

func TestSetNonce(t *testing.T) {
	ctx := context.Background()

	offerMgr, err := NewOfferManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer offerMgr.Shutdown()

	nonce, err := offerMgr.GetNonce(ctx)
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), nonce)

	err = offerMgr.SetNonce(ctx, 100)
	assert.Nil(t, err)

	nonce, err = offerMgr.GetNonce(ctx)
	assert.Nil(t, err)
	assert.Equal(t, uint64(101), nonce)
}
