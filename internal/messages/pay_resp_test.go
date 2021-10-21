package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPayRespRoundTrip(t *testing.T) {
	testReceipt := [65]byte{1, 2, 3, 4}
	data := EncodePayResp(testReceipt[:])
	receipt, err := DecodePayResp(data)
	assert.Empty(t, err)
	assert.Equal(t, testReceipt, receipt)
}
