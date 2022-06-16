package paychproto

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

func TestTwoNodes(t *testing.T) {
	ctx := context.Background()
	// Node 1
	trans1, proto1, shutdown1, err := getNode(ctx, 0)
	assert.Nil(t, err)
	defer shutdown1()

	// Node 2
	_, proto2, shutdown2, err := getNode(ctx, 1)
	assert.Nil(t, err)
	defer shutdown2()

	_, addr1, err := proto1.signer.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)
	_, addr2, err := proto2.signer.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	// Test query add offer.
	_, err = proto1.QueryAdd(ctx, crypto.FIL, addr2)
	assert.NotNil(t, err)

	pi2 := peer.AddrInfo{
		ID:    proto2.h.ID(),
		Addrs: proto2.h.Addrs(),
	}
	err = proto1.h.Connect(ctx, pi2)
	assert.Nil(t, err)
	time.Sleep(1 * time.Second)

	assert.Nil(t, proto1.addrProto.Publish(ctx))
	assert.Nil(t, proto2.addrProto.Publish(ctx))

	_, err = proto1.QueryAdd(ctx, crypto.FIL, addr2)
	assert.NotNil(t, err)

	err = proto1.peerMgr.BlockPeer(ctx, crypto.FIL, addr2)
	assert.Nil(t, err)

	_, err = proto1.QueryAdd(ctx, crypto.FIL, addr2)
	assert.NotNil(t, err)

	err = proto1.peerMgr.UnblockPeer(ctx, crypto.FIL, addr2)
	assert.Nil(t, err)

	_, err = proto1.QueryAdd(ctx, crypto.FIL, addr2)
	assert.NotNil(t, err)

	err = proto2.peerMgr.BlockPeer(ctx, crypto.FIL, addr1)
	assert.Nil(t, err)

	_, err = proto1.QueryAdd(ctx, crypto.FIL, addr2)
	assert.NotNil(t, err)

	err = proto2.peerMgr.UnblockPeer(ctx, crypto.FIL, addr1)
	assert.Nil(t, err)

	_, err = proto1.QueryAdd(ctx, crypto.FIL, addr2)
	assert.NotNil(t, err)

	// Now add settlement manager
	err = proto2.settleMgr.SetDefaultPolicy(ctx, crypto.FIL, 0)
	assert.Nil(t, err)

	_, err = proto1.QueryAdd(ctx, crypto.FIL, addr2)
	assert.NotNil(t, err)

	err = proto2.settleMgr.SetDefaultPolicy(ctx, crypto.FIL, 5*time.Second)
	assert.Nil(t, err)

	offer, err := proto1.QueryAdd(ctx, crypto.FIL, addr2)
	assert.Nil(t, err)

	// Test add paych
	chAddr, err := trans1.Create(ctx, crypto.FIL, addr2, big.NewInt(100))
	assert.Nil(t, err)

	err = proto1.Add(ctx, chAddr, offer)
	assert.Nil(t, err)

	// Test renew
	_, settlement1, err := proto2.monitor.Check(ctx, false, crypto.FIL, chAddr)
	assert.Nil(t, err)

	err = proto1.Renew(ctx, crypto.FIL, addr2, chAddr)
	assert.NotNil(t, err)

	// Wait for window
	time.Sleep(3 * time.Second)

	err = proto1.Renew(ctx, crypto.FIL, addr2, chAddr)
	assert.NotNil(t, err)

	err = proto2.renewMgr.SetDefaultPolicy(ctx, crypto.FIL, 0)
	assert.Nil(t, err)

	err = proto1.Renew(ctx, crypto.FIL, addr2, chAddr)
	assert.NotNil(t, err)

	err = proto2.renewMgr.SetDefaultPolicy(ctx, crypto.FIL, time.Second)
	assert.Nil(t, err)

	err = proto1.Renew(ctx, crypto.FIL, addr2, chAddr)
	assert.NotNil(t, err)

	err = proto2.renewMgr.SetDefaultPolicy(ctx, crypto.FIL, time.Minute)
	assert.Nil(t, err)

	err = proto1.Renew(ctx, crypto.FIL, addr2, chAddr)
	assert.Nil(t, err)

	time.Sleep(1 * time.Second)

	_, settlement2, err := proto2.monitor.Check(ctx, false, crypto.FIL, chAddr)
	assert.Nil(t, err)
	assert.True(t, settlement2.After(settlement1))

	_, settlement3, err := proto1.monitor.Check(ctx, true, crypto.FIL, chAddr)
	assert.Nil(t, err)
	assert.True(t, settlement2.Equal(settlement3))
}

