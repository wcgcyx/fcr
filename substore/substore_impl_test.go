package substore

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

func TestNewSubStoreImpl(t *testing.T) {
	ctx := context.Background()

	_, err := NewSubStoreImpl(ctx, Opts{})
	assert.NotNil(t, err)

	ss, err := NewSubStoreImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	assert.NotNil(t, ss)
	defer ss.Shutdown()
}

func TestAddSubscriber(t *testing.T) {
	ctx := context.Background()

	ss, err := NewSubStoreImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer ss.Shutdown()

	err = ss.AddSubscriber(ctx, crypto.FIL, "from1")
	assert.Nil(t, err)
	err = ss.AddSubscriber(ctx, crypto.FIL, "from1")
	assert.Nil(t, err)
	err = ss.AddSubscriber(ctx, crypto.FIL, "from2")
	assert.Nil(t, err)
	err = ss.AddSubscriber(ctx, crypto.FIL, "from3")
	assert.Nil(t, err)
}

func TestList(t *testing.T) {
	ctx := context.Background()

	ss, err := NewSubStoreImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer ss.Shutdown()

	curChan, errChan := ss.ListCurrencyIDs(ctx)
	currencyIDs := make([]byte, 0)
	for currencyID := range curChan {
		currencyIDs = append(currencyIDs, currencyID)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(currencyIDs))

	subChan, errChan := ss.ListSubscribers(ctx, 0)
	subs := make([]string, 0)
	for sub := range subChan {
		subs = append(subs, sub)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(subs))

	subChan, errChan = ss.ListSubscribers(ctx, crypto.FIL)
	subs = make([]string, 0)
	for sub := range subChan {
		subs = append(subs, sub)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 3, len(subs))

	err = ss.RemoveSubscriber(ctx, 0, "from1")
	assert.Nil(t, err)

	err = ss.RemoveSubscriber(ctx, crypto.FIL, "from2")
	assert.Nil(t, err)

	err = ss.RemoveSubscriber(ctx, crypto.FIL, "from3")
	assert.Nil(t, err)

	subChan, errChan = ss.ListSubscribers(ctx, crypto.FIL)
	subs = make([]string, 0)
	for sub := range subChan {
		subs = append(subs, sub)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(subs))

	err = ss.RemoveSubscriber(ctx, crypto.FIL, "from1")
	assert.Nil(t, err)

	subChan, errChan = ss.ListSubscribers(ctx, crypto.FIL)
	subs = make([]string, 0)
	for sub := range subChan {
		subs = append(subs, sub)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(subs))
}
