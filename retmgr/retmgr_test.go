package retmgr

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
	"io/ioutil"
	rand2 "math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	routing "github.com/libp2p/go-libp2p-routing"
	"github.com/stretchr/testify/assert"
	cofferproto "github.com/wcgcyx/fcr/cidnet/protos/offerproto"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/cservmgr"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/mpstore"
	"github.com/wcgcyx/fcr/offermgr"
	"github.com/wcgcyx/fcr/paychmon"
	"github.com/wcgcyx/fcr/paychstore"
	"github.com/wcgcyx/fcr/paymgr"
	pofferproto "github.com/wcgcyx/fcr/paynet/protos/offerproto"
	"github.com/wcgcyx/fcr/paynet/protos/paychproto"
	"github.com/wcgcyx/fcr/paynet/protos/payproto"
	"github.com/wcgcyx/fcr/paynet/protos/routeproto"
	"github.com/wcgcyx/fcr/peermgr"
	"github.com/wcgcyx/fcr/peernet/protos/addrproto"
	"github.com/wcgcyx/fcr/piecemgr"
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

type testNode struct {
	// hosted content
	content cid.Cid
	// Components
	h           host.Host
	signer      *crypto.SignerImpl
	transactor  *trans.TransactorImpl
	pieceMgr    *piecemgr.PieceManagerImpl
	pservMgr    *pservmgr.PaychServingManagerImpl
	reservMgr   *reservmgr.ReservationManagerImpl
	rs          *routestore.RouteStoreImpl
	subs        *substore.SubStoreImpl
	activeIn    *paychstore.PaychStoreImpl
	inactiveIn  *paychstore.PaychStoreImpl
	activeOut   *paychstore.PaychStoreImpl
	inactiveOut *paychstore.PaychStoreImpl
	payMgr      *paymgr.PaymentManagerImpl
	offerMgr    *offermgr.OfferManagerImpl
	settleMgr   *settlemgr.SettlementManagerImpl
	renewMgr    *renewmgr.RenewManagerImpl
	peerMgr     *peermgr.PeerManagerImpl
	monitor     *paychmon.PaychMonitorImpl
	// Protocols
	addrProto   *addrproto.AddrProtocol
	paychProto  *paychproto.PaychProtocol
	pofferProto *pofferproto.OfferProtocol
	routeProto  *routeproto.RouteProtocol
	cofferProto *cofferproto.OfferProtocol
	payProto    *payproto.PayProtocol
	// Retrieval manager
	retMgr *RetrievalManager
}

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	server := trans.NewMockFil(time.Millisecond)
	defer server.Shutdown()
	testAPI = server.GetAPI()
	m.Run()
}

