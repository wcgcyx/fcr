package routeproto

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
	"github.com/wcgcyx/fcr/offermgr"
	"github.com/wcgcyx/fcr/paychmon"
	"github.com/wcgcyx/fcr/paychstore"
	"github.com/wcgcyx/fcr/paymgr"
	"github.com/wcgcyx/fcr/paynet/protos/paychproto"
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
	trans1, cProto1, rProto1, shutdown1, err := getNode(ctx, 0)
	assert.Nil(t, err)
	defer shutdown1()

	// Node 2
	trans2, cProto2, rProto2, shutdown2, err := getNode(ctx, 1)
	assert.Nil(t, err)
	defer shutdown2()

	// Node 3
	trans3, cProto3, rProto3, shutdown3, err := getNode(ctx, 2)
	assert.Nil(t, err)
	defer shutdown3()

	// Node 4
	_, _, rProto4, shutdown4, err := getNode(ctx, 3)
	assert.Nil(t, err)
	defer shutdown4()

	// Node 5
	trans5, cProto5, rProto5, shutdown5, err := getNode(ctx, 4)
	assert.Nil(t, err)
	defer shutdown5()

	// Node 6
	trans6, cProto6, rProto6, shutdown6, err := getNode(ctx, 5)
	assert.Nil(t, err)
	defer shutdown6()

	_, addr2, err := rProto2.signer.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)
	_, addr3, err := rProto3.signer.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)
	_, addr4, err := rProto4.signer.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)
	_, addr5, err := rProto5.signer.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	// Need to add peers.
	pi2 := peer.AddrInfo{
		ID:    rProto2.h.ID(),
		Addrs: rProto2.h.Addrs(),
	}
	pi3 := peer.AddrInfo{
		ID:    rProto3.h.ID(),
		Addrs: rProto3.h.Addrs(),
	}
	pi4 := peer.AddrInfo{
		ID:    rProto4.h.ID(),
		Addrs: rProto4.h.Addrs(),
	}
	pi5 := peer.AddrInfo{
		ID:    rProto5.h.ID(),
		Addrs: rProto5.h.Addrs(),
	}
	pi6 := peer.AddrInfo{
		ID:    rProto6.h.ID(),
		Addrs: rProto6.h.Addrs(),
	}
	err = rProto1.h.Connect(ctx, pi2)
	assert.Nil(t, err)
	err = rProto1.h.Connect(ctx, pi3)
	assert.Nil(t, err)
	err = rProto1.h.Connect(ctx, pi4)
	assert.Nil(t, err)
	err = rProto1.h.Connect(ctx, pi5)
	assert.Nil(t, err)
	err = rProto1.h.Connect(ctx, pi6)
	assert.Nil(t, err)
	time.Sleep(1 * time.Second)

	assert.Nil(t, rProto1.addrProto.Publish(ctx))
	assert.Nil(t, rProto2.addrProto.Publish(ctx))
	assert.Nil(t, rProto3.addrProto.Publish(ctx))
	assert.Nil(t, rProto4.addrProto.Publish(ctx))
	assert.Nil(t, rProto5.addrProto.Publish(ctx))

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
		chAddr1, err2 = trans3.Create(ctx, crypto.FIL, addr4, big.NewInt(1000))
		assert.Nil(t, err2)
		err = cProto3.Add(ctx, chAddr1, offer)
		assert.Nil(t, err)
		err = rProto3.rs.AddDirectLink(ctx, crypto.FIL, addr4)
		assert.Nil(t, err)
	}()

	go func() {
		defer wg.Done()
		offer, err := cProto2.QueryAdd(ctx, crypto.FIL, addr3)
		assert.Nil(t, err)
		var err2 error
		chAddr2, err2 = trans2.Create(ctx, crypto.FIL, addr3, big.NewInt(1000))
		assert.Nil(t, err2)
		err = cProto2.Add(ctx, chAddr2, offer)
		assert.Nil(t, err)
		err = rProto2.rs.AddDirectLink(ctx, crypto.FIL, addr3)
		assert.Nil(t, err)

		offer, err = cProto2.QueryAdd(ctx, crypto.FIL, addr5)
		assert.Nil(t, err)
		chAddr3, err2 = trans2.Create(ctx, crypto.FIL, addr5, big.NewInt(1000))
		assert.Nil(t, err2)
		err = cProto2.Add(ctx, chAddr3, offer)
		assert.Nil(t, err)
		err = rProto2.rs.AddDirectLink(ctx, crypto.FIL, addr5)
		assert.Nil(t, err)
	}()

	go func() {
		defer wg.Done()
		offer, err := cProto1.QueryAdd(ctx, crypto.FIL, addr2)
		assert.Nil(t, err)
		var err2 error
		chAddr4, err2 = trans1.Create(ctx, crypto.FIL, addr2, big.NewInt(1000))
		assert.Nil(t, err2)
		err = cProto1.Add(ctx, chAddr4, offer)
		assert.Nil(t, err)
		err = rProto1.rs.AddDirectLink(ctx, crypto.FIL, addr2)
		assert.Nil(t, err)
	}()

	go func() {
		defer wg.Done()
		offer, err := cProto5.QueryAdd(ctx, crypto.FIL, addr3)
		assert.Nil(t, err)
		var err2 error
		chAddr5, err2 = trans5.Create(ctx, crypto.FIL, addr3, big.NewInt(1000))
		assert.Nil(t, err2)
		err = cProto5.Add(ctx, chAddr5, offer)
		assert.Nil(t, err)
		err = rProto5.rs.AddDirectLink(ctx, crypto.FIL, addr3)
		assert.Nil(t, err)
	}()
	wg.Wait()

	// Test routes publish.
	routeChan, errChan := rProto1.rs.ListRoutesTo(ctx, crypto.FIL, addr3)
	routes := make([][]string, 0)
	for route := range routeChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(routes))

	err = rProto3.pservMgr.Serve(ctx, crypto.FIL, addr4, chAddr1, big.NewInt(10), big.NewInt(100))
	assert.Nil(t, err)
	err = rProto2.pservMgr.Serve(ctx, crypto.FIL, addr3, chAddr2, big.NewInt(10), big.NewInt(100))
	assert.Nil(t, err)
	err = rProto2.pservMgr.Serve(ctx, crypto.FIL, addr5, chAddr3, big.NewInt(10), big.NewInt(100))
	assert.Nil(t, err)
	err = rProto1.pservMgr.Serve(ctx, crypto.FIL, addr2, chAddr4, big.NewInt(10), big.NewInt(100))
	assert.Nil(t, err)
	err = rProto5.pservMgr.Serve(ctx, crypto.FIL, addr3, chAddr5, big.NewInt(10), big.NewInt(100))
	assert.Nil(t, err)

	time.Sleep(5 * time.Second)

	routeChan, errChan = rProto1.rs.ListRoutesTo(ctx, crypto.FIL, addr3)
	routes = make([][]string, 0)
	for route := range routeChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 2, len(routes))

	routeChan, errChan = rProto1.rs.ListRoutesTo(ctx, crypto.FIL, addr4)
	routes = make([][]string, 0)
	for route := range routeChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 2, len(routes))

	routeChan, errChan = rProto1.rs.ListRoutesTo(ctx, crypto.FIL, addr5)
	routes = make([][]string, 0)
	for route := range routeChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(routes))

	// Test publish hook
	rProto2.Shutdown()
	rProto2 = NewRouteProtocol(rProto2.h, rProto2.addrProto, rProto2.paychProto, rProto2.signer, rProto2.peerMgr, rProto2.pservMgr, rProto2.rs, rProto2.sbs, Opts{PublishFreq: time.Minute, RouteExpiry: time.Hour, PublishWait: time.Second})
	time.Sleep(1*time.Second + 500*time.Millisecond)

	offer, err := cProto6.QueryAdd(ctx, crypto.FIL, addr2)
	assert.Nil(t, err)
	chAddr, err := trans6.Create(ctx, crypto.FIL, addr2, big.NewInt(1000))
	assert.Nil(t, err)
	err = cProto6.Add(ctx, chAddr, offer)
	assert.Nil(t, err)
	err = rProto6.rs.AddDirectLink(ctx, crypto.FIL, addr2)
	assert.Nil(t, err)
	time.Sleep(1*time.Second + 500*time.Millisecond)

	routeChan, errChan = rProto6.rs.ListRoutesTo(ctx, crypto.FIL, addr3)
	routes = make([][]string, 0)
	for route := range routeChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 2, len(routes))
}

