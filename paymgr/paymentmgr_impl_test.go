package paymgr

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
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/paychstore"
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
var testResCh = ""
var testResID = uint64(0)

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	server := trans.NewMockFil(time.Millisecond)
	defer server.Shutdown()
	testAPI = server.GetAPI()
	m.Run()
}

func TestTwoNodes(t *testing.T) {
	ctx := context.Background()

	signer1, trans1, _, _, pam1, shutdown, err := getNode(ctx, 0)
	assert.Nil(t, err)
	defer shutdown()

	signer2, _, _, _, pam2, shutdown, err := getNode(ctx, 1)
	assert.Nil(t, err)
	defer shutdown()

	_, addr1, err := signer1.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, addr2, err := signer2.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	// A payment channel from node 1 to node 2
	chAddr, err := trans1.Create(ctx, crypto.FIL, addr2, big.NewInt(1000))
	assert.Nil(t, err)

	err = pam1.AddOutboundChannel(ctx, crypto.FIL, addr2, chAddr)
	assert.Nil(t, err)

	err = pam2.AddInboundChannel(ctx, crypto.FIL, addr1, chAddr)
	assert.Nil(t, err)

	resCh, resID, err := pam1.ReserveForSelf(ctx, crypto.FIL, addr2, big.NewInt(100), time.Now().Add(time.Second), time.Second)
	assert.Nil(t, err)

	for i := 0; i < 50; i++ {
		voucher, _, commit1, _, err := pam1.PayForSelf(ctx, crypto.FIL, addr2, resCh, resID, big.NewInt(1))
		assert.Nil(t, err)

		amt, commit2, _, _, err := pam2.Receive(ctx, crypto.FIL, voucher)
		assert.Nil(t, err)
		assert.Equal(t, big.NewInt(1), amt)

		commit2()
		commit1(time.Now())
	}

	time.Sleep(time.Second + time.Millisecond)
	// Reservation expired.
	_, _, _, _, err = pam1.PayForSelf(ctx, crypto.FIL, addr2, resCh, resID, big.NewInt(1))
	assert.NotNil(t, err)

	// Topup channel, for the next test.
	err = trans1.Topup(ctx, crypto.FIL, chAddr, big.NewInt(10000))
	assert.Nil(t, err)

	err = pam1.UpdateOutboundChannelBalance(ctx, crypto.FIL, addr2, chAddr)
	assert.Nil(t, err)

	// Make a reservation to test the tmp dir loading.
	offer := fcroffer.PayOffer{
		CurrencyID: crypto.FIL,
		SrcAddr:    addr2,
		DestAddr:   "test",
		PPP:        big.NewInt(10),
		Period:     big.NewInt(100),
		Amt:        big.NewInt(10),
		Expiration: time.Now().Add(time.Hour),
		Inactivity: time.Hour,
	}
	testResCh, testResID, err = pam1.ReserveForSelfWithOffer(ctx, offer)
	assert.Nil(t, err)

	_, offer2, _, revert, err := pam1.PayForSelf(ctx, crypto.FIL, addr2, testResCh, testResID, big.NewInt(1))
	assert.Nil(t, err)
	revert("")
	assert.NotNil(t, offer2)
}

