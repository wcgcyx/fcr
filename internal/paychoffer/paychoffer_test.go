package paychoffer

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	testPrvString = "cfbaf80c86fa46bf1f4ddbc85461bc6561b82f319eaea4fabf4a312a5f41c6bb"
	testAddr      = "f1tynlzcntp3nmrpbbjjchq666zv6s6clr3pfdt3y"
)

func TestNewPaychOffer(t *testing.T) {
	prv, _ := hex.DecodeString(testPrvString)
	offer, err := NewPaychOffer(prv, 1, 5*time.Minute, time.Hour)
	assert.Empty(t, err)
	assert.NotEmpty(t, offer)

	assert.Equal(t, uint64(1), offer.CurrencyID())
	assert.Equal(t, testAddr, offer.Addr())
	assert.Equal(t, true, offer.Settlement() > time.Now().Unix())
	assert.NotEmpty(t, offer.Expiration())
	assert.Empty(t, offer.Verify())

	offer.signature[0] += 1
	assert.NotEmpty(t, offer.Verify())
}

func TestSerialization(t *testing.T) {
	prv, _ := hex.DecodeString(testPrvString)
	offer, err := NewPaychOffer(prv, 1, 5*time.Minute, time.Hour)
	assert.Empty(t, err)
	assert.NotEmpty(t, offer)

	data := offer.ToBytes()
	newOffer, err := FromBytes(data)
	assert.Empty(t, err)
	assert.Equal(t, offer.currencyID, newOffer.currencyID)
	assert.Equal(t, offer.addr, newOffer.addr)
	assert.Equal(t, offer.settlement, newOffer.settlement)
	assert.Equal(t, offer.expiration, newOffer.expiration)
	assert.Equal(t, offer.signature, newOffer.signature)
}
