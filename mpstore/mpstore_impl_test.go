package mpstore

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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/crypto"
)

const (
	testDS = "./test-ds"
)

var testKey = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1}
var testAddr = "f1ssv6id634b4q4mecyqsfqdibj3vqdhn6piro4yq"
var testMinerKey1 = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 2}
var testMinerAddr1 = "f1cdmmfnqhlkgbrbnp6kx6fia5my2a7t66s6wzq2y"
var correctProof1 = []byte{134, 209, 163, 9, 141, 112, 81, 118, 217, 186, 230, 157, 199, 156, 48, 245, 25, 224, 222, 6, 17, 36, 226, 32, 105, 174, 109, 49, 84, 200, 115, 188, 27, 81, 179, 161, 188, 31, 109, 167, 56, 101, 131, 159, 68, 145, 233, 88, 189, 38, 197, 33, 58, 203, 106, 220, 89, 105, 78, 61, 244, 110, 95, 79, 1}
var testMinerKey2 = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 3}
var testMinerAddr2 = "f1m2vpd7ylsyogtqehlge2afw3aa3pcamxxjjqr3a"
var correctProof2 = []byte{139, 248, 97, 109, 74, 248, 213, 29, 73, 44, 214, 57, 40, 176, 223, 143, 115, 7, 168, 72, 5, 190, 102, 173, 118, 145, 79, 166, 152, 93, 43, 95, 26, 35, 152, 73, 240, 87, 180, 171, 41, 145, 218, 23, 118, 27, 59, 197, 193, 181, 208, 216, 247, 239, 227, 238, 20, 41, 23, 45, 51, 219, 44, 117, 1}

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	m.Run()
}

func TestNewMinerProofStoreImpl(t *testing.T) {
	ctx := context.Background()

	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: testDS + "/signer"})
	assert.Nil(t, err)
	defer signer.Shutdown()

	err = signer.SetKey(ctx, crypto.FIL, crypto.SECP256K1, testKey)
	assert.Nil(t, err)

	_, err = NewMinerProofStoreImpl(ctx, signer, Opts{})
	assert.NotNil(t, err)

	mpStore, err := NewMinerProofStoreImpl(ctx, signer, Opts{Path: testDS})
	assert.Nil(t, err)
	defer mpStore.Shutdown()
}

func TestStoreProof(t *testing.T) {
	ctx := context.Background()

	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: testDS + "/signer"})
	assert.Nil(t, err)
	defer signer.Shutdown()

	mpStore, err := NewMinerProofStoreImpl(ctx, signer, Opts{Path: testDS})
	assert.Nil(t, err)
	defer mpStore.Shutdown()

	exists, _, _, _, err := mpStore.GetMinerProof(ctx, crypto.FIL)
	assert.Nil(t, err)
	assert.False(t, exists)

	err = mpStore.UpsertMinerProof(ctx, crypto.FIL, crypto.SECP256K1, testMinerAddr1, append([]byte{correctProof1[0] + 1}, correctProof1[1:]...))
	assert.NotNil(t, err)

	err = mpStore.UpsertMinerProof(ctx, crypto.FIL, crypto.SECP256K1, testMinerAddr1, correctProof1)
	assert.Nil(t, err)

	exists, keyType, minerAddr, proof, err := mpStore.GetMinerProof(ctx, crypto.FIL)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, crypto.SECP256K1, keyType)
	assert.Equal(t, testMinerAddr1, minerAddr)
	assert.Equal(t, correctProof1, proof)

	err = mpStore.UpsertMinerProof(ctx, crypto.FIL, crypto.SECP256K1, testMinerAddr2, correctProof2)
	assert.Nil(t, err)

	exists, keyType, minerAddr, proof, err = mpStore.GetMinerProof(ctx, crypto.FIL)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, crypto.SECP256K1, keyType)
	assert.Equal(t, testMinerAddr2, minerAddr)
	assert.Equal(t, correctProof2, proof)
}
