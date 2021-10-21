package pubsubstore

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

func TestNewPubSubStore(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	pss, err := NewPubSubStoreImplV1(ctx, testDB, []uint64{0, 1, 2})
	assert.Empty(t, err)
	assert.NotEmpty(t, pss)
	cancel()
	time.Sleep(1 * time.Second)
}

func TestAddSubscribing(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	pss, err := NewPubSubStoreImplV1(ctx, testDB, []uint64{0, 1, 2})
	assert.Empty(t, err)
	assert.NotEmpty(t, pss)

	pid1, err := peer.Decode(testPeerID1)
	assert.Empty(t, err)
	pid2, err := peer.Decode(testPeerID2)
	assert.Empty(t, err)
	pid3, err := peer.Decode(testPeerID3)
	assert.Empty(t, err)
	err = pss.AddSubscribing(3, pid1)
	assert.NotEmpty(t, err)
	err = pss.AddSubscribing(0, pid1)
	assert.Empty(t, err)
	err = pss.AddSubscribing(2, pid1)
	assert.Empty(t, err)
	err = pss.AddSubscribing(1, pid2)
	assert.Empty(t, err)
	err = pss.AddSubscribing(2, pid3)
	assert.Empty(t, err)

	res, err := pss.ListSubscribings(context.Background())
	assert.Empty(t, err)
	assert.Equal(t, []peer.ID{pid1}, res[0])
	assert.Equal(t, []peer.ID{pid2}, res[1])
	assert.ElementsMatch(t, []peer.ID{pid1, pid3}, res[2])

	err = pss.RemoveSubscribing(3, pid1)
	assert.NotEmpty(t, err)

	err = pss.RemoveSubscribing(0, pid1)
	assert.Empty(t, err)

	res, err = pss.ListSubscribings(context.Background())
	_, ok := res[0]
	assert.Empty(t, err)
	assert.False(t, ok)
	assert.Equal(t, []peer.ID{pid2}, res[1])
	assert.ElementsMatch(t, []peer.ID{pid1, pid3}, res[2])
	cancel()
	time.Sleep(1 * time.Second)
}

func TestAddSubscribers(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	pss, err := NewPubSubStoreImplV1(ctx, testDB, []uint64{0, 1, 2})
	assert.Empty(t, err)
	assert.NotEmpty(t, pss)

	pid1, err := peer.Decode(testPeerID1)
	assert.Empty(t, err)
	pid2, err := peer.Decode(testPeerID2)
	assert.Empty(t, err)
	pid3, err := peer.Decode(testPeerID3)
	assert.Empty(t, err)
	err = pss.AddSubscriber(3, pid1, time.Minute)
	assert.NotEmpty(t, err)
	err = pss.AddSubscriber(0, pid1, time.Minute)
	assert.Empty(t, err)
	err = pss.AddSubscriber(2, pid1, time.Minute)
	assert.Empty(t, err)
	err = pss.AddSubscriber(1, pid2, time.Minute)
	assert.Empty(t, err)
	err = pss.AddSubscriber(2, pid3, time.Minute)
	assert.Empty(t, err)

	res, err := pss.ListSubscribers(context.Background())
	assert.Empty(t, err)
	assert.Equal(t, []peer.ID{pid1}, res[0])
	assert.Equal(t, []peer.ID{pid2}, res[1])
	assert.ElementsMatch(t, []peer.ID{pid1, pid3}, res[2])

	err = pss.AddSubscriber(0, pid1, 3*time.Second)
	assert.Empty(t, err)

	res, err = pss.ListSubscribers(context.Background())
	assert.Empty(t, err)
	assert.Equal(t, []peer.ID{pid1}, res[0])

	time.Sleep(3 * time.Second)

	res, err = pss.ListSubscribers(context.Background())
	assert.Empty(t, err)
	_, ok := res[0]
	assert.False(t, ok)
	cancel()
	time.Sleep(1 * time.Second)
}
