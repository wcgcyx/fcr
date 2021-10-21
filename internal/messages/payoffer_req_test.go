package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPayOfferReqRoundTrip(t *testing.T) {
	testRoute := []string{"A", "B", "C"}
	data := EncodePayOfferReq(1, testRoute, big.NewInt(1000))
	currencyID, route, amt, err := DecodePayOfferReq(data)
	assert.Empty(t, err)
	assert.Equal(t, uint64(1), currencyID)
	assert.Equal(t, testRoute, route)
	assert.Equal(t, big.NewInt(1000), amt)
}