func TestTenNodes(t *testing.T) {
	ctx := context.Background()

	// Node 0
	node0, shutdown0, err := getNode(ctx, 0)
	assert.Nil(t, err)
	defer shutdown0()

	// Node 1
	node1, shutdown1, err := getNode(ctx, 1)
	assert.Nil(t, err)
	defer shutdown1()

	// Node 2
	node2, shutdown2, err := getNode(ctx, 2)
	assert.Nil(t, err)
	defer shutdown2()

	// Node 3
	node3, shutdown3, err := getNode(ctx, 3)
	assert.Nil(t, err)
	defer shutdown3()

	// Node 4
	node4, shutdown4, err := getNode(ctx, 4)
	assert.Nil(t, err)
	defer shutdown4()

	// Node 5
	node5, shutdown5, err := getNode(ctx, 5)
	assert.Nil(t, err)
	defer shutdown5()

	// Node 6
	node6, shutdown6, err := getNode(ctx, 6)
	assert.Nil(t, err)
	defer shutdown6()

	// Node 7
	node7, shutdown7, err := getNode(ctx, 7)
	assert.Nil(t, err)
	defer shutdown7()

	// Node 8
	node8, shutdown8, err := getNode(ctx, 8)
	assert.Nil(t, err)
	defer shutdown8()

	// Node 9
	node9, shutdown9, err := getNode(ctx, 9)
	assert.Nil(t, err)
	defer shutdown9()

	// Set-up testing scenario
	pi0 := peer.AddrInfo{
		ID:    node0.retMgr.h.ID(),
		Addrs: node0.retMgr.h.Addrs(),
	}

	err = node1.retMgr.h.Connect(ctx, pi0)
	assert.Nil(t, err)
	err = node2.retMgr.h.Connect(ctx, pi0)
	assert.Nil(t, err)
	err = node3.retMgr.h.Connect(ctx, pi0)
	assert.Nil(t, err)
	err = node4.retMgr.h.Connect(ctx, pi0)
	assert.Nil(t, err)
	err = node5.retMgr.h.Connect(ctx, pi0)
	assert.Nil(t, err)
	err = node6.retMgr.h.Connect(ctx, pi0)
	assert.Nil(t, err)
	err = node7.retMgr.h.Connect(ctx, pi0)
	assert.Nil(t, err)
	err = node8.retMgr.h.Connect(ctx, pi0)
	assert.Nil(t, err)
	err = node9.retMgr.h.Connect(ctx, pi0)
	assert.Nil(t, err)
	time.Sleep(1 * time.Second)

	assert.Nil(t, node1.addrProto.Publish(ctx))
	assert.Nil(t, node1.addrProto.Publish(ctx))
	assert.Nil(t, node1.addrProto.Publish(ctx))
	assert.Nil(t, node1.addrProto.Publish(ctx))
	assert.Nil(t, node1.addrProto.Publish(ctx))
	assert.Nil(t, node1.addrProto.Publish(ctx))
	assert.Nil(t, node1.addrProto.Publish(ctx))
	assert.Nil(t, node1.addrProto.Publish(ctx))
	assert.Nil(t, node1.addrProto.Publish(ctx))
	assert.Nil(t, node1.addrProto.Publish(ctx))

	// Create payment channels
	bal, err := big.FromString("1000000000000000000")
	assert.Nil(t, err)

	var wg sync.WaitGroup
	// Node 0 -> Node 1
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, createPaych(ctx, node0, node1, bal)) }()
	}()
	// Node 1 -> (Node 2, Node4, Node 6)
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, createPaych(ctx, node1, node2, bal)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, createPaych(ctx, node1, node4, bal)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, createPaych(ctx, node1, node6, bal)) }()
	}()
	// Node 2 -> Node 3
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, createPaych(ctx, node2, node3, bal)) }()
	}()
	// Node 3 -> Node 0
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, createPaych(ctx, node3, node0, bal)) }()
	}()
	// Node 4 -> (Node 5, Node 7)
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, createPaych(ctx, node4, node5, bal)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, createPaych(ctx, node4, node7, bal)) }()
	}()
	// Node 5 -> Node 2
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, createPaych(ctx, node5, node2, bal)) }()
	}()
	// Node 6 -> Node 7
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, createPaych(ctx, node6, node7, bal)) }()
	}()
	// Node 7 -> Node 8
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, createPaych(ctx, node7, node8, bal)) }()
	}()
	// Node 8 -> (Node 5, Node 9)
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, createPaych(ctx, node8, node5, bal)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, createPaych(ctx, node8, node9, bal)) }()
	}()
	// Node 9 -> Node 8
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, createPaych(ctx, node9, node8, bal)) }()
	}()
	wg.Wait()

	time.Sleep(5 * time.Second)
	p0 := fmt.Sprintf("%v/%v", testDS, "node0")
	p1 := fmt.Sprintf("%v/%v", testDS, "node1")
	p2 := fmt.Sprintf("%v/%v", testDS, "node2")
	p3 := fmt.Sprintf("%v/%v", testDS, "node3")
	p4 := fmt.Sprintf("%v/%v", testDS, "node4")
	p5 := fmt.Sprintf("%v/%v", testDS, "node5")
	p6 := fmt.Sprintf("%v/%v", testDS, "node6")
	p7 := fmt.Sprintf("%v/%v", testDS, "node7")
	p8 := fmt.Sprintf("%v/%v", testDS, "node8")
	p9 := fmt.Sprintf("%v/%v", testDS, "node9")
	os.Mkdir(p0, os.ModePerm)
	os.Mkdir(p1, os.ModePerm)
	os.Mkdir(p2, os.ModePerm)
	os.Mkdir(p3, os.ModePerm)
	os.Mkdir(p4, os.ModePerm)
	os.Mkdir(p5, os.ModePerm)
	os.Mkdir(p6, os.ModePerm)
	os.Mkdir(p7, os.ModePerm)
	os.Mkdir(p8, os.ModePerm)
	os.Mkdir(p9, os.ModePerm)

	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node0, node1, p0, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node0, node2, p0, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node0, node3, p0, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node0, node4, p0, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node0, node5, p0, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node0, node6, p0, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node0, node7, p0, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node0, node8, p0, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node0, node9, p0, 1)) }()
	}()
	wg.Wait()

	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node1, node0, p1, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node1, node2, p1, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node1, node3, p1, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node1, node4, p1, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node1, node5, p1, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node1, node6, p1, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node1, node7, p1, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node1, node8, p1, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node1, node9, p1, 1)) }()
	}()
	wg.Wait()

	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node2, node0, p2, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node2, node1, p2, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node2, node3, p2, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node2, node4, p2, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node2, node5, p2, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node2, node6, p2, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node2, node7, p2, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node2, node8, p2, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node2, node9, p2, 1)) }()
	}()
	wg.Wait()

	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node3, node0, p3, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node3, node1, p3, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node3, node2, p3, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node3, node4, p3, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node3, node5, p3, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node3, node6, p3, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node3, node7, p3, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node3, node8, p3, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node3, node9, p3, 1)) }()
	}()
	wg.Wait()

	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node4, node0, p4, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node4, node1, p4, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node4, node2, p4, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node4, node3, p4, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node4, node5, p4, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node4, node6, p4, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node4, node7, p4, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node4, node8, p4, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node4, node9, p4, 1)) }()
	}()
	wg.Wait()

	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node5, node0, p5, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node5, node1, p5, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node5, node2, p5, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node5, node3, p5, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node5, node4, p5, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node5, node6, p5, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node5, node7, p5, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node5, node8, p5, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node5, node9, p5, 1)) }()
	}()
	wg.Wait()

	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node6, node0, p6, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node6, node1, p6, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node6, node2, p6, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node6, node3, p6, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node6, node4, p6, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node6, node5, p6, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node6, node7, p6, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node6, node8, p6, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node6, node9, p6, 1)) }()
	}()
	wg.Wait()

	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node7, node0, p7, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node7, node1, p7, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node7, node2, p7, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node7, node3, p7, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node7, node4, p7, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node7, node5, p7, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node7, node6, p7, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node7, node8, p7, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node7, node9, p7, 1)) }()
	}()
	wg.Wait()

	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node8, node0, p8, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node8, node1, p8, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node8, node2, p8, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node8, node3, p8, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node8, node4, p8, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node8, node5, p8, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node8, node6, p8, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node8, node7, p8, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node8, node9, p8, 1)) }()
	}()
	wg.Wait()

	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node9, node0, p9, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node9, node1, p9, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node9, node2, p9, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node9, node3, p9, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node9, node4, p9, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node9, node5, p9, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node9, node6, p9, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node9, node7, p9, 1)) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); assert.Nil(t, testRetrieve(ctx, node9, node8, p9, 1)) }()
	}()
	wg.Wait()

	files, err := ioutil.ReadDir(p0)
	assert.Nil(t, err)
	assert.Equal(t, 9, len(files))
	files, err = ioutil.ReadDir(p1)
	assert.Nil(t, err)
	assert.Equal(t, 9, len(files))
	files, err = ioutil.ReadDir(p2)
	assert.Nil(t, err)
	assert.Equal(t, 9, len(files))
	files, err = ioutil.ReadDir(p3)
	assert.Nil(t, err)
	assert.Equal(t, 9, len(files))
	files, err = ioutil.ReadDir(p4)
	assert.Nil(t, err)
	assert.Equal(t, 9, len(files))
	files, err = ioutil.ReadDir(p5)
	assert.Nil(t, err)
	assert.Equal(t, 9, len(files))
	files, err = ioutil.ReadDir(p6)
	assert.Nil(t, err)
	assert.Equal(t, 9, len(files))
	files, err = ioutil.ReadDir(p7)
	assert.Nil(t, err)
	assert.Equal(t, 9, len(files))
	files, err = ioutil.ReadDir(p8)
	assert.Nil(t, err)
	assert.Equal(t, 9, len(files))
	files, err = ioutil.ReadDir(p9)
	assert.Nil(t, err)
	assert.Equal(t, 9, len(files))

	// Test retrievcal cache
	exists, err := node1.retMgr.RetrieveFromCache(ctx, node9.content, testDS)
	assert.Nil(t, err)
	assert.True(t, exists)

	size, err := node1.retMgr.GetRetrievalCacheSize(ctx)
	assert.Nil(t, err)
	assert.NotEmpty(t, size)

	err = node1.retMgr.CleanRetrievalCache(ctx)
	assert.Nil(t, err)

	size, err = node1.retMgr.GetRetrievalCacheSize(ctx)
	assert.Nil(t, err)
	assert.Empty(t, size)

	exists, err = node1.retMgr.RetrieveFromCache(ctx, node9.content, testDS)
	assert.Nil(t, err)
	assert.False(t, exists)

	// Test deadlock
	t.Log("Test deadlock...")
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); testRetrieve(ctx, node0, node3, p0, 5) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); testRetrieve(ctx, node1, node0, p1, 5) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); testRetrieve(ctx, node2, node1, p2, 5) }()
	}()
	wg.Wait()

	// Test restart.
	t.Log("Test restart...")
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); testRetrieve(ctx, node0, node1, p0, 5) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); testRetrieve(ctx, node2, node1, p2, 5) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); testRetrieve(ctx, node3, node1, p3, 5) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); testRetrieve(ctx, node4, node1, p4, 5) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); testRetrieve(ctx, node5, node1, p5, 5) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); testRetrieve(ctx, node6, node1, p6, 5) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); testRetrieve(ctx, node7, node1, p7, 5) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); testRetrieve(ctx, node8, node1, p8, 5) }()
	}()
	func() {
		wg.Add(1)
		go func() { defer wg.Done(); testRetrieve(ctx, node9, node1, p9, 5) }()
	}()
	func() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			node1.retMgr.Shutdown()
			assert.Nil(t, err)
			time.Sleep(1 * time.Second)
			node1.retMgr, err = NewRetrievalManager(ctx, node1.h, node1.addrProto, node1.payProto, node1.signer, node1.peerMgr, node1.pieceMgr, Opts{IOTimeout: 5 * time.Second, OpTimeout: 5 * time.Second, CachePath: fmt.Sprintf("%v/%v-%v", testDS, 1, "retcache"), TempPath: fmt.Sprintf("%v/%v-%v", testDS, 1, "rettempdir")})
			assert.Nil(t, err)
			err = node1.retMgr.CleanIncomingProcesses(ctx)
			assert.Nil(t, err)
			err = node1.retMgr.CleanOutgoingProcesses(ctx)
			assert.Nil(t, err)
		}()
	}()
	wg.Wait()
}

