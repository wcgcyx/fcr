package payproto

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
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	routing "github.com/libp2p/go-libp2p-routing"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/offermgr"
	"github.com/wcgcyx/fcr/paychmon"
	"github.com/wcgcyx/fcr/paychstore"
	"github.com/wcgcyx/fcr/paymgr"
	"github.com/wcgcyx/fcr/paynet/protos/offerproto"
	"github.com/wcgcyx/fcr/paynet/protos/paychproto"
	"github.com/wcgcyx/fcr/paynet/protos/routeproto"
	"github.com/wcgcyx/fcr/peermgr"
	"github.com/wcgcyx/fcr/peernet/protos/addrproto"
	"github.com/wcgcyx/fcr/pservmgr"
	"github.com/wcgcyx/fcr/renewmgr"
	"github.com/wcgcyx/fcr/reservmgr"
	"github.com/wcgcyx/fcr/routestore"
	"github.com/wcgcyx/fcr/settlemgr"
	"github.com/wcgcyx/fcr/substore"
	"github.com/wcgcyx/fcr/trans"
)

const (
	testDS = "./test-ds"
)

var testAPI = ""

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	server := trans.NewMockFil(time.Millisecond)
	defer server.Shutdown()
	testAPI = server.GetAPI()
	m.Run()
}

