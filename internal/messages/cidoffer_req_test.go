package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
)

const (
	testCID = "bafk2bzacecwz4gbjugyrb3orrzqidyxjibvwxojjppbgbwdwkcfocscu2chk4"
)

func TestCIDOfferReqRoundTrip(t *testing.T) {
	testID, err := cid.Parse(testCID)
	assert.Empty(t, err)
	data := EncodeCIDOfferReq(1, testID)
	currencyID, id, err := DecodeCIDOfferReq(data)
	assert.Empty(t, err)
	assert.Equal(t, uint64(1), currencyID)
	assert.Equal(t, testID, id)
}
