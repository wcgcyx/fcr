package paychmon

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
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/paychstore"
	"github.com/wcgcyx/fcr/paymgr"
	"github.com/wcgcyx/fcr/pservmgr"
	"github.com/wcgcyx/fcr/reservmgr"
	"github.com/wcgcyx/fcr/routestore"
	"github.com/wcgcyx/fcr/substore"
	"github.com/wcgcyx/fcr/trans"
)

const (
	testDS = "./test-ds"
)

var testAPI = ""
var testCh = ""

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	server := trans.NewMockFil(time.Millisecond)
	defer server.Shutdown()
	testAPI = server.GetAPI()
	m.Run()
}

func TestNewPaychMonitorImpl(t *testing.T) {
	ctx := context.Background()

	signer1, transactor1, _, pam1, shutdown, err := getNode(ctx, 0)
	assert.Nil(t, err)
	defer shutdown()

	signer2, transactor2, _, pam2, shutdown, err := getNode(ctx, 1)
	assert.Nil(t, err)
	defer shutdown()

	_, addr1, err := signer1.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, addr2, err := signer2.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, err = NewPaychMonitorImpl(ctx, pam1, transactor1, Opts{})
	assert.NotNil(t, err)

	mon1, err := NewPaychMonitorImpl(ctx, pam1, transactor1, Opts{Path: fmt.Sprintf("%v/0-%v", testDS, "paymon")})
	assert.Nil(t, err)
	defer mon1.Shutdown()

	mon2, err := NewPaychMonitorImpl(ctx, pam2, transactor2, Opts{Path: fmt.Sprintf("%v/1-%v", testDS, "paymon")})
	assert.Nil(t, err)
	defer mon2.Shutdown()

	chAddr, err := transactor1.Create(ctx, crypto.FIL, addr2, big.NewInt(100))
	assert.Nil(t, err)
	testCh = chAddr

	err = pam1.AddOutboundChannel(ctx, crypto.FIL, addr2, chAddr)
	assert.Nil(t, err)

	err = pam2.AddInboundChannel(ctx, crypto.FIL, addr1, chAddr)
	assert.Nil(t, err)

	settlement := time.Now().Add(time.Hour)

	err = mon1.Track(ctx, true, crypto.FIL, chAddr, settlement)
	assert.Nil(t, err)

	err = mon2.Track(ctx, false, crypto.FIL, chAddr, settlement)
	assert.Nil(t, err)

	time.Sleep(3 * time.Second)

	_, _, err = mon1.Check(ctx, false, crypto.FIL, chAddr)
	assert.NotNil(t, err)

	_, _, err = mon2.Check(ctx, true, crypto.FIL, chAddr)
	assert.NotNil(t, err)

	_, settlement1, err := mon1.Check(ctx, true, crypto.FIL, chAddr)
	assert.Nil(t, err)

	_, settlement2, err := mon2.Check(ctx, false, crypto.FIL, chAddr)
	assert.Nil(t, err)

	assert.True(t, settlement.Equal(settlement1))
	assert.True(t, settlement.Equal(settlement2))
}

func TestSettlement(t *testing.T) {
	ctx := context.Background()

	signer1, transactor1, _, pam1, shutdown, err := getNode(ctx, 0)
	assert.Nil(t, err)
	defer shutdown()

	signer2, transactor2, _, pam2, shutdown, err := getNode(ctx, 1)
	assert.Nil(t, err)
	defer shutdown()

	_, addr1, err := signer1.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, addr2, err := signer2.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	mon1, err := NewPaychMonitorImpl(ctx, pam1, transactor1, Opts{Path: fmt.Sprintf("%v/0-%v", testDS, "paymon"), CheckFreq: time.Second})
	assert.Nil(t, err)
	defer mon1.Shutdown()

	mon2, err := NewPaychMonitorImpl(ctx, pam2, transactor2, Opts{Path: fmt.Sprintf("%v/1-%v", testDS, "paymon"), CheckFreq: time.Second})
	assert.Nil(t, err)
	defer mon2.Shutdown()

	_, _, err = pam1.ReserveForSelf(ctx, crypto.FIL, addr2, big.NewInt(1), time.Now().Add(time.Hour), time.Hour)
	assert.Nil(t, err)

	err = transactor1.Settle(ctx, crypto.FIL, testCh)
	assert.Nil(t, err)

	err = mon1.Retire(ctx, true, crypto.FIL, testCh)
	assert.Nil(t, err)

	time.Sleep(5 * time.Second)

	_, _, err = pam1.ReserveForSelf(ctx, crypto.FIL, addr2, big.NewInt(1), time.Now().Add(time.Hour), time.Hour)
	assert.NotNil(t, err)

	err = pam2.RetireInboundChannel(ctx, crypto.FIL, addr1, testCh)
	assert.NotNil(t, err)
}

