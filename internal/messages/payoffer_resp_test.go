package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/internal/payoffer"
)

func TestPayOfferRespRoundTrip(t *testing.T) {
	prv, err := hex.DecodeString(testPrvString)
	assert.Empty(t, err)
	offer, err := payoffer.NewPayOffer(prv, 1, testAddr, big.NewInt(50), big.NewInt(1000), big.NewInt(500000), time.Minute, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	data := EncodePayOfferResp(*offer)
	offerDecoded, err := DecodePayOfferResp(data)
	assert.Empty(t, err)
	assert.Equal(t, offer.ID(), offerDecoded.ID())
}
