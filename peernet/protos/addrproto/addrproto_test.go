package addrproto

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
	"github.com/wcgcyx/fcr/peermgr"
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

func TestFiveNodes(t *testing.T) {
	ctx := context.Background()
	// Node 1
	proto1, shutdown1, err := getNode(ctx, 0)
	assert.Nil(t, err)
	defer shutdown1()

	// Node 2
	proto2, shutdown2, err := getNode(ctx, 1)
	assert.Nil(t, err)
	defer shutdown2()

	// Node 3
	proto3, shutdown3, err := getNode(ctx, 2)
	assert.Nil(t, err)
	defer shutdown3()

	// Node 4
	proto4, shutdown4, err := getNode(ctx, 3)
	assert.Nil(t, err)
	defer shutdown4()

	// Node 5
	proto5, shutdown5, err := getNode(ctx, 4)
	assert.Nil(t, err)
	defer shutdown5()

	// Connect each other
	err = proto2.h.Connect(ctx, peer.AddrInfo{ID: proto1.h.ID(), Addrs: proto1.h.Addrs()})
	assert.Nil(t, err)
	err = proto3.h.Connect(ctx, peer.AddrInfo{ID: proto1.h.ID(), Addrs: proto1.h.Addrs()})
	assert.Nil(t, err)
	err = proto4.h.Connect(ctx, peer.AddrInfo{ID: proto1.h.ID(), Addrs: proto1.h.Addrs()})
	assert.Nil(t, err)
	err = proto5.h.Connect(ctx, peer.AddrInfo{ID: proto1.h.ID(), Addrs: proto1.h.Addrs()})
	assert.Nil(t, err)
	time.Sleep(1 * time.Second)

	assert.Nil(t, proto1.Publish(ctx))
	assert.Nil(t, proto2.Publish(ctx))
	assert.Nil(t, proto3.Publish(ctx))
	assert.Nil(t, proto4.Publish(ctx))
	assert.Nil(t, proto5.Publish(ctx))

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

	_, err = proto1.ConnectToPeer(ctx, crypto.FIL, addr1)
	assert.NotNil(t, err)

	_, err = proto1.ConnectToPeer(ctx, 0, addr2)
	assert.NotNil(t, err)

	id, err := proto1.ConnectToPeer(ctx, crypto.FIL, addr2)
	assert.Nil(t, err)
	assert.Equal(t, proto2.h.ID(), id)

	id, err = proto1.ConnectToPeer(ctx, crypto.FIL, addr3)
	assert.Nil(t, err)
	assert.Equal(t, proto3.h.ID(), id)

	id, err = proto1.ConnectToPeer(ctx, crypto.FIL, addr4)
	assert.Nil(t, err)
	assert.Equal(t, proto4.h.ID(), id)

	id, err = proto1.ConnectToPeer(ctx, crypto.FIL, addr5)
	assert.Nil(t, err)
	assert.Equal(t, proto5.h.ID(), id)

	id, err = proto1.ConnectToPeer(ctx, crypto.FIL, addr2)
	assert.Nil(t, err)
	assert.Equal(t, proto2.h.ID(), id)
}

// getNode is used to create an addr proto node.
func getNode(ctx context.Context, index int) (*AddrProtocol, func(), error) {
	// New signer
	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "signer")})
	if err != nil {
		return nil, nil, err
	}
	prv := make([]byte, 32)
	rand.Read(prv)
	signer.SetKey(ctx, crypto.FIL, crypto.SECP256K1, prv)
	// New peer manager
	peerMgr, err := peermgr.NewPeerManagerImpl(ctx, peermgr.Opts{Path: fmt.Sprintf("%v/%v-%v", testDS, index, "peermgr")})
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
	proto := NewAddrProtocol(h, dualDHT, signer, peerMgr, Opts{IOTimeout: time.Minute, OpTimeout: time.Minute, PublishFreq: 5 * time.Second})
	shutdown := func() {
		proto.Shutdown()
		peerMgr.Shutdown()
		signer.Shutdown()
	}
	return proto, shutdown, nil
}
