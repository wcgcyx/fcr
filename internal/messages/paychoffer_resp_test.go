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

const (
	testPrvString = "cfbaf80c86fa46bf1f4ddbc85461bc6561b82f319eaea4fabf4a312a5f41c6bb"
)

func TestPaychOfferRespRoundTrip(t *testing.T) {
	prv, err := hex.DecodeString(testPrvString)
	assert.Empty(t, err)
	offer, err := paychoffer.NewPaychOffer(prv, 1, time.Hour, time.Minute)
	assert.Empty(t, err)
	data := EncodePaychOfferResp(*offer)
	offerDecoded, err := DecodePaychOfferResp(data)
	assert.Empty(t, err)
	assert.Equal(t, offer.ToBytes(), offerDecoded.ToBytes())
}