func TestThreeNodes(t *testing.T) {
	ctx := context.Background()

	signer1, _, _, _, pam1, shutdown, err := getNode(ctx, 0)
	assert.Nil(t, err)
	defer shutdown()

	signer2, trans2, pservMgr2, reservMgr2, pam2, shutdown, err := getNode(ctx, 1)
	assert.Nil(t, err)
	defer shutdown()

	signer3, _, _, _, pam3, shutdown, err := getNode(ctx, 2)
	assert.Nil(t, err)
	defer shutdown()

	_, addr1, err := signer1.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, addr2, err := signer2.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, addr3, err := signer3.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	// Test if the tmp dir perserves after a testing.
	_, offer2, _, revert, err := pam1.PayForSelf(ctx, crypto.FIL, addr2, testResCh, testResID, big.NewInt(1))
	assert.Nil(t, err)
	revert("")
	assert.NotNil(t, offer2)

	// A payment channel from node 2 to node 3
	chAddr, err := trans2.Create(ctx, crypto.FIL, addr3, big.NewInt(10000))
	assert.Nil(t, err)

	err = pam2.AddOutboundChannel(ctx, crypto.FIL, addr3, chAddr)
	assert.Nil(t, err)

	err = pam3.AddInboundChannel(ctx, crypto.FIL, addr2, chAddr)
	assert.Nil(t, err)

	// pam1 pays to pam3 with pam2 as the proxy

	// Reserve without serving
	_, _, _, _, err = pam2.ReserveForOthersFinal(ctx, crypto.FIL, addr3, big.NewInt(1234), time.Now().Add(time.Second), time.Second, addr1)
	assert.NotNil(t, err)

	// The price here does not matter.
	err = pservMgr2.Serve(ctx, crypto.FIL, addr3, chAddr, big.NewInt(10), big.NewInt(100))
	assert.Nil(t, err)

	_, _, _, _, err = pam2.ReserveForOthersFinal(ctx, crypto.FIL, addr3, big.NewInt(1234), time.Now().Add(time.Second), time.Second, addr1)
	assert.NotNil(t, err)

	err = reservMgr2.SetDefaultPolicy(ctx, crypto.FIL, true, nil)
	assert.Nil(t, err)

	ppp, period, resChAddr, resID, err := pam2.ReserveForOthersFinal(ctx, crypto.FIL, addr3, big.NewInt(1234), time.Now().Add(time.Second), time.Second, addr1)
	assert.Nil(t, err)

	offer := fcroffer.PayOffer{
		CurrencyID: crypto.FIL,
		SrcAddr:    addr2,
		DestAddr:   addr3,
		PPP:        ppp,
		Period:     period,
		Amt:        big.NewInt(1234),
		Expiration: time.Now().Add(time.Second),
		Inactivity: time.Second,
		ResCh:      resChAddr,
		ResID:      resID,
	}

	resCh1, resID1, err := pam1.ReserveForSelfWithOffer(ctx, offer)
	assert.Nil(t, err)

	// First, make a payment after initial expiration.
	time.Sleep(time.Second + time.Millisecond)
	_, _, _, _, err = pam1.PayForSelf(ctx, crypto.FIL, addr2, resCh1, resID1, big.NewInt(1))
	assert.NotNil(t, err)

	ppp, period, resCh2, resID2, err := pam2.ReserveForOthersFinal(ctx, crypto.FIL, addr3, big.NewInt(1234), time.Now().Add(time.Second), time.Second, addr1)
	assert.Nil(t, err)

	offer = fcroffer.PayOffer{
		CurrencyID: crypto.FIL,
		SrcAddr:    addr2,
		DestAddr:   addr3,
		PPP:        ppp,
		Period:     period,
		Amt:        big.NewInt(1234),
		Expiration: time.Now().Add(time.Second),
		Inactivity: time.Second,
		ResCh:      resCh2,
		ResID:      resID2,
	}

	resCh1, resID1, err = pam1.ReserveForSelfWithOffer(ctx, offer)
	assert.Nil(t, err)

	for i := 0; i < 41; i++ {
		voucher1, _, commit1, _, err := pam1.PayForSelf(ctx, crypto.FIL, addr2, resCh1, resID1, big.NewInt(30))
		assert.Nil(t, err)

		gross, commit2, _, _, err := pam2.Receive(ctx, crypto.FIL, voucher1)
		assert.Nil(t, err)

		voucher2, _, commit3, _, err := pam2.PayForOthers(ctx, crypto.FIL, addr3, resCh2, resID2, gross, big.NewInt(30))
		assert.Nil(t, err)

		petty, commit4, _, _, err := pam3.Receive(ctx, crypto.FIL, voucher2)
		assert.Nil(t, err)
		assert.Equal(t, big.NewInt(30), petty)

		commit4()
		accessed := time.Now()
		commit3(accessed)
		commit2()
		commit1(accessed)
	}
	// Then, try to make a payment exceed maximum
	_, _, _, _, err = pam1.PayForSelf(ctx, crypto.FIL, addr2, resCh1, resID1, big.NewInt(30))
	assert.NotNil(t, err)

	// Finally, try to make a payment after inactivity time.
	time.Sleep(time.Second + time.Millisecond)
	_, _, _, _, err = pam1.PayForSelf(ctx, crypto.FIL, addr2, resCh1, resID1, big.NewInt(4))
	assert.NotNil(t, err)
}

