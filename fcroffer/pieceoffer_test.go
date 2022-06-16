package fcroffer

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
	"crypto/sha256"
	"math/big"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
)

const testCID = "bafkqafkunbuxgidjomqgcidumvzxiidgnfwgkibr"

func TestPieceOfferRoundTrip(t *testing.T) {
	id, err := cid.Parse(testCID)
	assert.Nil(t, err)
	offer1 := PieceOffer{
		ID:                 id,
		Size:               100,
		CurrencyID:         1,
		PPB:                big.NewInt(100),
		RecipientAddr:      "test-to",
		LinkedMinerKeyType: 1,
		LinkedMinerAddr:    "test-miner",
		LinkedMinerProof:   []byte{1, 2, 3},
		Expiration:         time.Now().Add(time.Minute),
		Inactivity:         30 * time.Second,
		Nonce:              1,
	}
	// Mock sign
	toSign, err := offer1.GetToBeSigned()
	assert.Nil(t, err)
	sig1 := sha256.Sum256(toSign)
	offer1.AddSignature(1, sig1[:])
	data, err := offer1.Encode()
	assert.Nil(t, err)
	offer2 := PieceOffer{}
	err = offer2.Decode(data)
	assert.Nil(t, err)
	assert.Equal(t, offer1.ID, offer2.ID)
	assert.Equal(t, offer1.Size, offer2.Size)
	assert.Equal(t, offer1.CurrencyID, offer2.CurrencyID)
	assert.Equal(t, offer1.PPB, offer2.PPB)
	assert.Equal(t, offer1.RecipientAddr, offer2.RecipientAddr)
	assert.Equal(t, offer1.LinkedMinerKeyType, offer2.LinkedMinerKeyType)
	assert.Equal(t, offer1.LinkedMinerAddr, offer2.LinkedMinerAddr)
	assert.Equal(t, offer1.LinkedMinerProof, offer2.LinkedMinerProof)
	assert.True(t, offer1.Expiration.Equal(offer2.Expiration))
	assert.Equal(t, offer1.Inactivity, offer2.Inactivity)
	assert.Equal(t, offer1.Nonce, offer2.Nonce)
	assert.Equal(t, offer1.SignatureType, offer2.SignatureType)
	assert.Equal(t, offer1.Signature, offer2.Signature)
	// Mock verify.
	toSign, err = offer2.GetToBeSigned()
	assert.Nil(t, err)
	sig2 := sha256.Sum256(toSign)
	assert.Equal(t, offer2.Signature, sig2[:])
}
