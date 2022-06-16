package renewmgr

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

func TestNewRenewManagerImpl(t *testing.T) {
	ctx := context.Background()

	_, err := NewRenewManagerImpl(ctx, Opts{})
	assert.NotNil(t, err)

	renewMgr, err := NewRenewManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer renewMgr.Shutdown()
}

func TestDefaultPolicy(t *testing.T) {
	ctx := context.Background()

	renewMgr, err := NewRenewManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer renewMgr.Shutdown()

	err = renewMgr.RemoveDefaultPolicy(ctx, crypto.FIL)
	assert.NotNil(t, err)

	err = renewMgr.SetDefaultPolicy(ctx, crypto.FIL, time.Hour)
	assert.Nil(t, err)

	policy, err := renewMgr.GetDefaultPolicy(ctx, crypto.FIL)
	assert.Nil(t, err)

	assert.Equal(t, time.Hour, policy)

	err = renewMgr.RemoveDefaultPolicy(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, err = renewMgr.GetDefaultPolicy(ctx, crypto.FIL)
	assert.NotNil(t, err)

	err = renewMgr.SetDefaultPolicy(ctx, crypto.FIL, time.Hour)
	assert.Nil(t, err)
}

func TestSenderPolicy(t *testing.T) {
	ctx := context.Background()

	renewMgr, err := NewRenewManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer renewMgr.Shutdown()

	policy, err := renewMgr.GetSenderPolicy(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)
	assert.Equal(t, time.Hour, policy)

	err = renewMgr.RemoveDefaultPolicy(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, err = renewMgr.GetSenderPolicy(ctx, crypto.FIL, "peer1")
	assert.NotNil(t, err)

	err = renewMgr.SetSenderPolicy(ctx, crypto.FIL, "peer1", time.Minute)
	assert.Nil(t, err)

	policy, err = renewMgr.GetSenderPolicy(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)
	assert.Equal(t, time.Minute, policy)

	err = renewMgr.RemoveSenderPolicy(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)

	err = renewMgr.SetDefaultPolicy(ctx, crypto.FIL, time.Hour)
	assert.Nil(t, err)

	policy, err = renewMgr.GetSenderPolicy(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)
	assert.Equal(t, time.Hour, policy)

	err = renewMgr.SetSenderPolicy(ctx, crypto.FIL, "peer1", time.Minute)
	assert.Nil(t, err)
}

func TestPaychPolicy(t *testing.T) {
	ctx := context.Background()

	renewMgr, err := NewRenewManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer renewMgr.Shutdown()

	err = renewMgr.SetPaychPolicy(ctx, crypto.FIL, "peer1", "paych1", time.Second)
	assert.Nil(t, err)

	policy, err := renewMgr.GetPaychPolicy(ctx, crypto.FIL, "peer1", "paych1")
	assert.Nil(t, err)
	assert.Equal(t, time.Second, policy)

	err = renewMgr.RemovePaychPolicy(ctx, crypto.FIL, "peer1", "paych1")
	assert.Nil(t, err)

	policy, err = renewMgr.GetPaychPolicy(ctx, crypto.FIL, "peer1", "paych1")
	assert.Nil(t, err)
	assert.Equal(t, time.Minute, policy)

	err = renewMgr.RemoveSenderPolicy(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)

	policy, err = renewMgr.GetPaychPolicy(ctx, crypto.FIL, "peer1", "paych1")
	assert.Nil(t, err)
	assert.Equal(t, time.Hour, policy)

	err = renewMgr.RemoveDefaultPolicy(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, err = renewMgr.GetPaychPolicy(ctx, crypto.FIL, "peer1", "paych1")
	assert.NotNil(t, err)

	err = renewMgr.SetPaychPolicy(ctx, crypto.FIL, "peer1", "paych1", time.Second)
	assert.Nil(t, err)

	err = renewMgr.SetSenderPolicy(ctx, crypto.FIL, "peer1", time.Minute)
	assert.Nil(t, err)

	err = renewMgr.SetDefaultPolicy(ctx, crypto.FIL, time.Hour)
	assert.Nil(t, err)
}

func TestList(t *testing.T) {
	ctx := context.Background()

	renewMgr, err := NewRenewManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer renewMgr.Shutdown()

	curChan, errChan := renewMgr.ListCurrencyIDs(ctx)
	currencies := make([]byte, 0)
	for currencyID := range curChan {
		currencies = append(currencies, currencyID)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(currencies))

	senderChan, errChan := renewMgr.ListSenders(ctx, 0)
	senders := make([]string, 0)
	for sender := range senderChan {
		senders = append(senders, sender)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(senders))

	senderChan, errChan = renewMgr.ListSenders(ctx, crypto.FIL)
	senders = make([]string, 0)
	for sender := range senderChan {
		senders = append(senders, sender)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 2, len(senders))

	paychChan, errChan := renewMgr.ListPaychs(ctx, 0, "peer1")
	paychs := make([]string, 0)
	for paych := range paychChan {
		paychs = append(paychs, paych)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(paychs))

	paychChan, errChan = renewMgr.ListPaychs(ctx, crypto.FIL, "peer0")
	paychs = make([]string, 0)
	for paych := range paychChan {
		paychs = append(paychs, paych)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(paychs))

	paychChan, errChan = renewMgr.ListPaychs(ctx, crypto.FIL, "peer1")
	paychs = make([]string, 0)
	for paych := range paychChan {
		paychs = append(paychs, paych)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 2, len(paychs))

	err = renewMgr.RemoveSenderPolicy(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)

	err = renewMgr.RemovePaychPolicy(ctx, crypto.FIL, "peer1", "paych1")
	assert.Nil(t, err)
}