func TestExpiry(t *testing.T) {
	ctx := context.Background()

	signer1, transactor1, _, pam1, shutdown, err := getNode(ctx, 0)
	assert.Nil(t, err)
	defer shutdown()

	signer2, transactor2, _, pam2, shutdown, err := getNode(ctx, 1)
	assert.Nil(t, err)
	defer shutdown()

	_, addr1, err := signer1.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, addr2, err := signer2.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	mon1, err := NewPaychMonitorImpl(ctx, pam1, transactor1, Opts{Path: fmt.Sprintf("%v/0-%v", testDS, "paymon"), CheckFreq: time.Second})
	assert.Nil(t, err)
	defer mon1.Shutdown()

	mon2, err := NewPaychMonitorImpl(ctx, pam2, transactor2, Opts{Path: fmt.Sprintf("%v/1-%v", testDS, "paymon"), CheckFreq: time.Second})
	assert.Nil(t, err)
	defer mon2.Shutdown()

	// Create one channel.
	chAddr, err := transactor1.Create(ctx, crypto.FIL, addr2, big.NewInt(100))
	assert.Nil(t, err)

	err = pam1.AddOutboundChannel(ctx, crypto.FIL, addr2, chAddr)
	assert.Nil(t, err)

	err = pam2.AddInboundChannel(ctx, crypto.FIL, addr1, chAddr)
	assert.Nil(t, err)

	err = mon1.Track(ctx, true, crypto.FIL, chAddr, time.Now().Add(time.Second))
	assert.Nil(t, err)

	err = mon2.Track(ctx, false, crypto.FIL, chAddr, time.Now().Add(time.Second))
	assert.Nil(t, err)

	_, _, err = pam1.ReserveForSelf(ctx, crypto.FIL, addr2, big.NewInt(1), time.Now().Add(time.Hour), time.Hour)
	assert.Nil(t, err)

	time.Sleep(2 * time.Second)

	_, _, err = pam1.ReserveForSelf(ctx, crypto.FIL, addr2, big.NewInt(1), time.Now().Add(time.Hour), time.Hour)
	assert.NotNil(t, err)

	err = pam2.RetireInboundChannel(ctx, crypto.FIL, addr1, chAddr)
	assert.NotNil(t, err)

	// Create another channel.
	chAddr, err = transactor1.Create(ctx, crypto.FIL, addr2, big.NewInt(100))
	assert.Nil(t, err)

	err = pam1.AddOutboundChannel(ctx, crypto.FIL, addr2, chAddr)
	assert.Nil(t, err)

	err = pam2.AddInboundChannel(ctx, crypto.FIL, addr1, chAddr)
	assert.Nil(t, err)

	err = mon1.Track(ctx, true, crypto.FIL, chAddr, time.Now().Add(time.Second))
	assert.Nil(t, err)

	err = mon2.Track(ctx, false, crypto.FIL, chAddr, time.Now().Add(time.Second))
	assert.Nil(t, err)

	_, _, err = pam1.ReserveForSelf(ctx, crypto.FIL, addr2, big.NewInt(1), time.Now().Add(time.Hour), time.Hour)
	assert.Nil(t, err)

	err = mon1.Renew(ctx, true, crypto.FIL, chAddr, time.Now().Add(2*time.Second))
	assert.Nil(t, err)
	// Note: the new settlement time should be consistent, this is for testing only.
	err = mon2.Renew(ctx, false, crypto.FIL, chAddr, time.Now().Add(2*time.Second))
	assert.Nil(t, err)

	time.Sleep(time.Second + time.Millisecond)

	_, _, err = pam1.ReserveForSelf(ctx, crypto.FIL, addr2, big.NewInt(1), time.Now().Add(time.Hour), time.Hour)
	assert.Nil(t, err)

	// Note: the retirement of any channel should be done through the monitor, this is just for testing only.
	err = pam2.RetireInboundChannel(ctx, crypto.FIL, addr1, chAddr)
	assert.Nil(t, err)
}

// getNode is used to create a payment manager node.
func getNode(ctx context.Context, index int) (*crypto.SignerImpl, *trans.TransactorImpl, *pservmgr.PaychServingManagerImpl, *paymgr.PaymentManagerImpl, func(), error) {
	// New signer
	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "signer")})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	prv := make([]byte, 32)
	rand.Read(prv)
	signer.SetKey(ctx, crypto.FIL, crypto.SECP256K1, prv)
	// New transactor
	confidence := uint64(0)
	transactor, err := trans.NewTransactorImpl(ctx, signer, trans.Opts{FilecoinEnabled: true, FilecoinAPI: testAPI, FilecoinConfidence: &confidence})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	// New paych serving manager.
	pservMgr, err := pservmgr.NewPaychServingManagerImpl(ctx, pservmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "pservmgr")})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	// New reservation policy manager.
	reservMgr, err := reservmgr.NewReservationManagerImpl(ctx, reservmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "reservmgr")})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	// New route store
	rs, err := routestore.NewRouteStoreImpl(ctx, signer, routestore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "rs"), CleanFreq: time.Second})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	// New sub store
	subs, err := substore.NewSubStoreImpl(ctx, substore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "subs")})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	// New active in paych store.
	activeIn, err := paychstore.NewPaychStoreImpl(ctx, true, false, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "active-in")})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	// New inactive in paych store.
	inactiveIn, err := paychstore.NewPaychStoreImpl(ctx, false, false, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "inactive-in")})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	// New active out paych store.
	activeOut, err := paychstore.NewPaychStoreImpl(ctx, true, true, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "active-out")})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	// New inactive out paych store.
	inactiveOut, err := paychstore.NewPaychStoreImpl(ctx, false, true, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "inactive-out")})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	// New pam
	pam, err := paymgr.NewPaymentManagerImpl(ctx, activeOut, activeIn, inactiveOut, inactiveIn, pservMgr, reservMgr, rs, subs, transactor, signer, paymgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "paymgr"), ResCleanFreq: time.Second, CacheSyncFreq: time.Second, PeerCleanFreq: time.Second})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	shutdown := func() {
		pam.Shutdown()
		activeIn.Shutdown()
		activeOut.Shutdown()
		inactiveIn.Shutdown()
		inactiveOut.Shutdown()
		signer.Shutdown()
		reservMgr.Shutdown()
		pservMgr.Shutdown()
		rs.Shutdown()
		subs.Shutdown()
	}
	return signer, transactor, pservMgr, pam, shutdown, nil
}
