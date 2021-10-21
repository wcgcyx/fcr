package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/internal/paychoffer"
)

func TestPaychRegisterRoundTrip(t *testing.T) {
	prv, err := hex.DecodeString(testPrvString)
	assert.Empty(t, err)
	offer, err := paychoffer.NewPaychOffer(prv, 1, time.Hour, time.Minute)
	assert.Empty(t, err)
	data := EncodePaychRegister(testAddr, offer)
	addr, offerDecoded, err := DecodePaychRegister(data)
	assert.Empty(t, err)
	assert.Equal(t, testAddr, addr)
	assert.Equal(t, offer.ToBytes(), offerDecoded.ToBytes())
}
