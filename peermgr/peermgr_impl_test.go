package peermgr

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

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/crypto"
)

const (
	testDS     = "./test-ds"
	testPeerID = "QmXm6tWTUnaa5aep6vAwYkEAMpn8C5YFMgX9bh9BRW8Vi1"
)

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	m.Run()
}

func TestPeerManagerImpl(t *testing.T) {
	ctx := context.Background()

	_, err := NewPeerManagerImpl(ctx, Opts{})
	assert.NotNil(t, err)

	peerMgr, err := NewPeerManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer peerMgr.Shutdown()
}

func TestAddPeer(t *testing.T) {
	ctx := context.Background()

	peerMgr, err := NewPeerManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer peerMgr.Shutdown()

	pid, err := peer.Decode(testPeerID)
	assert.Nil(t, err)
	maddr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/9010")
	assert.Nil(t, err)

	err = peerMgr.AddPeer(ctx, crypto.FIL, "peer1", peer.AddrInfo{ID: pid, Addrs: []multiaddr.Multiaddr{maddr}})
	assert.Nil(t, err)

	maddr, err = multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/9011")
	assert.Nil(t, err)

	err = peerMgr.AddPeer(ctx, crypto.FIL, "peer1", peer.AddrInfo{ID: pid, Addrs: []multiaddr.Multiaddr{maddr}})
	assert.Nil(t, err)

	err = peerMgr.AddPeer(ctx, crypto.FIL, "peer1", peer.AddrInfo{ID: pid, Addrs: []multiaddr.Multiaddr{maddr}})
	assert.Nil(t, err)

	maddr, err = multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/9012")
	assert.Nil(t, err)

	err = peerMgr.AddPeer(ctx, crypto.FIL, "peer2", peer.AddrInfo{ID: pid, Addrs: []multiaddr.Multiaddr{maddr}})
	assert.Nil(t, err)
}

func TestRemovePeer(t *testing.T) {
	ctx := context.Background()

	peerMgr, err := NewPeerManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer peerMgr.Shutdown()

	_, err = peerMgr.PeerAddr(ctx, 0, "peer1")
	assert.NotNil(t, err)

	_, err = peerMgr.PeerAddr(ctx, crypto.FIL, "peer0")
	assert.NotNil(t, err)

	pi, err := peerMgr.PeerAddr(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)
	assert.Equal(t, testPeerID, pi.ID.String())
	assert.Equal(t, "/ip4/127.0.0.1/tcp/9011", pi.Addrs[0].String())

	exists, err := peerMgr.HasPeer(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)
	assert.True(t, exists)

	err = peerMgr.RemovePeer(ctx, crypto.FIL, "peer0")
	assert.NotNil(t, err)

	err = peerMgr.RemovePeer(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)

	_, err = peerMgr.PeerAddr(ctx, crypto.FIL, "peer1")
	assert.NotNil(t, err)

	exists, err = peerMgr.HasPeer(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)
	assert.False(t, exists)

	pid, err := peer.Decode(testPeerID)
	assert.Nil(t, err)

	maddr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/9011")
	assert.Nil(t, err)

	err = peerMgr.AddPeer(ctx, crypto.FIL, "peer1", peer.AddrInfo{ID: pid, Addrs: []multiaddr.Multiaddr{maddr}})
	assert.Nil(t, err)
}

func TestBlock(t *testing.T) {
	ctx := context.Background()

	peerMgr, err := NewPeerManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer peerMgr.Shutdown()

	_, err = peerMgr.IsBlocked(ctx, 0, "peer1")
	assert.NotNil(t, err)

	_, err = peerMgr.IsBlocked(ctx, crypto.FIL, "peer0")
	assert.NotNil(t, err)

	blocked, err := peerMgr.IsBlocked(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)
	assert.False(t, blocked)

	err = peerMgr.BlockPeer(ctx, 0, "peer1")
	assert.NotNil(t, err)

	err = peerMgr.BlockPeer(ctx, crypto.FIL, "peer0")
	assert.NotNil(t, err)

	err = peerMgr.BlockPeer(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)

	blocked, err = peerMgr.IsBlocked(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)
	assert.True(t, blocked)

	err = peerMgr.BlockPeer(ctx, crypto.FIL, "peer1")
	assert.NotNil(t, err)

	err = peerMgr.UnblockPeer(ctx, 0, "peer1")
	assert.NotNil(t, err)

	err = peerMgr.UnblockPeer(ctx, crypto.FIL, "peer0")
	assert.NotNil(t, err)

	err = peerMgr.UnblockPeer(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)

	blocked, err = peerMgr.IsBlocked(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)
	assert.False(t, blocked)

	err = peerMgr.UnblockPeer(ctx, crypto.FIL, "peer1")
	assert.NotNil(t, err)
}