func TestFourNodes(t *testing.T) {
	ctx := context.Background()

	signer1, _, _, _, pam1, shutdown, err := getNode(ctx, 0)
	assert.Nil(t, err)
	defer shutdown()

	signer2, _, _, _, pam2, shutdown, err := getNode(ctx, 1)
	assert.Nil(t, err)
	defer shutdown()

	signer3, trans3, pservMgr3, reservMgr3, pam3, shutdown, err := getNode(ctx, 2)
	assert.Nil(t, err)
	defer shutdown()

	signer4, _, _, _, pam4, shutdown, err := getNode(ctx, 3)
	assert.Nil(t, err)
	defer shutdown()

	_, addr1, err := signer1.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, addr2, err := signer2.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, addr3, err := signer3.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	_, addr4, err := signer4.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	// A payment channel from node 3 to node 4
	chAddr, err := trans3.Create(ctx, crypto.FIL, addr4, big.NewInt(10000))
	assert.Nil(t, err)

	err = pam3.AddOutboundChannel(ctx, crypto.FIL, addr4, chAddr)
	assert.Nil(t, err)

	err = pam4.AddInboundChannel(ctx, crypto.FIL, addr3, chAddr)
	assert.Nil(t, err)

	// pam1 pays to pam4 with pam2 and pam3 as the proxy
	// price here does not matter.
	err = pservMgr3.Serve(ctx, crypto.FIL, addr4, chAddr, big.NewInt(1), big.NewInt(100))
	assert.Nil(t, err)

	err = reservMgr3.SetDefaultPolicy(ctx, crypto.FIL, true, nil)
	assert.Nil(t, err)

	createdAt := time.Now()
	ppp, period, resCh3, resID3, err := pam3.ReserveForOthersFinal(ctx, crypto.FIL, addr4, big.NewInt(1234), createdAt.Add(time.Second), time.Second, addr2)
	assert.Nil(t, err)
	subOffer := fcroffer.PayOffer{
		CurrencyID: crypto.FIL,
		SrcAddr:    addr3,
		DestAddr:   addr4,
		PPP:        ppp,
		Period:     period,
		Amt:        big.NewInt(1234),
		Expiration: createdAt.Add(time.Second),
		Inactivity: time.Second,
		ResCh:      resCh3,
		ResID:      resID3,
	}

	ppp, period, resCh2, resID2, err := pam2.ReserveForOthersIntermediate(ctx, subOffer, addr1)
	assert.Nil(t, err)
	offer := fcroffer.PayOffer{
		CurrencyID: crypto.FIL,
		SrcAddr:    addr2,
		DestAddr:   addr4,
		PPP:        ppp,
		Period:     period,
		Amt:        big.NewInt(1234),
		Expiration: createdAt.Add(time.Second),
		Inactivity: time.Second,
		ResCh:      resCh2,
		ResID:      resID3,
	}

	resCh1, resID1, err := pam1.ReserveForSelfWithOffer(ctx, offer)
	assert.Nil(t, err)

	for i := 0; i < 41; i++ {
		voucher1, _, commit1, _, err := pam1.PayForSelf(ctx, crypto.FIL, addr2, resCh1, resID1, big.NewInt(30))
		assert.Nil(t, err)

		gross, commit2, _, _, err := pam2.Receive(ctx, crypto.FIL, voucher1)
		assert.Nil(t, err)

		voucher2, _, commit3, _, err := pam2.PayForOthers(ctx, crypto.FIL, addr3, resCh2, resID2, gross, big.NewInt(30))
		assert.Nil(t, err)

		gross, commit4, _, _, err := pam3.Receive(ctx, crypto.FIL, voucher2)
		assert.Nil(t, err)

		voucher3, _, commit5, _, err := pam3.PayForOthers(ctx, crypto.FIL, addr4, resCh3, resID3, gross, big.NewInt(30))
		assert.Nil(t, err)

		petty, commit6, _, _, err := pam4.Receive(ctx, crypto.FIL, voucher3)
		assert.Nil(t, err)
		assert.Equal(t, big.NewInt(30), petty)

		commit6()
		accessed := time.Now()
		commit5(accessed)
		commit4()
		commit3(accessed)
		commit2()
		commit1(accessed)
	}
	// Now try to make a payment with a network loss
	voucher1, _, _, revert1, err := pam1.PayForSelf(ctx, crypto.FIL, addr2, resCh1, resID1, big.NewInt(4))
	assert.Nil(t, err)

	gross, commit2, _, _, err := pam2.Receive(ctx, crypto.FIL, voucher1)
	assert.Nil(t, err)

	voucher2, _, commit3, _, err := pam2.PayForOthers(ctx, crypto.FIL, addr3, resCh2, resID2, gross, big.NewInt(4))
	assert.Nil(t, err)

	gross, commit4, _, _, err := pam3.Receive(ctx, crypto.FIL, voucher2)
	assert.Nil(t, err)

	voucher3, _, commit5, _, err := pam3.PayForOthers(ctx, crypto.FIL, addr4, resCh3, resID3, gross, big.NewInt(4))
	assert.Nil(t, err)

	petty, commit6, _, _, err := pam4.Receive(ctx, crypto.FIL, voucher3)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(4), petty)

	commit6()
	accessed := time.Now()
	commit5(accessed)
	commit4()
	commit3(accessed)
	commit2()
	revert1("")

	// Test network loss
	resCh1, resID1, err = pam1.ReserveForSelf(ctx, crypto.FIL, addr2, big.NewInt(100), time.Now().Add(time.Second), time.Second)
	assert.Nil(t, err)

	voucher1, _, _, revert1, err = pam1.PayForSelf(ctx, crypto.FIL, addr2, resCh1, resID1, big.NewInt(10))
	assert.Nil(t, err)

	// There is network loss.
	_, _, _, networkLossVoucher, err := pam2.Receive(ctx, crypto.FIL, voucher1)
	assert.NotNil(t, err)
	assert.NotNil(t, networkLossVoucher)
	revert1(networkLossVoucher)

	err = pam1.BearNetworkLoss(ctx, crypto.FIL, addr2, resCh1)
	assert.Nil(t, err)

	resCh1, resID1, err = pam1.ReserveForSelf(ctx, crypto.FIL, addr2, big.NewInt(100), time.Now().Add(time.Second), time.Second)
	assert.Nil(t, err)

	voucher1, _, commit1, _, err := pam1.PayForSelf(ctx, crypto.FIL, addr2, resCh1, resID1, big.NewInt(10))
	assert.Nil(t, err)

	amt, commit2, _, _, err := pam2.Receive(ctx, crypto.FIL, voucher1)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(10), amt)

	commit2()
	commit1(time.Now())

	// Test stop serving and retire channel
	err = pservMgr3.Stop(ctx, crypto.FIL, addr4, chAddr)
	assert.Nil(t, err)

	_, _, _, _, err = pam3.ReserveForOthersFinal(ctx, crypto.FIL, addr4, big.NewInt(1234), time.Now().Add(time.Second), time.Second, addr2)
	assert.NotNil(t, err)

	err = pam3.RetireInboundChannel(ctx, crypto.FIL, addr2, resCh2)
	assert.Nil(t, err)

	err = pam3.RetireOutboundChannel(ctx, crypto.FIL, addr4, chAddr)
	assert.Nil(t, err)

	time.Sleep(time.Second + time.Millisecond)
}

