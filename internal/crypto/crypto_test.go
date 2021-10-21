package crypto

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testPrvString = "cfbaf80c86fa46bf1f4ddbc85461bc6561b82f319eaea4fabf4a312a5f41c6bb"
	testAddr      = "f1tynlzcntp3nmrpbbjjchq666zv6s6clr3pfdt3y"
	testData      = "283184bc1aa68af3cb7e5c7cd2bf2e2ea24df1e5b3dc3c720f55938971ee6543"
	testSig       = "10c01bc5d5b281d7f7d11a128e923be653b4b0f58db7141bbadccc2d6a0700f601090e26153d0da1922d10f718a4f28f457b55c3ad7be4b4f5ac50baa4d7530700"
)

func TestGenerateKeyPair(t *testing.T) {
	prv, addr, err := GenerateKeyPair()
	assert.Empty(t, err)
	assert.NotEmpty(t, prv)
	assert.NotEmpty(t, addr)
}

func TestRoundTrip(t *testing.T) {
	prv, _ := hex.DecodeString(testPrvString)
	addr, err := GetAddress(prv)
	assert.Empty(t, err)
	assert.Equal(t, testAddr, addr)

	data, _ := hex.DecodeString(testData)
	sig, err := Sign(prv, data)
	assert.Empty(t, err)
	assert.Equal(t, testSig, hex.EncodeToString(sig))

	err = Verify(addr, sig, data)
	assert.Empty(t, err)

	sig[0] += 1
	err = Verify(addr, sig, data)
	assert.NotEmpty(t, err)

	err = Verify(addr, sig[1:], data)
	assert.NotEmpty(t, err)
}
