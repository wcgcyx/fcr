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

	"github.com/stretchr/testify/assert"
)

func TestPayOfferRoundTrip(t *testing.T) {
	offer1 := PayOffer{
		CurrencyID: 1,
		SrcAddr:    "test-src-addr",
		DestAddr:   "test-dest-addr",
		PPP:        big.NewInt(10),
		Period:     big.NewInt(100),
		Amt:        big.NewInt(300),
		Expiration: time.Now().Add(time.Minute),
		Inactivity: 30 * time.Second,
		ResTo:      "test-res-to",
		ResCh:      "test-res-ch",
		ResID:      2,
		Nonce:      1,
	}
	// Mock sign.
	toSign, err := offer1.GetToBeSigned()
	assert.Nil(t, err)
	sig1 := sha256.Sum256(toSign)
	offer1.AddSignature(1, sig1[:])
	data, err := offer1.Encode()
	assert.Nil(t, err)
	offer2 := PayOffer{}
	err = offer2.Decode(data)
	assert.Nil(t, err)
	assert.Equal(t, offer1.CurrencyID, offer2.CurrencyID)
	assert.Equal(t, offer1.SrcAddr, offer2.SrcAddr)
	assert.Equal(t, offer1.DestAddr, offer2.DestAddr)
	assert.Equal(t, offer1.PPP, offer2.PPP)
	assert.Equal(t, offer1.Period, offer2.Period)
	assert.Equal(t, offer1.Amt, offer2.Amt)
	assert.True(t, offer1.Expiration.Equal(offer2.Expiration))
	assert.Equal(t, offer1.Inactivity, offer2.Inactivity)
	assert.Equal(t, offer1.ResTo, offer2.ResTo)
	assert.Equal(t, offer1.ResCh, offer2.ResCh)
	assert.Equal(t, offer1.ResID, offer2.ResID)
	assert.Equal(t, offer1.Nonce, offer2.Nonce)
	assert.Equal(t, offer1.SignatureType, offer2.SignatureType)
	assert.Equal(t, offer1.Signature, offer2.Signature)
	// Mock verify.
	toSign, err = offer2.GetToBeSigned()
	assert.Nil(t, err)
	sig2 := sha256.Sum256(toSign)
	assert.Equal(t, offer2.Signature, sig2[:])
}
