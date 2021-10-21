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

func TestPayReqRoundTrip(t *testing.T) {
	prv, err := hex.DecodeString(testPrvString)
	assert.Empty(t, err)
	offer, err := payoffer.NewPayOffer(prv, 1, testAddr, big.NewInt(50), big.NewInt(1000), big.NewInt(500000), time.Minute, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	data := EncodePayReq(1, "testorigin", [32]byte{1, 2, 3, 4}, big.NewInt(10), testAddr, "testvoucher", offer)
	currencyID, originAddr, originSecret, originAmt, fromAddr, voucher, offerDecoded, err := DecodePayReq(data)
	assert.Empty(t, err)
	assert.Equal(t, uint64(1), currencyID)
	assert.Equal(t, "testorigin", originAddr)
	assert.Equal(t, [32]byte{1, 2, 3, 4}, originSecret)
	assert.Equal(t, big.NewInt(10), originAmt)
	assert.Equal(t, testAddr, fromAddr)
	assert.Equal(t, "testvoucher", voucher)
	assert.Equal(t, offer.ID(), offerDecoded.ID())
}
