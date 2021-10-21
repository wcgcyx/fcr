package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testAddr = "f1tynlzcntp3nmrpbbjjchq666zv6s6clr3pfdt3y"
)

func TestPaychOfferReqRoundTrip(t *testing.T) {
	data := EncodePaychOfferReq(1, testAddr)
	currencyID, addr, err := DecodePaychOfferReq(data)
	assert.Empty(t, err)
	assert.Equal(t, uint64(1), currencyID)
	assert.Equal(t, testAddr, addr)
}