func TestFiveNodes(t *testing.T) {
	ctx := context.Background()
	// Node 1
	trans1, rs1, pservMgr1, cProto1, rProto1, proto1, shutdown1, err := getNode(ctx, 0)
	assert.Nil(t, err)
	defer shutdown1()

	// Node 2
	trans2, rs2, pservMgr2, cProto2, _, proto2, shutdown2, err := getNode(ctx, 1)
	assert.Nil(t, err)
	defer shutdown2()

	// Node 3
	trans3, rs3, pservMgr3, cProto3, _, proto3, shutdown3, err := getNode(ctx, 2)
	assert.Nil(t, err)
	defer shutdown3()

	// Node 4
	_, rs4, _, _, _, proto4, shutdown4, err := getNode(ctx, 3)
	assert.Nil(t, err)
	defer shutdown4()

	// Node 5
	trans5, rs5, pservMgr5, cProto5, _, proto5, shutdown5, err := getNode(ctx, 4)
	assert.Nil(t, err)
	defer shutdown5()

	_, addr1, err := proto1.signer.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)
	_, addr2, err := proto2.signer.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)
	_, addr3, err := proto3.signer.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)
	_, addr4, err := proto4.signer.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)
	_, addr5, err := proto5.signer.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	// Need to add peers.
	pi2 := peer.AddrInfo{
		ID:    proto2.h.ID(),
		Addrs: proto2.h.Addrs(),
	}
	pi3 := peer.AddrInfo{
		ID:    proto3.h.ID(),
		Addrs: proto3.h.Addrs(),
	}
	pi4 := peer.AddrInfo{
		ID:    proto4.h.ID(),
		Addrs: proto4.h.Addrs(),
	}
	pi5 := peer.AddrInfo{
		ID:    proto5.h.ID(),
		Addrs: proto5.h.Addrs(),
	}
	err = proto1.h.Connect(ctx, pi2)
	assert.Nil(t, err)
	err = proto1.h.Connect(ctx, pi3)
	assert.Nil(t, err)
	err = proto1.h.Connect(ctx, pi4)
	assert.Nil(t, err)
	err = proto1.h.Connect(ctx, pi5)
	assert.Nil(t, err)
	time.Sleep(1 * time.Second)

	assert.Nil(t, proto1.addrProto.Publish(ctx))
	assert.Nil(t, proto2.addrProto.Publish(ctx))
	assert.Nil(t, proto3.addrProto.Publish(ctx))
	assert.Nil(t, proto4.addrProto.Publish(ctx))
	assert.Nil(t, proto5.addrProto.Publish(ctx))

	// Testing scenario, two routes overall:
	// Node 1 -> Node 2 -> Node 3 -> Node 4
	// Node 2 -> Node 5 -> Node 3
	wg := sync.WaitGroup{}
	wg.Add(4)
	var chAddr1, chAddr2, chAddr3, chAddr4, chAddr5 string
	go func() {
		defer wg.Done()
		offer, err := cProto3.QueryAdd(ctx, crypto.FIL, addr4)
		assert.Nil(t, err)
		var err2 error
		chAddr1, err2 = trans3.Create(ctx, crypto.FIL, addr4, big.NewInt(100000))
		assert.Nil(t, err2)
		err = cProto3.Add(ctx, chAddr1, offer)
		assert.Nil(t, err)
		err = rs3.AddDirectLink(ctx, crypto.FIL, addr4)
		assert.Nil(t, err)
	}()

	go func() {
		defer wg.Done()
		offer, err := cProto2.QueryAdd(ctx, crypto.FIL, addr3)
		assert.Nil(t, err)
		var err2 error
		chAddr2, err2 = trans2.Create(ctx, crypto.FIL, addr3, big.NewInt(100000))
		assert.Nil(t, err2)
		err = cProto2.Add(ctx, chAddr2, offer)
		assert.Nil(t, err)
		err = rs2.AddDirectLink(ctx, crypto.FIL, addr3)
		assert.Nil(t, err)

		offer, err = cProto2.QueryAdd(ctx, crypto.FIL, addr5)
		assert.Nil(t, err)
		chAddr3, err2 = trans2.Create(ctx, crypto.FIL, addr5, big.NewInt(100000))
		assert.Nil(t, err2)
		err = cProto2.Add(ctx, chAddr3, offer)
		assert.Nil(t, err)
		err = rs4.AddDirectLink(ctx, crypto.FIL, addr5)
		assert.Nil(t, err)
	}()

	go func() {
		defer wg.Done()
		offer, err := cProto1.QueryAdd(ctx, crypto.FIL, addr2)
		assert.Nil(t, err)
		var err2 error
		chAddr4, err2 = trans1.Create(ctx, crypto.FIL, addr2, big.NewInt(100000))
		assert.Nil(t, err2)
		err = cProto1.Add(ctx, chAddr4, offer)
		assert.Nil(t, err)
		err = rs5.AddDirectLink(ctx, crypto.FIL, addr2)
		assert.Nil(t, err)
	}()

	go func() {
		defer wg.Done()
		offer, err := cProto5.QueryAdd(ctx, crypto.FIL, addr3)
		assert.Nil(t, err)
		var err2 error
		chAddr5, err2 = trans5.Create(ctx, crypto.FIL, addr3, big.NewInt(100000))
		assert.Nil(t, err2)
		err = cProto5.Add(ctx, chAddr5, offer)
		assert.Nil(t, err)
		err = rs5.AddDirectLink(ctx, crypto.FIL, addr3)
		assert.Nil(t, err)
	}()
	wg.Wait()

	// Serve channels
	err = pservMgr3.Serve(ctx, crypto.FIL, addr4, chAddr1, big.NewInt(10), big.NewInt(100))
	assert.Nil(t, err)
	err = pservMgr2.Serve(ctx, crypto.FIL, addr3, chAddr2, big.NewInt(10), big.NewInt(100))
	assert.Nil(t, err)
	err = pservMgr2.Serve(ctx, crypto.FIL, addr5, chAddr3, big.NewInt(10), big.NewInt(100))
	assert.Nil(t, err)
	err = pservMgr1.Serve(ctx, crypto.FIL, addr2, chAddr4, big.NewInt(10), big.NewInt(100))
	assert.Nil(t, err)
	err = pservMgr5.Serve(ctx, crypto.FIL, addr3, chAddr5, big.NewInt(10), big.NewInt(100))
	assert.Nil(t, err)

	// Wait for route publish.
	time.Sleep(5 * time.Second)

	var resCh0, resCh1, resCh2, resCh3, resCh4, resCh5 string
	var resID0, resID1, resID2, resID3, resID4, resID5 uint64
	var offer1, offer2, offer3, offer4, offer5 fcroffer.PayOffer

	wg.Add(4)
	// Payment node 1 -> node 2
	go func() {
		defer wg.Done()
		resCh0, resID0, err = proto1.payMgr.ReserveForSelf(ctx, crypto.FIL, addr2, big.NewInt(1234), time.Now().Add(time.Hour), time.Hour)
		assert.Nil(t, err)
	}()

	// Query offer, node 1 -> node 3
	go func() {
		defer wg.Done()
		routeOut, errOut := rs1.ListRoutesTo(ctx, crypto.FIL, addr3)
		offers := make([]fcroffer.PayOffer, 0)
		for route := range routeOut {
			offer, err := rProto1.QueryOffer(ctx, crypto.FIL, route, big.NewInt(1234))
			assert.Nil(t, err)
			offers = append(offers, offer)
		}
		err = <-errOut
		assert.Nil(t, err)
		assert.Equal(t, 2, len(offers))
		offer1 = offers[0]
		resCh1, resID1, err = proto1.payMgr.ReserveForSelfWithOffer(ctx, offer1)
		assert.Nil(t, err)
		offer2 = offers[1]
		resCh2, resID2, err = proto1.payMgr.ReserveForSelfWithOffer(ctx, offer2)
		assert.Nil(t, err)
	}()

	// Query offer, node 1 -> node 4
	go func() {
		defer wg.Done()
		routeOut, errOut := rs1.ListRoutesTo(ctx, crypto.FIL, addr4)
		offers := make([]fcroffer.PayOffer, 0)
		for route := range routeOut {
			offer, err := rProto1.QueryOffer(ctx, crypto.FIL, route, big.NewInt(1234))
			assert.Nil(t, err)
			offers = append(offers, offer)
		}
		err = <-errOut
		assert.Nil(t, err)
		assert.Equal(t, 2, len(offers))
		offer3 = offers[0]
		resCh3, resID3, err = proto1.payMgr.ReserveForSelfWithOffer(ctx, offer3)
		assert.Nil(t, err)
		offer4 = offers[1]
		resCh4, resID4, err = proto1.payMgr.ReserveForSelfWithOffer(ctx, offer4)
		assert.Nil(t, err)
	}()

	// Query offer, node 1 -> node 5
	go func() {
		defer wg.Done()
		routeOut, errOut := rs1.ListRoutesTo(ctx, crypto.FIL, addr5)
		offers := make([]fcroffer.PayOffer, 0)
		for route := range routeOut {
			offer, err := rProto1.QueryOffer(ctx, crypto.FIL, route, big.NewInt(1234))
			assert.Nil(t, err)
			offers = append(offers, offer)
		}
		err = <-errOut
		assert.Nil(t, err)
		assert.Equal(t, 1, len(offers))
		offer5 = offers[0]
		resCh5, resID5, err = proto1.payMgr.ReserveForSelfWithOffer(ctx, offer5)
		assert.Nil(t, err)
	}()
	wg.Wait()

	wg.Add(6)
	// Payment
	go func() {
		defer wg.Done()
		var err0 error
		for i := 0; i < 100; i++ {
			time.Sleep(time.Millisecond * 10)
			err0 = proto1.PayForSelf(ctx, crypto.FIL, addr2, resCh0, resID0, big.NewInt(10))
			assert.Nil(t, err0)
		}
		err0 = proto1.PayForSelf(ctx, crypto.FIL, addr2, resCh0, resID0, big.NewInt(1))
		assert.Nil(t, err0)
	}()

	go func() {
		defer wg.Done()
		var err1 error
		for i := 0; i < 50; i++ {
			time.Sleep(time.Millisecond * 20)
			err1 = proto1.PayForSelfWithOffer(ctx, offer1, resCh1, resID1, big.NewInt(20))
			assert.Nil(t, err1)
		}
		err1 = proto1.PayForSelfWithOffer(ctx, offer1, resCh1, resID1, big.NewInt(2))
		assert.Nil(t, err1)
	}()

	go func() {
		defer wg.Done()
		var err2 error
		for i := 0; i < 25; i++ {
			time.Sleep(time.Millisecond * 40)
			err2 = proto1.PayForSelfWithOffer(ctx, offer2, resCh2, resID2, big.NewInt(40))
			assert.Nil(t, err2)
		}
		err2 = proto1.PayForSelfWithOffer(ctx, offer2, resCh2, resID2, big.NewInt(3))
		assert.Nil(t, err2)
	}()

	go func() {
		defer wg.Done()
		var err3 error
		for i := 0; i < 20; i++ {
			time.Sleep(time.Millisecond * 50)
			err3 = proto1.PayForSelfWithOffer(ctx, offer3, resCh3, resID3, big.NewInt(50))
			assert.Nil(t, err3)
		}
		err3 = proto1.PayForSelfWithOffer(ctx, offer3, resCh3, resID3, big.NewInt(4))
		assert.Nil(t, err3)
	}()

	go func() {
		defer wg.Done()
		var err4 error
		for i := 0; i < 10; i++ {
			time.Sleep(time.Millisecond * 100)
			err4 = proto1.PayForSelfWithOffer(ctx, offer4, resCh4, resID4, big.NewInt(100))
			assert.Nil(t, err4)
		}
		err4 = proto1.PayForSelfWithOffer(ctx, offer4, resCh4, resID4, big.NewInt(5))
		assert.Nil(t, err4)
	}()

	go func() {
		defer wg.Done()
		var err5 error
		for i := 0; i < 5; i++ {
			time.Sleep(time.Millisecond * 200)
			err5 = proto1.PayForSelfWithOffer(ctx, offer5, resCh5, resID5, big.NewInt(200))
			assert.Nil(t, err5)
		}
		err5 = proto1.PayForSelfWithOffer(ctx, offer5, resCh5, resID5, big.NewInt(6))
		assert.Nil(t, err5)
	}()

	wg.Wait()

	received, err := proto2.Receive(ctx, crypto.FIL, addr1)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(1001), received)

	received, err = proto3.Receive(ctx, crypto.FIL, addr1)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(2005), received)

	received, err = proto4.Receive(ctx, crypto.FIL, addr1)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(2009), received)

	received, err = proto5.Receive(ctx, crypto.FIL, addr1)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(1006), received)

	time.Sleep(time.Second)
}