// getNode is used to create a payment manager node.
func getNode(ctx context.Context, index int) (*trans.TransactorImpl, *PaychProtocol, func(), error) {
	// New signer
	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "signer")})
	if err != nil {
		return nil, nil, nil, err
	}
	prv := make([]byte, 32)
	rand.Read(prv)
	signer.SetKey(ctx, crypto.FIL, crypto.SECP256K1, prv)
	// New transactor
	confidence := uint64(0)
	transactor, err := trans.NewTransactorImpl(ctx, signer, trans.Opts{FilecoinEnabled: true, FilecoinAPI: testAPI, FilecoinConfidence: &confidence})
	if err != nil {
		return nil, nil, nil, err
	}
	// New paych serving manager.
	pservMgr, err := pservmgr.NewPaychServingManagerImpl(ctx, pservmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "pservmgr")})
	if err != nil {
		return nil, nil, nil, err
	}
	// New reservation policy manager.
	reservMgr, err := reservmgr.NewReservationManagerImpl(ctx, reservmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "reservmgr")})
	if err != nil {
		return nil, nil, nil, err
	}
	// New route store
	rs, err := routestore.NewRouteStoreImpl(ctx, signer, routestore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "rs"), CleanFreq: time.Second})
	if err != nil {
		return nil, nil, nil, err
	}
	// New sub store
	subs, err := substore.NewSubStoreImpl(ctx, substore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "subs")})
	if err != nil {
		return nil, nil, nil, err
	}
	// New active in paych store.
	activeIn, err := paychstore.NewPaychStoreImpl(ctx, true, false, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "active-in")})
	if err != nil {
		return nil, nil, nil, err
	}
	// New inactive in paych store.
	inactiveIn, err := paychstore.NewPaychStoreImpl(ctx, false, false, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "inactive-in")})
	if err != nil {
		return nil, nil, nil, err
	}
	// New active out paych store.
	activeOut, err := paychstore.NewPaychStoreImpl(ctx, true, true, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "active-out")})
	if err != nil {
		return nil, nil, nil, err
	}
	// New inactive out paych store.
	inactiveOut, err := paychstore.NewPaychStoreImpl(ctx, false, true, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "inactive-out")})
	if err != nil {
		return nil, nil, nil, err
	}
	// New pam
	pam, err := paymgr.NewPaymentManagerImpl(ctx, activeOut, activeIn, inactiveOut, inactiveIn, pservMgr, reservMgr, rs, subs, transactor, signer, paymgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "paymgr"), ResCleanFreq: time.Second, CacheSyncFreq: time.Second, PeerCleanFreq: time.Second})
	if err != nil {
		return nil, nil, nil, err
	}
	// New offer manager
	offerMgr, err := offermgr.NewOfferManagerImpl(ctx, offermgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "offermgr")})
	if err != nil {
		return nil, nil, nil, err
	}
	// New settle manager
	settleMgr, err := settlemgr.NewSettlementManagerImpl(ctx, settlemgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "settlemgr")})
	if err != nil {
		return nil, nil, nil, err
	}
	// New renew manager
	renewMgr, err := renewmgr.NewRenewManagerImpl(ctx, renewmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "renewmgr")})
	if err != nil {
		return nil, nil, nil, err
	}
	// New peer manager
	peerMgr, err := peermgr.NewPeerManagerImpl(ctx, peermgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "peermgr")})
	if err != nil {
		return nil, nil, nil, err
	}
	// New monitor
	monitor, err := paychmon.NewPaychMonitorImpl(ctx, pam, transactor, paychmon.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "paychmon"), CheckFreq: time.Second})
	if err != nil {
		return nil, nil, nil, err
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
	aProto := addrproto.NewAddrProtocol(h, dualDHT, signer, peerMgr, addrproto.Opts{PublishFreq: 1 * time.Second})
	proto := NewPaychProtocol(h, aProto, signer, peerMgr, offerMgr, settleMgr, renewMgr, pam, monitor, Opts{RenewWindow: 50})
	shutdown := func() {
		proto.Shutdown()
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
	return transactor, proto, shutdown, nil
}
