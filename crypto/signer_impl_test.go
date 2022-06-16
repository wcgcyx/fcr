package crypto

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
	"encoding/hex"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	testDS = "./test-ds"
)

var testKey = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1}
var testAddr = "f1ssv6id634b4q4mecyqsfqdibj3vqdhn6piro4yq"
var testSig = "65a449d22888b59eae85da7e5a9b8f5367cd3166e35b8ec8b96c7b2aa0d423e476e7a1c9b029aa177d7f546119e83759b616ec8f2725af4845de21603d58549900"

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	m.Run()
}

func TestNewSignerImpl(t *testing.T) {
	ctx := context.Background()

	_, err := NewSignerImpl(ctx, Opts{})
	assert.NotNil(t, err)

	signer, err := NewSignerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer signer.Shutdown()
}

func TestSetKey(t *testing.T) {
	ctx := context.Background()
	signer, err := NewSignerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer signer.Shutdown()

	err = signer.SetKey(ctx, 0, SECP256K1, []byte{0})
	assert.NotNil(t, err)

	err = signer.SetKey(ctx, FIL, 0, []byte{0})
	assert.NotNil(t, err)

	err = signer.SetKey(ctx, FIL, SECP256K1, []byte{0})
	assert.NotNil(t, err)

	err = signer.SetKey(ctx, FIL, SECP256K1, []byte{1})
	assert.NotNil(t, err)

	err = signer.SetKey(ctx, FIL, SECP256K1, testKey)
	assert.Nil(t, err)

	err = signer.SetKey(ctx, FIL, SECP256K1, testKey)
	assert.NotNil(t, err)
}

func TestGetAddr(t *testing.T) {
	ctx := context.Background()
	signer, err := NewSignerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer signer.Shutdown()

	_, _, err = signer.GetAddr(ctx, 0)
	assert.NotNil(t, err)

	keyType, addr, err := signer.GetAddr(ctx, FIL)
	assert.Nil(t, err)
	assert.Equal(t, SECP256K1, keyType)
	assert.Equal(t, testAddr, addr)
}

func TestSign(t *testing.T) {
	ctx := context.Background()
	signer, err := NewSignerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer signer.Shutdown()

	_, _, err = signer.Sign(ctx, 0, []byte{0})
	assert.NotNil(t, err)

	keyType, sig, err := signer.Sign(ctx, FIL, []byte{0})
	assert.Nil(t, err)
	assert.Equal(t, SECP256K1, keyType)
	assert.Equal(t, testSig, hex.EncodeToString(sig))

	err = Verify(0, []byte{0}, SECP256K1, sig, testAddr)
	assert.NotNil(t, err)

	err = Verify(FIL, []byte{1}, SECP256K1, sig, testAddr)
	assert.NotNil(t, err)

	err = Verify(FIL, []byte{0}, 0, sig, testAddr)
	assert.NotNil(t, err)

	sig[0] += 1
	err = Verify(FIL, []byte{0}, SECP256K1, sig, testAddr)
	assert.NotNil(t, err)
	sig[0] -= 1

	testAddr = "a" + testAddr[1:]
	err = Verify(FIL, []byte{0}, SECP256K1, sig, testAddr)
	assert.NotNil(t, err)
	testAddr = "f" + testAddr[1:]

	err = Verify(FIL, []byte{0}, SECP256K1, sig, testAddr)
	assert.Nil(t, err)
}

func TestRetireKey(t *testing.T) {
	ctx := context.Background()
	signer, err := NewSignerImpl(ctx, Opts{Path: testDS})
	assert.Nil(t, err)
	defer signer.Shutdown()

	curChan, errChan := signer.ListCurrencyIDs(ctx)
	currencyIDs := make([]byte, 0)
	for cur := range curChan {
		currencyIDs = append(currencyIDs, cur)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(currencyIDs))

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		errChan := signer.RetireKey(ctx, 0, 2*time.Second)
		assert.NotNil(t, <-errChan)

		errChan = signer.RetireKey(ctx, FIL, 2*time.Second)
		assert.NotNil(t, <-errChan)

		errChan = signer.RetireKey(ctx, FIL, 2*time.Second)
		assert.NotNil(t, <-errChan)

		errChan = signer.RetireKey(ctx, FIL, 2*time.Second)
		assert.NotNil(t, <-errChan)

		errChan = signer.RetireKey(ctx, FIL, 2*time.Second)
		assert.NotNil(t, <-errChan)

		errChan = signer.RetireKey(ctx, FIL, 2*time.Second)
		assert.Nil(t, <-errChan)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		time.Sleep(100 * time.Millisecond)
		errChan := signer.RetireKey(ctx, FIL, time.Second)
		assert.NotNil(t, <-errChan)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		time.Sleep(150 * time.Millisecond)
		err = signer.SetKey(ctx, FIL, SECP256K1, testKey)
		assert.NotNil(t, err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		time.Sleep(200 * time.Millisecond)
		_, _, err = signer.GetAddr(ctx, FIL)
		assert.Nil(t, err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		time.Sleep(250 * time.Millisecond)
		_, _, err := signer.Sign(ctx, FIL, []byte{0})
		assert.Nil(t, err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		time.Sleep(300 * time.Millisecond)
		err = signer.StopRetire(ctx, FIL)
		assert.Nil(t, err)
	}()

	wg.Wait()

	curChan, errChan = signer.ListCurrencyIDs(ctx)
	currencyIDs = make([]byte, 0)
	for cur := range curChan {
		currencyIDs = append(currencyIDs, cur)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(currencyIDs))
}
