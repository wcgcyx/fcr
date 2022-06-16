package settlemgr

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
	"time"

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

func TestNewSettlementManagerImpl(t *testing.T) {
	ctx := context.Background()

	_, err := NewSettlementManagerImpl(ctx, Opts{})
	assert.NotNil(t, err)

	settlementMgr, err := NewSettlementManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer settlementMgr.Shutdown()
}

func TestDefaultPolicy(t *testing.T) {
	ctx := context.Background()

	settlementMgr, err := NewSettlementManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer settlementMgr.Shutdown()

	err = settlementMgr.RemoveDefaultPolicy(ctx, crypto.FIL)
	assert.NotNil(t, err)

	err = settlementMgr.SetDefaultPolicy(ctx, crypto.FIL, time.Hour)
	assert.Nil(t, err)

	policy, err := settlementMgr.GetDefaultPolicy(ctx, crypto.FIL)
	assert.Nil(t, err)

	assert.Equal(t, time.Hour, policy)

	err = settlementMgr.RemoveDefaultPolicy(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, err = settlementMgr.GetDefaultPolicy(ctx, crypto.FIL)
	assert.NotNil(t, err)

	err = settlementMgr.SetDefaultPolicy(ctx, crypto.FIL, time.Hour)
	assert.Nil(t, err)
}

func TestSenderPolicy(t *testing.T) {
	ctx := context.Background()

	settlementMgr, err := NewSettlementManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer settlementMgr.Shutdown()

	policy, err := settlementMgr.GetSenderPolicy(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)
	assert.Equal(t, time.Hour, policy)

	err = settlementMgr.RemoveDefaultPolicy(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, err = settlementMgr.GetSenderPolicy(ctx, crypto.FIL, "peer1")
	assert.NotNil(t, err)

	err = settlementMgr.SetSenderPolicy(ctx, crypto.FIL, "peer1", time.Minute)
	assert.Nil(t, err)

	policy, err = settlementMgr.GetSenderPolicy(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)
	assert.Equal(t, time.Minute, policy)

	err = settlementMgr.RemoveSenderPolicy(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)

	err = settlementMgr.SetDefaultPolicy(ctx, crypto.FIL, time.Hour)
	assert.Nil(t, err)

	policy, err = settlementMgr.GetSenderPolicy(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)
	assert.Equal(t, time.Hour, policy)
}

func TestList(t *testing.T) {
	ctx := context.Background()

	settlementMgr, err := NewSettlementManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer settlementMgr.Shutdown()

	curChan, errChan := settlementMgr.ListCurrencyIDs(ctx)
	currencies := make([]byte, 0)
	for currencyID := range curChan {
		currencies = append(currencies, currencyID)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(currencies))

	policyChan, errChan := settlementMgr.ListSenders(ctx, 0)
	policies := make([]string, 0)
	for policy := range policyChan {
		policies = append(policies, policy)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(policies))

	policyChan, errChan = settlementMgr.ListSenders(ctx, crypto.FIL)
	policies = make([]string, 0)
	for policy := range policyChan {
		policies = append(policies, policy)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(policies))
}