// testRetrieve will fast retrieve a content from given node.
func testRetrieve(ctx context.Context, req *testNode, resp *testNode, outPath string, max int) error {
	offerChan := req.cofferProto.FindOffersAsync(ctx, crypto.FIL, resp.content, 1)
	pieceOffer := <-offerChan
	required := big.NewInt(0).Mul(big.NewFromGo(pieceOffer.PPB).Int, big.NewInt(int64(pieceOffer.Size)).Int)
	// Try retrieve directly
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	paychChan, errChan := req.activeOut.ListPaychsByPeer(subCtx, crypto.FIL, pieceOffer.RecipientAddr)
	exists := false
	for range paychChan {
		exists = true
		cancel()
	}
	err := <-errChan
	if err != nil {
		return err
	}
	var resCh string
	var resID uint64
	var payOffer *fcroffer.PayOffer
	if exists {
		resCh, resID, err = req.payMgr.ReserveForSelf(ctx, crypto.FIL, pieceOffer.RecipientAddr, required, pieceOffer.Expiration, pieceOffer.Inactivity)
		if err != nil {
			return err
		}
	} else {
		routes, errChan := req.rs.ListRoutesTo(ctx, crypto.FIL, pieceOffer.RecipientAddr)
		for route := range routes {
			temp, err := req.pofferProto.QueryOffer(ctx, crypto.FIL, route, required)
			if err != nil {
				return err
			}
			if payOffer == nil || temp.PPP.Cmp(payOffer.PPP) < 0 {
				payOffer = &temp
			}
		}
		err = <-errChan
		if err != nil {
			return err
		}
		if payOffer == nil {
			return fmt.Errorf("fail to find payment offer")
		}
		resCh, resID, err = req.payMgr.ReserveForSelfWithOffer(ctx, *payOffer)
		if err != nil {
			return err
		}
	}
	// Retrieve
	for i := 0; i < max; i++ {
		_, from, _ := req.signer.GetAddr(context.Background(), crypto.FIL)
		fmt.Println("Start retrieving", pieceOffer.ID, "from", from)
		_, errChan = req.retMgr.Retrieve(ctx, pieceOffer, payOffer, resCh, resID, outPath)
		err = <-errChan
		if err == nil {
			fmt.Println("Succeed in retrieving", pieceOffer.ID)
			return nil
		}
		fmt.Println("Fail to retrieve", pieceOffer.ID, err.Error(), "retry...")
		time.Sleep(time.Duration(rand2.Int63n(100)) * time.Millisecond)
	}
	fmt.Println("Exceed max attempts to retrieve", pieceOffer.ID)
	return err
}