// getNode is used to create a payment manager node.
func getNode(ctx context.Context, index int) (*trans.TransactorImpl, routestore.RouteStore, pservmgr.PaychServingManager, *paychproto.PaychProtocol, *offerproto.OfferProtocol, *PayProtocol, func(), error) {
	// New signer
	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "signer")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	prv := make([]byte, 32)
	rand.Read(prv)
	signer.SetKey(ctx, crypto.FIL, crypto.SECP256K1, prv)
	// New transactor
	confidence := uint64(0)
	transactor, err := trans.NewTransactorImpl(ctx, signer, trans.Opts{FilecoinEnabled: true, FilecoinAPI: testAPI, FilecoinConfidence: &confidence})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	// New paych serving manager.
	pservMgr, err := pservmgr.NewPaychServingManagerImpl(ctx, pservmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "pservmgr")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	// New reservation policy manager.
	reservMgr, err := reservmgr.NewReservationManagerImpl(ctx, reservmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "reservmgr")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	err = reservMgr.SetDefaultPolicy(ctx, crypto.FIL, true, nil)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	// New route store
	rs, err := routestore.NewRouteStoreImpl(ctx, signer, routestore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "rs"), CleanFreq: time.Second})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	// New sub store
	subs, err := substore.NewSubStoreImpl(ctx, substore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "subs")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	// New active in paych store.
	activeIn, err := paychstore.NewPaychStoreImpl(ctx, true, false, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "active-in")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	// New inactive in paych store.
	inactiveIn, err := paychstore.NewPaychStoreImpl(ctx, false, false, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "inactive-in")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	// New active out paych store.
	activeOut, err := paychstore.NewPaychStoreImpl(ctx, true, true, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "active-out")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	// New inactive out paych store.
	inactiveOut, err := paychstore.NewPaychStoreImpl(ctx, false, true, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "inactive-out")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	// New pam
	pam, err := paymgr.NewPaymentManagerImpl(ctx, activeOut, activeIn, inactiveOut, inactiveIn, pservMgr, reservMgr, rs, subs, transactor, signer, paymgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "paymgr"), ResCleanFreq: time.Second, CacheSyncFreq: time.Second, PeerCleanFreq: time.Second})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	// New offer manager
	offerMgr, err := offermgr.NewOfferManagerImpl(ctx, offermgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "offermgr")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	// New settle manager
	settleMgr, err := settlemgr.NewSettlementManagerImpl(ctx, settlemgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "settlemgr")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	err = settleMgr.SetDefaultPolicy(ctx, crypto.FIL, time.Hour)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	// New renew manager
	renewMgr, err := renewmgr.NewRenewManagerImpl(ctx, renewmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "renewmgr")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	// New peer manager
	peerMgr, err := peermgr.NewPeerManagerImpl(ctx, peermgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "peermgr")})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	// New monitor
	monitor, err := paychmon.NewPaychMonitorImpl(ctx, pam, transactor, paychmon.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "paychmon"), CheckFreq: time.Second})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	// New host
	var dualDHT *dual.DHT
	h, err := libp2p.New(
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			dualDHT, err = dual.New(ctx, h, dual.DHTOption(dht.ProtocolPrefix("/fc-retrieval")))
			return dualDHT, err
		}),
	)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	aProto := addrproto.NewAddrProtocol(h, dualDHT, signer, peerMgr, addrproto.Opts{PublishFreq: 1 * time.Second})
	pProto, err := NewPayProtocol(ctx, h, aProto, signer, peerMgr, pam, Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "payproto"), CleanFreq: time.Second})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	cProto := paychproto.NewPaychProtocol(h, aProto, signer, peerMgr, offerMgr, settleMgr, renewMgr, pam, monitor, paychproto.Opts{RenewWindow: 50})
	rProto := routeproto.NewRouteProtocol(h, aProto, cProto, signer, peerMgr, pservMgr, rs, subs, routeproto.Opts{PublishFreq: 1 * time.Second, RouteExpiry: time.Hour})
	oProto := offerproto.NewOfferProtocol(h, aProto, signer, peerMgr, offerMgr, pservMgr, rs, pam, offerproto.Opts{})
	shutdown := func() {
		pProto.Shutdown()
		oProto.Shutdown()
		cProto.Shutdown()
		rProto.Shutdown()
		monitor.Shutdown()
		peerMgr.Shutdown()
		renewMgr.Shutdown()
		settleMgr.Shutdown()
		offerMgr.Shutdown()
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
	return transactor, rs, pservMgr, cProto, oProto, pProto, shutdown, nil
}
