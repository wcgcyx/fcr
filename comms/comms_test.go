package comms

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
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/crypto"
)

const (
	testDS = "./test-ds"
)

var testKey1 = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1}
var testKey2 = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 2}

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	m.Run()
}

func TestRoundTrip(t *testing.T) {
	ctx := context.Background()

	h1, err := libp2p.New()
	assert.Nil(t, err)
	h2, err := libp2p.New()
	assert.Nil(t, err)
	pi2 := peer.AddrInfo{
		ID:    h2.ID(),
		Addrs: h2.Addrs(),
	}
	err = h1.Connect(ctx, pi2)
	assert.Nil(t, err)

	// Signer 1.
	signer1, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: fmt.Sprintf("%v/%v", testDS, 1)})
	assert.Nil(t, err)
	err = signer1.SetKey(ctx, crypto.FIL, crypto.SECP256K1, testKey1)
	assert.Nil(t, err)
	_, addr1, err := signer1.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)
	// Signer 2.
	signer2, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: fmt.Sprintf("%v/%v", testDS, 2)})
	assert.Nil(t, err)
	err = signer2.SetKey(ctx, crypto.FIL, crypto.SECP256K1, testKey2)
	assert.Nil(t, err)
	_, addr2, err := signer2.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	// Set handler
	h2.SetStreamHandler("/test-protocol", func(conn network.Stream) {
		defer conn.Close()
		req, err := NewRequestIn(ctx, time.Minute, time.Minute, conn, signer2)
		assert.Nil(t, err)

		// Check request
		assert.Equal(t, crypto.FIL, req.CurrencyID)
		assert.Equal(t, addr1, req.FromAddr)
		assert.Equal(t, addr2, req.ToAddr)

		// Receive three times.
		reqData, err := req.Receive(ctx, time.Minute)
		assert.Nil(t, err)
		assert.Equal(t, []byte{1, 2, 3, 4}, reqData)
		err = req.Respond(ctx, time.Minute, time.Minute, true, []byte{4, 3, 2, 1})
		assert.Nil(t, err)

		reqData, err = req.Receive(ctx, time.Minute)
		assert.Nil(t, err)
		assert.Equal(t, []byte{2, 2, 3, 4}, reqData)
		err = req.Respond(ctx, time.Minute, time.Minute, true, []byte{3, 3, 2, 1})
		assert.Nil(t, err)

		reqData, err = req.Receive(ctx, time.Minute)
		assert.Nil(t, err)
		assert.Equal(t, []byte{3, 2, 3, 4}, reqData)
		err = req.Respond(ctx, time.Minute, time.Minute, true, []byte{2, 3, 2, 1})
		assert.Nil(t, err)
	})

	conn, err := h1.NewStream(ctx, h2.ID(), "/test-protocol")
	assert.Nil(t, err)
	req, err := NewRequestOut(ctx, time.Minute, time.Minute, conn, signer1, crypto.FIL, addr2)
	assert.Nil(t, err)

	// Send three times.
	err = req.Send(ctx, time.Minute, time.Minute, []byte{1, 2, 3, 4})
	assert.Nil(t, err)
	status, resp, err := req.GetResponse(ctx, time.Minute)
	assert.Nil(t, err)
	assert.True(t, status)
	assert.Equal(t, []byte{4, 3, 2, 1}, resp)

	err = req.Send(ctx, time.Minute, time.Minute, []byte{2, 2, 3, 4})
	assert.Nil(t, err)
	status, resp, err = req.GetResponse(ctx, time.Minute)
	assert.Nil(t, err)
	assert.True(t, status)
	assert.Equal(t, []byte{3, 3, 2, 1}, resp)

	err = req.Send(ctx, time.Minute, time.Minute, []byte{3, 2, 3, 4})
	assert.Nil(t, err)
	status, resp, err = req.GetResponse(ctx, time.Minute)
	assert.Nil(t, err)
	assert.True(t, status)
	assert.Equal(t, []byte{2, 3, 2, 1}, resp)
}
