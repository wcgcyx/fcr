package reservmgr

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

func TestNewReservationManagerImpl(t *testing.T) {
	ctx := context.Background()

	_, err := NewReservationManagerImpl(ctx, Opts{})
	assert.NotNil(t, err)

	reservMgr, err := NewReservationManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer reservMgr.Shutdown()
}

func TestDefaultPolicy(t *testing.T) {
	ctx := context.Background()

	reservMgr, err := NewReservationManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer reservMgr.Shutdown()

	err = reservMgr.RemoveDefaultPolicy(ctx, crypto.FIL)
	assert.NotNil(t, err)

	err = reservMgr.SetDefaultPolicy(ctx, crypto.FIL, false, big.NewInt(-1))
	assert.NotNil(t, err)

	err = reservMgr.SetDefaultPolicy(ctx, crypto.FIL, false, big.NewInt(100))
	assert.Nil(t, err)

	unlimited, policy, err := reservMgr.GetDefaultPolicy(ctx, crypto.FIL)
	assert.Nil(t, err)
	assert.False(t, unlimited)
	assert.Equal(t, big.NewInt(100), policy)

	err = reservMgr.RemoveDefaultPolicy(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, _, err = reservMgr.GetDefaultPolicy(ctx, crypto.FIL)
	assert.NotNil(t, err)

	err = reservMgr.SetDefaultPolicy(ctx, crypto.FIL, true, nil)
	assert.Nil(t, err)

	unlimited, policy, err = reservMgr.GetDefaultPolicy(ctx, crypto.FIL)
	assert.Nil(t, err)
	assert.True(t, unlimited)
	assert.Nil(t, policy)

	err = reservMgr.RemoveDefaultPolicy(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, _, err = reservMgr.GetDefaultPolicy(ctx, crypto.FIL)
	assert.NotNil(t, err)

	err = reservMgr.SetDefaultPolicy(ctx, crypto.FIL, false, big.NewInt(100))
	assert.Nil(t, err)
}

func TestPaychPolicy(t *testing.T) {
	ctx := context.Background()

	reservMgr, err := NewReservationManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer reservMgr.Shutdown()

	unlimited, policy, err := reservMgr.GetPaychPolicy(ctx, crypto.FIL, "ch1")
	assert.Nil(t, err)
	assert.False(t, unlimited)
	assert.Equal(t, big.NewInt(100), policy)

	err = reservMgr.RemoveDefaultPolicy(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, _, err = reservMgr.GetPaychPolicy(ctx, crypto.FIL, "ch1")
	assert.NotNil(t, err)

	err = reservMgr.SetPaychPolicy(ctx, crypto.FIL, "ch1", false, big.NewInt(101))
	assert.Nil(t, err)

	unlimited, policy, err = reservMgr.GetPaychPolicy(ctx, crypto.FIL, "ch1")
	assert.Nil(t, err)
	assert.False(t, unlimited)
	assert.Equal(t, big.NewInt(101), policy)

	err = reservMgr.RemovePaychPolicy(ctx, crypto.FIL, "ch1")
	assert.Nil(t, err)

	err = reservMgr.SetDefaultPolicy(ctx, crypto.FIL, false, big.NewInt(100))
	assert.Nil(t, err)

	unlimited, policy, err = reservMgr.GetPaychPolicy(ctx, crypto.FIL, "ch1")
	assert.Nil(t, err)
	assert.False(t, unlimited)
	assert.Equal(t, big.NewInt(100), policy)

	err = reservMgr.SetPaychPolicy(ctx, crypto.FIL, "ch1", false, big.NewInt(101))
	assert.Nil(t, err)
}

func TestPeerPolicy(t *testing.T) {
	ctx := context.Background()

	reservMgr, err := NewReservationManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer reservMgr.Shutdown()

	err = reservMgr.SetPeerPolicy(ctx, crypto.FIL, "ch1", "peer1", false, big.NewInt(102))
	assert.Nil(t, err)

	unlimited, policy, err := reservMgr.GetPeerPolicy(ctx, crypto.FIL, "ch1", "peer1")
	assert.Nil(t, err)
	assert.False(t, unlimited)
	assert.Equal(t, big.NewInt(102), policy)

	err = reservMgr.RemovePeerPolicy(ctx, crypto.FIL, "ch1", "peer1")
	assert.Nil(t, err)

	unlimited, policy, err = reservMgr.GetPeerPolicy(ctx, crypto.FIL, "ch1", "peer1")
	assert.Nil(t, err)
	assert.False(t, unlimited)
	assert.Equal(t, big.NewInt(101), policy)

	err = reservMgr.RemovePaychPolicy(ctx, crypto.FIL, "ch1")
	assert.Nil(t, err)

	unlimited, policy, err = reservMgr.GetPeerPolicy(ctx, crypto.FIL, "ch1", "peer1")
	assert.Nil(t, err)
	assert.False(t, unlimited)
	assert.Equal(t, big.NewInt(100), policy)

	err = reservMgr.RemoveDefaultPolicy(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, _, err = reservMgr.GetPeerPolicy(ctx, crypto.FIL, "ch1", "peer1")
	assert.NotNil(t, err)

	err = reservMgr.SetPeerPolicy(ctx, crypto.FIL, "ch1", "peer1", false, big.NewInt(102))
	assert.Nil(t, err)

	err = reservMgr.SetPaychPolicy(ctx, crypto.FIL, "ch1", false, big.NewInt(101))
	assert.Nil(t, err)

	err = reservMgr.SetDefaultPolicy(ctx, crypto.FIL, false, big.NewInt(100))
	assert.Nil(t, err)
}

func TestList(t *testing.T) {
	ctx := context.Background()

	reservMgr, err := NewReservationManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer reservMgr.Shutdown()

	curChan, errChan := reservMgr.ListCurrencyIDs(ctx)
	currencies := make([]byte, 0)
	for currencyID := range curChan {
		currencies = append(currencies, currencyID)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(currencies))

	paychChan, errChan := reservMgr.ListPaychs(ctx, 0)
	paychs := make([]string, 0)
	for sender := range paychChan {
		paychs = append(paychs, sender)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(paychs))

	paychChan, errChan = reservMgr.ListPaychs(ctx, crypto.FIL)
	paychs = make([]string, 0)
	for sender := range paychChan {
		paychs = append(paychs, sender)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 2, len(paychs))

	peerChan, errChan := reservMgr.ListPeers(ctx, 0, "ch1")
	peers := make([]string, 0)
	for peer := range peerChan {
		peers = append(peers, peer)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(peers))

	peerChan, errChan = reservMgr.ListPeers(ctx, crypto.FIL, "ch0")
	peers = make([]string, 0)
	for peer := range peerChan {
		peers = append(peers, peer)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(peers))

	peerChan, errChan = reservMgr.ListPeers(ctx, crypto.FIL, "ch1")
	peers = make([]string, 0)
	for peer := range peerChan {
		peers = append(peers, peer)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 2, len(peers))

	err = reservMgr.RemovePaychPolicy(ctx, crypto.FIL, "ch1")
	assert.Nil(t, err)

	err = reservMgr.RemovePeerPolicy(ctx, crypto.FIL, "ch1", "peer1")
	assert.Nil(t, err)
}