// getNode is used to create a payment manager node.
func getNode(ctx context.Context, index int) (*crypto.SignerImpl, *trans.TransactorImpl, *pservmgr.PaychServingManagerImpl, *reservmgr.ReservationManagerImpl, *PaymentManagerImpl, func(), error) {
	// New signer
	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "signer")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	prv := make([]byte, 32)
	rand.Read(prv)
	signer.SetKey(ctx, crypto.FIL, crypto.SECP256K1, prv)
	// New transactor
	confidence := uint64(0)
	transactor, err := trans.NewTransactorImpl(ctx, signer, trans.Opts{FilecoinEnabled: true, FilecoinAPI: testAPI, FilecoinConfidence: &confidence})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	// New paych serving manager.
	pservMgr, err := pservmgr.NewPaychServingManagerImpl(ctx, pservmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "pservmgr")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	// New reservation policy manager.
	reservMgr, err := reservmgr.NewReservationManagerImpl(ctx, reservmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "reservmgr")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	// New route store
	rs, err := routestore.NewRouteStoreImpl(ctx, signer, routestore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "rs"), CleanFreq: time.Second})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	// New sub store
	subs, err := substore.NewSubStoreImpl(ctx, substore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "subs")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	// New active in paych store.
	activeIn, err := paychstore.NewPaychStoreImpl(ctx, true, false, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "active-in")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	// New inactive in paych store.
	inactiveIn, err := paychstore.NewPaychStoreImpl(ctx, false, false, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "inactive-in")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	// New active out paych store.
	activeOut, err := paychstore.NewPaychStoreImpl(ctx, true, true, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "active-out")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	// New inactive out paych store.
	inactiveOut, err := paychstore.NewPaychStoreImpl(ctx, false, true, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "inactive-out")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	// New pam
	pam, err := NewPaymentManagerImpl(ctx, activeOut, activeIn, inactiveOut, inactiveIn, pservMgr, reservMgr, rs, subs, transactor, signer, Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "paymgr"), ResCleanFreq: time.Second, CacheSyncFreq: time.Second, PeerCleanFreq: time.Second})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
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
	return signer, transactor, pservMgr, reservMgr, pam, shutdown, nil
}
