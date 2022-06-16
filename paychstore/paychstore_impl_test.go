package paychstore

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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/paychstate"
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

func TestNewPaychStoreImpl(t *testing.T) {
	ctx := context.Background()

	_, err := NewPaychStoreImpl(ctx, true, true, Opts{})
	assert.NotNil(t, err)

	s, err := NewPaychStoreImpl(ctx, true, true, Opts{Path: testDS})
	assert.Nil(t, err)
	defer s.Shutdown()

	curChan, errChan := s.ListCurrencyIDs(ctx)
	currencies := make([]byte, 0)
	for currencyID := range curChan {
		currencies = append(currencies, currencyID)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(currencies))
}

func TestUpsert(t *testing.T) {
	ctx := context.Background()

	s, err := NewPaychStoreImpl(ctx, true, true, Opts{Path: testDS})
	assert.Nil(t, err)
	defer s.Shutdown()

	s1 := paychstate.State{
		CurrencyID:         1,
		FromAddr:           "test-from-1",
		ToAddr:             "test-to-1",
		ChAddr:             "test-ch-1",
		Balance:            big.NewInt(100),
		Redeemed:           big.NewInt(2),
		Nonce:              2,
		Voucher:            "test-voucher",
		NetworkLossVoucher: "",
		StateNonce:         2,
	}
	s.Upsert(s1)
	time.Sleep(100 * time.Millisecond)

	s2, err := s.Read(ctx, 1, "test-to-1", "test-ch-1")
	assert.Nil(t, err)
	assert.Equal(t, s1, s2)

	s3 := paychstate.State{
		CurrencyID:         1,
		FromAddr:           "test-from-1",
		ToAddr:             "test-to-1",
		ChAddr:             "test-ch-1",
		Balance:            big.NewInt(100),
		Redeemed:           big.NewInt(2),
		Nonce:              2,
		Voucher:            "test-voucher",
		NetworkLossVoucher: "",
		StateNonce:         1,
	}
	s.Upsert(s3)
	time.Sleep(100 * time.Millisecond)

	s4, err := s.Read(ctx, 1, "test-to-1", "test-ch-1")
	assert.Nil(t, err)
	assert.NotEqual(t, s3, s4)
	assert.Equal(t, s1, s4)

	s5 := paychstate.State{
		CurrencyID:         1,
		FromAddr:           "test-from-2",
		ToAddr:             "test-to-2",
		ChAddr:             "test-ch-2",
		Balance:            big.NewInt(100),
		Redeemed:           big.NewInt(2),
		Nonce:              2,
		Voucher:            "test-voucher",
		NetworkLossVoucher: "",
		StateNonce:         2,
	}
	s.Upsert(s5)
	time.Sleep(100 * time.Millisecond)

	s6 := paychstate.State{
		CurrencyID:         1,
		FromAddr:           "test-from-3",
		ToAddr:             "test-to-3",
		ChAddr:             "test-ch-3",
		Balance:            big.NewInt(100),
		Redeemed:           big.NewInt(2),
		Nonce:              2,
		Voucher:            "test-voucher",
		NetworkLossVoucher: "",
		StateNonce:         2,
	}
	s.Upsert(s6)
	time.Sleep(100 * time.Millisecond)

	_, err = s.Read(ctx, 2, "test-to-1", "test-ch-1")
	assert.NotNil(t, err)

	_, err = s.Read(ctx, 1, "test-to-4", "test-ch-1")
	assert.NotNil(t, err)

	_, err = s.Read(ctx, 1, "test-to-1", "test-ch-4")
	assert.NotNil(t, err)

	s7, err := s.Read(ctx, 1, "test-to-2", "test-ch-2")
	assert.Nil(t, err)
	assert.Equal(t, s5, s7)

	s8, err := s.Read(ctx, 1, "test-to-3", "test-ch-3")
	assert.Nil(t, err)
	assert.Equal(t, s6, s8)
}

func TestList(t *testing.T) {
	ctx := context.Background()

	s, err := NewPaychStoreImpl(ctx, true, true, Opts{Path: testDS})
	assert.Nil(t, err)
	defer s.Shutdown()

	curChan, errChan := s.ListCurrencyIDs(ctx)
	currencies := make([]byte, 0)
	for currencyID := range curChan {
		currencies = append(currencies, currencyID)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(currencies))

	peerChan, errChan := s.ListPeers(ctx, 0)
	peers := make([]string, 0)
	for peer := range peerChan {
		peers = append(peers, peer)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(peers))

	peerChan, errChan = s.ListPeers(ctx, 1)
	peers = make([]string, 0)
	for peer := range peerChan {
		peers = append(peers, peer)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 3, len(peers))

	s.Remove(0, "test-to-2", "test-ch-2")
	s.Remove(1, "test-to-2", "test-ch-4")
	s.Remove(1, "test-to-2", "test-ch-2")
	time.Sleep(100 * time.Millisecond)

	peerChan, errChan = s.ListPeers(ctx, 1)
	peers = make([]string, 0)
	for peer := range peerChan {
		peers = append(peers, peer)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 2, len(peers))

	paychChan, errChan := s.ListPaychsByPeer(ctx, 0, "test-to-1")
	paychs := make([]string, 0)
	for paych := range paychChan {
		paychs = append(paychs, paych)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(paychs))

	paychChan, errChan = s.ListPaychsByPeer(ctx, 1, "test-to-1")
	paychs = make([]string, 0)
	for paych := range paychChan {
		paychs = append(paychs, paych)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(paychs))
}
