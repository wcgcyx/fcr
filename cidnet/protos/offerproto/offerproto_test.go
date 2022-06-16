package offerproto

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

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	routing "github.com/libp2p/go-libp2p-routing"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/cservmgr"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/mpstore"
	"github.com/wcgcyx/fcr/offermgr"
	"github.com/wcgcyx/fcr/peermgr"
	"github.com/wcgcyx/fcr/peernet/protos/addrproto"
	"github.com/wcgcyx/fcr/piecemgr"
)

const (
	testDS    = "./test-ds"
	testFile1 = "./test-ds/file1"
	testFile2 = "./test-ds/file2"
	testFile3 = "./test-ds/file3"
	testCID   = "bafkqafkunbuxgidjomqgcidumvzxiidgnfwgkibr"
)

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	// Create three test files.
	data := make([]byte, 1000)
	rand.Read(data)
	os.WriteFile(testFile1, data, os.ModePerm)
	rand.Read(data)
	os.WriteFile(testFile2, data, os.ModePerm)
	rand.Read(data)
	os.WriteFile(testFile3, data, os.ModePerm)
	m.Run()
}

func TestFiveNodes(t *testing.T) {
	ctx := context.Background()

	_, _, proto1, shutdown1, err := getNode(ctx, 0)
	assert.Nil(t, err)
	defer shutdown1()

	pc2, serv2, proto2, shutdown2, err := getNode(ctx, 1)
	assert.Nil(t, err)
	defer shutdown2()

	pc3, serv3, proto3, shutdown3, err := getNode(ctx, 2)
	assert.Nil(t, err)
	defer shutdown3()

	pc4, serv4, proto4, shutdown4, err := getNode(ctx, 3)
	assert.Nil(t, err)
	defer shutdown4()

	_, serv5, proto5, shutdown5, err := getNode(ctx, 4)
	assert.Nil(t, err)
	defer shutdown5()

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

	id2, err := pc2.Import(ctx, testFile1)
	assert.Nil(t, err)
	id3, err := pc3.Import(ctx, testFile2)
	assert.Nil(t, err)
	id4, err := pc4.Import(ctx, testFile3)
	assert.Nil(t, err)
	id5, err := cid.Parse(testCID)
	assert.Nil(t, err)

	err = serv2.Serve(ctx, id2, crypto.FIL, big.NewInt(1))
	assert.Nil(t, err)
	err = serv3.Serve(ctx, id3, crypto.FIL, big.NewInt(1))
	assert.Nil(t, err)
	err = serv4.Serve(ctx, id4, crypto.FIL, big.NewInt(1))
	assert.Nil(t, err)
	err = serv5.Serve(ctx, id5, crypto.FIL, big.NewInt(1))
	assert.Nil(t, err)

	time.Sleep(1 * time.Second)

	subCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	offerChan := proto1.FindOffersAsync(subCtx, 0, id2, 0)
	offers := make([]fcroffer.PieceOffer, 0)
	for offer := range offerChan {
		offers = append(offers, offer)
	}
	assert.Equal(t, 0, len(offers))

	subCtx, cancel = context.WithTimeout(ctx, time.Second)
	defer cancel()
	offerChan = proto1.FindOffersAsync(subCtx, crypto.FIL, id5, 0)
	offers = make([]fcroffer.PieceOffer, 0)
	for offer := range offerChan {
		offers = append(offers, offer)
	}
	assert.Equal(t, 0, len(offers))

	subCtx, cancel = context.WithTimeout(ctx, time.Second)
	defer cancel()
	offerChan = proto1.FindOffersAsync(subCtx, crypto.FIL, id2, 0)
	offers = make([]fcroffer.PieceOffer, 0)
	for offer := range offerChan {
		offers = append(offers, offer)
	}
	assert.Equal(t, 1, len(offers))

	subCtx, cancel = context.WithTimeout(ctx, time.Second)
	defer cancel()
	offerChan = proto1.FindOffersAsync(subCtx, crypto.FIL, id3, 0)
	offers = make([]fcroffer.PieceOffer, 0)
	for offer := range offerChan {
		offers = append(offers, offer)
	}
	assert.Equal(t, 1, len(offers))

	subCtx, cancel = context.WithTimeout(ctx, time.Second)
	defer cancel()
	offerChan = proto1.FindOffersAsync(subCtx, crypto.FIL, id4, 0)
	offers = make([]fcroffer.PieceOffer, 0)
	for offer := range offerChan {
		offers = append(offers, offer)
	}
	assert.Equal(t, 1, len(offers))
}

// getNode is used to create a offer proto node.
func getNode(ctx context.Context, index int) (piecemgr.PieceManager, cservmgr.PieceServingManager, *OfferProtocol, func(), error) {
	// New signer
	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "signer")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	prv := make([]byte, 32)
	rand.Read(prv)
	signer.SetKey(ctx, crypto.FIL, crypto.SECP256K1, prv)
	// New piece manager
	pieceMgr, err := piecemgr.NewPieceManagerImpl(ctx, piecemgr.Opts{PsPath: fmt.Sprintf("%v/%v-%v", testDS, index, "ps"), BsPath: fmt.Sprintf("%v/%v-%v", testDS, index, "bs"), BrefsPath: fmt.Sprintf("%v/%v-%v", testDS, index, "brefs")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New serving manager
	cservMgr, err := cservmgr.NewPieceServingManager(ctx, cservmgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "cservmgr")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New offer manager
	offerMgr, err := offermgr.NewOfferManagerImpl(ctx, offermgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "offermgr")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New miner proof store
	mps, err := mpstore.NewMinerProofStoreImpl(ctx, signer, mpstore.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "mpstore")})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// New peer manager
	peerMgr, err := peermgr.NewPeerManagerImpl(ctx, peermgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "peermgr")})
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
	oProto := NewOfferProtocol(h, dualDHT, aProto, signer, peerMgr, offerMgr, pieceMgr, cservMgr, mps, Opts{PublishFreq: 1 * time.Second})
	shutdown := func() {
		oProto.Shutdown()
		aProto.Shutdown()
		peerMgr.Shutdown()
		mps.Shutdown()
		offerMgr.Shutdown()
		cservMgr.Shutdown()
		pieceMgr.Shutdown()
		signer.Shutdown()
	}
	return pieceMgr, cservMgr, oProto, shutdown, nil
}