// createPaych will fast create and serve a paych from given node to given node.
func createPaych(ctx context.Context, req *testNode, resp *testNode, bal big.Int) error {
	_, addr, err := resp.signer.GetAddr(ctx, crypto.FIL)
	if err != nil {
		return err
	}
	paychOffer, err := req.paychProto.QueryAdd(ctx, crypto.FIL, addr)
	if err != nil {
		return err
	}
	chAddr, err := req.transactor.Create(ctx, crypto.FIL, paychOffer.ToAddr, bal.Int)
	if err != nil {
		return err
	}
	err = req.paychProto.Add(ctx, chAddr, paychOffer)
	if err != nil {
		return err
	}
	return req.pservMgr.Serve(ctx, crypto.FIL, addr, chAddr, big.NewInt(10000).Int, big.NewInt(10).Int)
}

// getNode is used to create a testing node.
func getNode(ctx context.Context, index int) (*testNode, func(), error) {
	// New signer
	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "signer")})
	if err != nil {
		return nil, nil, err
	}
	prv := make([]byte, 32)
	rand.Read(prv)
	signer.SetKey(ctx, crypto.FIL, crypto.SECP256K1, prv)
	// New transactor
	confidence := uint64(0)
	transactor, err := trans.NewTransactorImpl(ctx, signer, trans.Opts{FilecoinEnabled: true, FilecoinAPI: testAPI, FilecoinConfidence: &confidence})
	if err != nil {
		return nil, nil, err
	}
	// New paych serving manager
	pservMgr, err := pservmgr.NewPaychServingManagerImpl(ctx, pservmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "pservmgr")})
	if err != nil {
		return nil, nil, err
	}
	// New reservation policy manager.
	reservMgr, err := reservmgr.NewReservationManagerImpl(ctx, reservmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "reservmgr")})
	if err != nil {
		return nil, nil, err
	}
	err = reservMgr.SetDefaultPolicy(ctx, crypto.FIL, true, nil)
	if err != nil {
		return nil, nil, err
	}
	// New route store
	rs, err := routestore.NewRouteStoreImpl(ctx, signer, routestore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "rs"), CleanFreq: time.Second, MaxHopFIL: 10})
	if err != nil {
		return nil, nil, err
	}
	// New sub store
	subs, err := substore.NewSubStoreImpl(ctx, substore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "subs")})
	if err != nil {
		return nil, nil, err
	}
	// New active in paych store.
	activeIn, err := paychstore.NewPaychStoreImpl(ctx, true, false, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "active-in")})
	if err != nil {
		return nil, nil, err
	}
	// New inactive in paych store.
	inactiveIn, err := paychstore.NewPaychStoreImpl(ctx, false, false, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "inactive-in")})
	if err != nil {
		return nil, nil, err
	}
	// New active out paych store.
	activeOut, err := paychstore.NewPaychStoreImpl(ctx, true, true, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "active-out")})
	if err != nil {
		return nil, nil, err
	}
	// New inactive out paych store.
	inactiveOut, err := paychstore.NewPaychStoreImpl(ctx, false, true, paychstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "inactive-out")})
	if err != nil {
		return nil, nil, err
	}
	// New payment manager.
	payMgr, err := paymgr.NewPaymentManagerImpl(ctx, activeOut, activeIn, inactiveOut, inactiveIn, pservMgr, reservMgr, rs, subs, transactor, signer, paymgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "paymgr"), ResCleanFreq: time.Second, CacheSyncFreq: time.Second, PeerCleanFreq: time.Second})
	if err != nil {
		return nil, nil, err
	}
	// New offer manager
	offerMgr, err := offermgr.NewOfferManagerImpl(ctx, offermgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "offermgr")})
	if err != nil {
		return nil, nil, err
	}
	// New settle manager
	settleMgr, err := settlemgr.NewSettlementManagerImpl(ctx, settlemgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "settlemgr")})
	if err != nil {
		return nil, nil, err
	}
	// Set default policy
	err = settleMgr.SetDefaultPolicy(ctx, crypto.FIL, time.Hour)
	if err != nil {
		return nil, nil, err
	}
	// New renew manager
	renewMgr, err := renewmgr.NewRenewManagerImpl(ctx, renewmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "renewmgr")})
	if err != nil {
		return nil, nil, err
	}
	// New peer manager
	peerMgr, err := peermgr.NewPeerManagerImpl(ctx, peermgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "peermgr")})
	if err != nil {
		return nil, nil, err
	}
	// New monitor
	monitor, err := paychmon.NewPaychMonitorImpl(ctx, payMgr, transactor, paychmon.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "paychmon"), CheckFreq: time.Second})
	if err != nil {
		return nil, nil, err
	}
	// New piece manager
	pieceMgr, err := piecemgr.NewPieceManagerImpl(ctx, piecemgr.Opts{PsPath: fmt.Sprintf("%v/%v-%v", testDS, index, "ps"), BsPath: fmt.Sprintf("%v/%v-%v", testDS, index, "bs"), BrefsPath: fmt.Sprintf("%v/%v-%v", testDS, index, "brefs")})
	if err != nil {
		return nil, nil, err
	}
	// New piece serving manager
	cservMgr, err := cservmgr.NewPieceServingManager(ctx, cservmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "cservmgr")})
	if err != nil {
		return nil, nil, err
	}
	// New miner proof store
	mps, err := mpstore.NewMinerProofStoreImpl(ctx, signer, mpstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "mpstore")})
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	// Protocols
	addrProto := addrproto.NewAddrProtocol(h, dualDHT, signer, peerMgr, addrproto.Opts{PublishFreq: 1 * time.Second})
	paychProto := paychproto.NewPaychProtocol(h, addrProto, signer, peerMgr, offerMgr, settleMgr, renewMgr, payMgr, monitor, paychproto.Opts{RenewWindow: 50})
	routeProto := routeproto.NewRouteProtocol(h, addrProto, paychProto, signer, peerMgr, pservMgr, rs, subs, routeproto.Opts{PublishFreq: 1 * time.Second, RouteExpiry: time.Hour})
	pofferProto := pofferproto.NewOfferProtocol(h, addrProto, signer, peerMgr, offerMgr, pservMgr, rs, payMgr, pofferproto.Opts{})
	cofferProto := cofferproto.NewOfferProtocol(h, dualDHT, addrProto, signer, peerMgr, offerMgr, pieceMgr, cservMgr, mps, cofferproto.Opts{PublishFreq: 1 * time.Second})
	payProto, err := payproto.NewPayProtocol(ctx, h, addrProto, signer, peerMgr, payMgr, payproto.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "payproto"), CleanFreq: time.Second, IOTimeout: 5 * time.Second})
	if err != nil {
		return nil, nil, err
	}
	retMgr, err := NewRetrievalManager(ctx, h, addrProto, payProto, signer, peerMgr, pieceMgr, Opts{IOTimeout: 5 * time.Second, OpTimeout: 5 * time.Second, CachePath: fmt.Sprintf("%v/%v-%v", testDS, index, "retcache"), TempPath: fmt.Sprintf("%v/%v-%v", testDS, index, "rettempdir")})
	if err != nil {
		return nil, nil, err
	}
	// Generate and import one file
	data := make([]byte, 10000000)
	rand.Read(data)
	os.WriteFile(fmt.Sprintf("%v/%v-%v", testDS, index, "tempFile"), data, os.ModePerm)
	id, err := pieceMgr.Import(ctx, fmt.Sprintf("%v/%v-%v", testDS, index, "tempFile"))
	if err != nil {
		return nil, nil, err
	}
	// Serve the file
	err = cservMgr.Serve(ctx, id, crypto.FIL, big.NewInt(1).Int)
	if err != nil {
		return nil, nil, err
	}
	node := &testNode{
		content:     id,
		h:           h,
		signer:      signer,
		transactor:  transactor,
		pieceMgr:    pieceMgr,
		pservMgr:    pservMgr,
		reservMgr:   reservMgr,
		rs:          rs,
		subs:        subs,
		activeIn:    activeIn,
		inactiveIn:  inactiveIn,
		activeOut:   activeOut,
		inactiveOut: inactiveOut,
		payMgr:      payMgr,
		offerMgr:    offerMgr,
		settleMgr:   settleMgr,
		renewMgr:    renewMgr,
		peerMgr:     peerMgr,
		monitor:     monitor,
		addrProto:   addrProto,
		paychProto:  paychProto,
		pofferProto: pofferProto,
		routeProto:  routeProto,
		cofferProto: cofferProto,
		payProto:    payProto,
		retMgr:      retMgr,
	}
	shutdown := func() {
		node.retMgr.Shutdown()
		node.payProto.Shutdown()
		node.cofferProto.Shutdown()
		node.routeProto.Shutdown()
		node.pofferProto.Shutdown()
		node.paychProto.Shutdown()
		node.monitor.Shutdown()
		node.peerMgr.Shutdown()
		node.renewMgr.Shutdown()
		node.settleMgr.Shutdown()
		node.offerMgr.Shutdown()
		node.payMgr.Shutdown()
		node.inactiveOut.Shutdown()
		node.activeOut.Shutdown()
		node.inactiveIn.Shutdown()
		node.activeIn.Shutdown()
		node.subs.Shutdown()
		node.rs.Shutdown()
		node.reservMgr.Shutdown()
		node.pservMgr.Shutdown()
		node.signer.Shutdown()
	}
	return node, shutdown, nil
}
