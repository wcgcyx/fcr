package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

const (
	testPeerID = "QmXm6tWTUnaa5aep6vAwYkEAMpn8C5YFMgX9bh9BRW8Vi1"
	testIP     = "/ip4/127.0.0.1/tcp/9010"
)

func TestSubscribeRoundTrip(t *testing.T) {
	pid, err := peer.Decode(testPeerID)
	assert.Empty(t, err)
	maddr, err := multiaddr.NewMultiaddr(testIP)
	assert.Empty(t, err)
	data := EncodeSubscribe(1, peer.AddrInfo{
		ID:    pid,
		Addrs: []multiaddr.Multiaddr{maddr},
	})
	currencyID, pi, err := DecodeSubscribe(data)
	assert.Empty(t, err)
	assert.Equal(t, uint64(1), currencyID)
	assert.Equal(t, pid, pi.ID)
	assert.Equal(t, []multiaddr.Multiaddr{maddr}, pi.Addrs)
}
