package io

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
	"encoding/binary"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
)

func TestRoundTrip(t *testing.T) {
	ctx := context.Background()

	h1, err := libp2p.New()
	assert.Nil(t, err)
	h2, err := libp2p.New()
	assert.Nil(t, err)
	addr2 := peer.AddrInfo{
		ID:    h2.ID(),
		Addrs: h2.Addrs(),
	}
	err = h1.Connect(ctx, addr2)
	assert.Nil(t, err)

	h2.SetStreamHandler("/test-protocol", func(conn network.Stream) {
		defer conn.Close()
		data, err := Read(ctx, conn)
		assert.Nil(t, err)
		assert.Equal(t, []byte{1, 2, 3, 4}, data)
	})

	conn, err := h1.NewStream(ctx, h2.ID(), "/test-protocol")
	assert.Nil(t, err)
	err = Write(ctx, conn, []byte{1, 2, 3, 4})
	assert.Nil(t, err)

	// Test failure cases
	h2.SetStreamHandler("/test-protocol-failure", func(conn network.Stream) {
		defer conn.Close()
		_, err := Read(ctx, conn)
		assert.NotNil(t, err)
	})

	conn, err = h1.NewStream(ctx, h2.ID(), "/test-protocol-failure")
	assert.Nil(t, err)
	subCtx, cancel := context.WithCancel(ctx)
	cancel()
	err = Write(subCtx, conn, []byte{1, 2, 3, 4})
	assert.NotNil(t, err)

	largeData := make([]byte, MaxDataLength+1)
	err = Write(ctx, conn, largeData)
	assert.NotNil(t, err)

	conn, err = h1.NewStream(ctx, h2.ID(), "/test-protocol-failure")
	assert.Nil(t, err)
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(MaxDataLength+1))
	_, err = conn.Write(append(length, largeData...))
	assert.Nil(t, err)

	h2.SetStreamHandler("/test-protocol-failure-2", func(conn network.Stream) {
		defer conn.Close()
		subCtx, cancel := context.WithCancel(ctx)
		cancel()
		_, err := Read(subCtx, conn)
		assert.NotNil(t, err)
	})

	conn, err = h1.NewStream(ctx, h2.ID(), "/test-protocol-failure-2")
	assert.Nil(t, err)
	err = Write(ctx, conn, []byte{1, 2, 3, 4})
	assert.Nil(t, err)
}