func TestRecord(t *testing.T) {
	ctx := context.Background()

	peerMgr, err := NewPeerManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer peerMgr.Shutdown()

	err = peerMgr.AddToHistory(ctx, 0, "peer1", Record{Description: "rec1", CreatedAt: time.Now().Add(time.Second)})
	assert.NotNil(t, err)

	err = peerMgr.AddToHistory(ctx, crypto.FIL, "peer0", Record{Description: "rec1", CreatedAt: time.Now().Add(time.Second)})
	assert.NotNil(t, err)

	err = peerMgr.AddToHistory(ctx, crypto.FIL, "peer1", Record{Description: "rec1", CreatedAt: time.Now().Add(time.Second)})
	assert.Nil(t, err)

	err = peerMgr.AddToHistory(ctx, crypto.FIL, "peer1", Record{Description: "rec2", CreatedAt: time.Now().Add(2 * time.Second)})
	assert.Nil(t, err)

	err = peerMgr.AddToHistory(ctx, crypto.FIL, "peer1", Record{Description: "rec3", CreatedAt: time.Now().Add(3 * time.Second)})
	assert.Nil(t, err)

	err = peerMgr.AddToHistory(ctx, crypto.FIL, "peer2", Record{Description: "rec4", CreatedAt: time.Now().Add(time.Second)})
	assert.Nil(t, err)

	err = peerMgr.AddToHistory(ctx, crypto.FIL, "peer2", Record{Description: "rec5", CreatedAt: time.Now().Add(2 * time.Second)})
	assert.Nil(t, err)

	err = peerMgr.AddToHistory(ctx, crypto.FIL, "peer2", Record{Description: "rec6", CreatedAt: time.Now().Add(3 * time.Second)})
	assert.Nil(t, err)

	err = peerMgr.SetRecID(ctx, crypto.FIL, "peer2", big.NewInt(0))
	assert.Nil(t, err)

	// It should overwrite rec4
	err = peerMgr.AddToHistory(ctx, crypto.FIL, "peer2", Record{Description: "rec7", CreatedAt: time.Now().Add(3 * time.Second)})
	assert.Nil(t, err)
}

func TestList(t *testing.T) {
	ctx := context.Background()

	peerMgr, err := NewPeerManagerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer peerMgr.Shutdown()

	curChan, errChan := peerMgr.ListCurrencyIDs(ctx)
	currencyIDs := make([]byte, 0)
	for currencyID := range curChan {
		currencyIDs = append(currencyIDs, currencyID)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(currencyIDs))

	peerChan, errChan := peerMgr.ListPeers(ctx, crypto.FIL)
	peers := make([]string, 0)
	for peer := range peerChan {
		peers = append(peers, peer)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 2, len(peers))

	recIDChan, recChan, errChan := peerMgr.ListHistory(ctx, crypto.FIL, "peer1")
	recIDs := make([]*big.Int, 0)
	records := make([]Record, 0)
	for recID := range recIDChan {
		record := <-recChan
		recIDs = append(recIDs, recID)
		records = append(records, record)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 3, len(recIDs))
	assert.Equal(t, 3, len(records))
	assert.Equal(t, []*big.Int{big.NewInt(3), big.NewInt(2), big.NewInt(1)}, recIDs)
	assert.Equal(t, []string{"rec3", "rec2", "rec1"}, []string{records[0].Description, records[1].Description, records[2].Description})

	recIDChan, recChan, errChan = peerMgr.ListHistory(ctx, crypto.FIL, "peer2")
	recIDs = make([]*big.Int, 0)
	records = make([]Record, 0)
	for recID := range recIDChan {
		record := <-recChan
		recIDs = append(recIDs, recID)
		records = append(records, record)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 3, len(recIDs))
	assert.Equal(t, 3, len(records))
	assert.Equal(t, []*big.Int{big.NewInt(1), big.NewInt(3), big.NewInt(2)}, recIDs)
	assert.Equal(t, []string{"rec7", "rec6", "rec5"}, []string{records[0].Description, records[1].Description, records[2].Description})

	err = peerMgr.RemoveRecord(ctx, crypto.FIL, "peer1", big.NewInt(100))
	assert.NotNil(t, err)

	err = peerMgr.RemoveRecord(ctx, crypto.FIL, "peer1", big.NewInt(1))
	assert.Nil(t, err)

	recIDChan, recChan, errChan = peerMgr.ListHistory(ctx, crypto.FIL, "peer1")
	recIDs = make([]*big.Int, 0)
	records = make([]Record, 0)
	for recID := range recIDChan {
		record := <-recChan
		recIDs = append(recIDs, recID)
		records = append(records, record)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 2, len(recIDs))
	assert.Equal(t, 2, len(records))
	assert.Equal(t, []*big.Int{big.NewInt(3), big.NewInt(2)}, recIDs)
	assert.Equal(t, []string{"rec3", "rec2"}, []string{records[0].Description, records[1].Description})

	err = peerMgr.RemovePeer(ctx, crypto.FIL, "peer2")
	assert.Nil(t, err)

	recIDChan, recChan, errChan = peerMgr.ListHistory(ctx, crypto.FIL, "peer2")
	recIDs = make([]*big.Int, 0)
	records = make([]Record, 0)
	for recID := range recIDChan {
		record := <-recChan
		recIDs = append(recIDs, recID)
		records = append(records, record)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(recIDs))
	assert.Equal(t, 0, len(records))
}
