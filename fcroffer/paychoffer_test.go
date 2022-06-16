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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPaychOfferRoundTrip(t *testing.T) {
	offer1 := PaychOffer{
		CurrencyID: 1,
		FromAddr:   "test-src-addr",
		ToAddr:     "test-dest-addr",
		Settlement: time.Now().Add(time.Hour),
		Expiration: time.Now().Add(time.Minute),
		Nonce:      1,
	}
	// Mock sign.
	toSign, err := offer1.GetToBeSigned()
	assert.Nil(t, err)
	sig1 := sha256.Sum256(toSign)
	offer1.AddSignature(1, sig1[:])
	data, err := offer1.Encode()
	assert.Nil(t, err)
	offer2 := PaychOffer{}
	err = offer2.Decode(data)
	assert.Nil(t, err)
	assert.Equal(t, offer1.CurrencyID, offer2.CurrencyID)
	assert.Equal(t, offer1.FromAddr, offer2.FromAddr)
	assert.Equal(t, offer1.ToAddr, offer2.ToAddr)
	assert.True(t, offer1.Settlement.Equal(offer2.Settlement))
	assert.True(t, offer1.Expiration.Equal(offer2.Expiration))
	assert.Equal(t, offer1.Nonce, offer2.Nonce)
	assert.Equal(t, offer1.SignatureType, offer2.SignatureType)
	assert.Equal(t, offer1.Signature, offer2.Signature)
	// Mock verify.
	toSign, err = offer2.GetToBeSigned()
	assert.Nil(t, err)
	sig2 := sha256.Sum256(toSign)
	assert.Equal(t, offer2.Signature, sig2[:])
}
