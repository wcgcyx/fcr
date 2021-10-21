package peermgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
)

const (
	testDB      = "./testdb"
	testPeerID1 = "QmXm6tWTUnaa5aep6vAwYkEAMpn8C5YFMgX9bh9BRW8Vi1"
	testPeerID2 = "QmXm6tWTUnaa5aep6vAwYkEAMpn8C5YFMgX9bh9BRW8Vi2"
	testPeerID3 = "QmXm6tWTUnaa5aep6vAwYkEAMpn8C5YFMgX9bh9BRW8Vi3"
)

func TestNewPeerManager(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	mgr, err := NewPeerManagerImplV1(ctx, testDB, "cid", []uint64{0, 1, 2})
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr)
	cancel()
	time.Sleep(1 * time.Second)
}

func TestAddPeer(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	mgr, err := NewPeerManagerImplV1(ctx, testDB, "cid", []uint64{0, 1, 2})
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr)

	pid1, err := peer.Decode(testPeerID1)
	assert.Empty(t, err)
	pid2, err := peer.Decode(testPeerID2)
	assert.Empty(t, err)
	pid3, err := peer.Decode(testPeerID3)
	assert.Empty(t, err)
	err = mgr.AddPeer(3, "testaddr1", pid1)
	assert.NotEmpty(t, err)

	err = mgr.AddPeer(0, "testaddr1", pid1)
	assert.Empty(t, err)
	err = mgr.AddPeer(1, "testaddr1", pid1)
	assert.Empty(t, err)
	err = mgr.AddPeer(1, "testaddr2", pid2)
	assert.Empty(t, err)
	err = mgr.AddPeer(2, "testaddr3", pid3)
	assert.Empty(t, err)

	mgr.Record(3, "testaddr1", true)
	mgr.Record(0, "testaddr4", true)
	mgr.Record(0, "testaddr1", true)
	mgr.Record(0, "testaddr1", true)
	mgr.Record(0, "testaddr1", false)
	ppid1, _, success, failure, err := mgr.GetPeerInfo(0, "testaddr1")
	assert.Empty(t, err)
	assert.Equal(t, ppid1, pid1)
	assert.Equal(t, 2, success)
	assert.Equal(t, 1, failure)

	err = mgr.AddPeer(0, "testaddr1", pid2)
	assert.Empty(t, err)
	ppid1, _, success, failure, err = mgr.GetPeerInfo(0, "testaddr1")
	assert.Empty(t, err)
	assert.Equal(t, ppid1, pid2)
	assert.Equal(t, 2, success)
	assert.Equal(t, 1, failure)

	res, err := mgr.ListPeers(context.Background())
	assert.Empty(t, err)
	assert.Equal(t, []string{"testaddr1"}, res[0])
	assert.ElementsMatch(t, []string{"testaddr1", "testaddr2"}, res[1])
	assert.Equal(t, []string{"testaddr3"}, res[2])
	cancel()
	time.Sleep(1 * time.Second)
}

func TestBlockPeer(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	mgr, err := NewPeerManagerImplV1(ctx, testDB, "cid", []uint64{0, 1, 2})
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr)

	pid1, err := peer.Decode(testPeerID1)
	assert.Empty(t, err)
	pid2, err := peer.Decode(testPeerID2)
	assert.Empty(t, err)
	pid3, err := peer.Decode(testPeerID3)
	assert.Empty(t, err)
	err = mgr.AddPeer(3, "testaddr1", pid1)
	assert.NotEmpty(t, err)

	err = mgr.AddPeer(0, "testaddr1", pid1)
	assert.Empty(t, err)
	err = mgr.AddPeer(1, "testaddr1", pid1)
	assert.Empty(t, err)
	err = mgr.AddPeer(1, "testaddr2", pid2)
	assert.Empty(t, err)
	err = mgr.AddPeer(2, "testaddr3", pid3)
	assert.Empty(t, err)

	_, blocked, _, _, err := mgr.GetPeerInfo(0, "testaddr1")
	assert.Empty(t, err)
	assert.False(t, blocked)

	err = mgr.BlockPeer(3, "testaddr1")
	assert.NotEmpty(t, err)

	err = mgr.BlockPeer(0, "testaddr4")
	assert.NotEmpty(t, err)

	err = mgr.BlockPeer(0, "testaddr1")
	assert.Empty(t, err)

	_, blocked, _, _, err = mgr.GetPeerInfo(0, "testaddr1")
	assert.Empty(t, err)
	assert.True(t, blocked)

	err = mgr.UnblockPeer(3, "testaddr1")
	assert.NotEmpty(t, err)

	err = mgr.UnblockPeer(0, "testaddr4")
	assert.NotEmpty(t, err)

	err = mgr.UnblockPeer(0, "testaddr1")
	assert.Empty(t, err)

	_, blocked, _, _, err = mgr.GetPeerInfo(0, "testaddr1")
	assert.Empty(t, err)
	assert.False(t, blocked)
	cancel()
	time.Sleep(1 * time.Second)
}
