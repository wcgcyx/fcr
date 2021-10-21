package io

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
)

func TestRoundTrip(t *testing.T) {
	ctx := context.Background()
	h1, err := libp2p.New(ctx)
	assert.Empty(t, err)
	h2, err := libp2p.New(ctx)
	assert.Empty(t, err)
	addr2 := peer.AddrInfo{
		ID:    h2.ID(),
		Addrs: h2.Addrs(),
	}
	err = h1.Connect(ctx, addr2)
	assert.Empty(t, err)

	h2.SetStreamHandler("/test-protocol", func(conn network.Stream) {
		defer conn.Close()
		data, err := Read(conn, time.Minute)
		assert.Empty(t, err)
		assert.Equal(t, []byte{1, 2, 3, 4}, data)
	})

	conn, err := h1.NewStream(ctx, h2.ID(), "/test-protocol")
	assert.Empty(t, err)
	err = Write(conn, []byte{1, 2, 3, 4}, time.Minute)
	assert.Empty(t, err)
}
