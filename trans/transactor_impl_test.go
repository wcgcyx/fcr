package trans

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

	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/crypto"
)

const (
	testDS = "./test-ds"
)

var testKey = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1}
var testAddr = "f1ssv6id634b4q4mecyqsfqdibj3vqdhn6piro4yq"
var testPeer = "f1tynlzcntp3nmrpbbjjchq666zv6s6clr3pfdt3y"
var testCh = ""
var testAPI = ""

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	server := NewMockFil(time.Millisecond)
	defer server.Shutdown()
	testAPI = server.GetAPI()
	m.Run()
}

func TestNewTransactorImpl(t *testing.T) {
	ctx := context.Background()

	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: testDS})
	assert.Nil(t, err)
	defer signer.Shutdown()
	err = signer.SetKey(ctx, crypto.FIL, crypto.SECP256K1, testKey)
	assert.Nil(t, err)

	transactor, err := NewTransactorImpl(ctx, signer, Opts{})
	assert.NotNil(t, err)
	assert.Nil(t, transactor)

	confidence := uint64(0)
	transactor, err = NewTransactorImpl(
		ctx,
		signer,
		Opts{
			FilecoinEnabled:    true,
			FilecoinAPI:        testAPI,
			FilecoinAuthToken:  "",
			FilecoinConfidence: &confidence})
	assert.Nil(t, err)
	assert.NotNil(t, transactor)
}

func TestCreate(t *testing.T) {
	ctx := context.Background()

	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: testDS})
	assert.Nil(t, err)
	defer signer.Shutdown()

	confidence := uint64(0)
	transactor, err := NewTransactorImpl(
		ctx,
		signer,
		Opts{
			FilecoinEnabled:    true,
			FilecoinAPI:        testAPI,
			FilecoinAuthToken:  "",
			FilecoinConfidence: &confidence})
	assert.Nil(t, err)
	assert.NotNil(t, transactor)

	chAddr, err := transactor.Create(ctx, crypto.FIL, testPeer, big.NewInt(100))
	assert.Nil(t, err)
	testCh = chAddr
}

func TestTopup(t *testing.T) {
	ctx := context.Background()

	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: testDS})
	assert.Nil(t, err)
	defer signer.Shutdown()

	confidence := uint64(0)
	transactor, err := NewTransactorImpl(
		ctx,
		signer,
		Opts{
			FilecoinEnabled:    true,
			FilecoinAPI:        testAPI,
			FilecoinAuthToken:  "",
			FilecoinConfidence: &confidence})
	assert.Nil(t, err)
	assert.NotNil(t, transactor)

	_, _, _, _, balance, sender, recipient, err := transactor.Check(ctx, crypto.FIL, testCh)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(100), balance)
	assert.Equal(t, testAddr, sender)
	assert.Equal(t, testPeer, recipient)

	err = transactor.Topup(ctx, crypto.FIL, testCh, big.NewInt(100))
	assert.Nil(t, err)

	_, _, _, _, balance, _, _, err = transactor.Check(ctx, crypto.FIL, testCh)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(200), balance)
}

func TestVoucher(t *testing.T) {
	ctx := context.Background()

	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: testDS})
	assert.Nil(t, err)
	defer signer.Shutdown()

	confidence := uint64(0)
	transactor, err := NewTransactorImpl(
		ctx,
		signer,
		Opts{
			FilecoinEnabled:    true,
			FilecoinAPI:        testAPI,
			FilecoinAuthToken:  "",
			FilecoinConfidence: &confidence})
	assert.Nil(t, err)
	assert.NotNil(t, transactor)

	voucher, err := transactor.GenerateVoucher(ctx, crypto.FIL, testCh, 0, 1, big.NewInt(10))
	assert.Nil(t, err)
	sender, chAddr, lane, nonce, redeemed, err := transactor.VerifyVoucher(crypto.FIL, voucher)
	assert.Nil(t, err)
	assert.Equal(t, testAddr, sender)
	assert.Equal(t, testCh, chAddr)
	assert.Equal(t, uint64(0), lane)
	assert.Equal(t, uint64(1), nonce)
	assert.Equal(t, big.NewInt(10), redeemed)

	err = transactor.Update(ctx, crypto.FIL, chAddr, voucher)
	assert.Nil(t, err)

	settlingAt, _, _, redeemed, _, _, _, err := transactor.Check(ctx, crypto.FIL, testCh)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(10), redeemed)
	assert.Equal(t, int64(0), settlingAt)

	err = transactor.Settle(ctx, crypto.FIL, testCh)
	assert.Nil(t, err)
	settlingAt, _, _, _, _, _, _, err = transactor.Check(ctx, crypto.FIL, testCh)
	assert.Nil(t, err)
	assert.NotEqual(t, int64(0), settlingAt)

	err = transactor.Collect(ctx, crypto.FIL, testCh)
	assert.NotNil(t, err)

	time.Sleep(5 * time.Second)
	err = transactor.Collect(ctx, crypto.FIL, testCh)
	assert.Nil(t, err)
}