// getNode is used to create a payment manager node.
func getNode(ctx context.Context, index int) (*trans.TransactorImpl, *paychproto.PaychProtocol, *RouteProtocol, func(), error) {
	// New signer
	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "signer")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	prv := make([]byte, 32)
	rand.Read(prv)
	signer.SetKey(ctx, crypto.FIL, crypto.SECP256K1, prv)
	// New transactor
	confidence := uint64(0)
	transactor, err := trans.NewTransactorImpl(ctx, signer, trans.Opts{FilecoinEnabled: true, FilecoinAPI: testAPI, FilecoinConfidence: &confidence})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New paych serving manager.
	pservMgr, err := pservmgr.NewPaychServingManagerImpl(ctx, pservmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "pservmgr")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New reservation policy manager.
	reservMgr, err := reservmgr.NewReservationManagerImpl(ctx, reservmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "reservmgr")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New route store
	rs, err := routestore.NewRouteStoreImpl(ctx, signer, routestore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "rs"), CleanFreq: time.Second})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New sub store
	subs, err := substore.NewSubStoreImpl(ctx, substore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "subs")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New active in paych store.
	activeIn, err := paychstore.NewPaychStoreImpl(ctx, true, false, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "active-in")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New inactive in paych store.
	inactiveIn, err := paychstore.NewPaychStoreImpl(ctx, false, false, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "inactive-in")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New active out paych store.
	activeOut, err := paychstore.NewPaychStoreImpl(ctx, true, true, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "active-out")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New inactive out paych store.
	inactiveOut, err := paychstore.NewPaychStoreImpl(ctx, false, true, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "inactive-out")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New pam
	pam, err := paymgr.NewPaymentManagerImpl(ctx, activeOut, activeIn, inactiveOut, inactiveIn, pservMgr, reservMgr, rs, subs, transactor, signer, paymgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "paymgr"), ResCleanFreq: time.Second, CacheSyncFreq: time.Second, PeerCleanFreq: time.Second})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New offer manager
	offerMgr, err := offermgr.NewOfferManagerImpl(ctx, offermgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "offermgr")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New settle manager
	settleMgr, err := settlemgr.NewSettlementManagerImpl(ctx, settlemgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "settlemgr")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	err = settleMgr.SetDefaultPolicy(ctx, crypto.FIL, time.Hour)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New renew manager
	renewMgr, err := renewmgr.NewRenewManagerImpl(ctx, renewmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "renewmgr")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New peer manager
	peerMgr, err := peermgr.NewPeerManagerImpl(ctx, peermgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "peermgr")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New monitor
	monitor, err := paychmon.NewPaychMonitorImpl(ctx, pam, transactor, paychmon.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "paychmon"), CheckFreq: time.Second})
	if err != nil {
		return nil, nil, nil, nil, err
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
		return nil, nil, nil, nil, err
	}
	aProto := addrproto.NewAddrProtocol(h, dualDHT, signer, peerMgr, addrproto.Opts{PublishFreq: 1 * time.Second})
	cProto := paychproto.NewPaychProtocol(h, aProto, signer, peerMgr, offerMgr, settleMgr, renewMgr, pam, monitor, paychproto.Opts{RenewWindow: 50})
	rProto := NewRouteProtocol(h, aProto, cProto, signer, peerMgr, pservMgr, rs, subs, Opts{PublishFreq: 1 * time.Second, RouteExpiry: time.Hour})
	shutdown := func() {
		cProto.Shutdown()
		rProto.Shutdown()
		aProto.Shutdown()
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
	return transactor, cProto, rProto, shutdown, nil
}
