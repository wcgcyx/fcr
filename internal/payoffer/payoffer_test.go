package payoffer

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
)

const (
	testPrvString   = "cfbaf80c86fa46bf1f4ddbc85461bc6561b82f319eaea4fabf4a312a5f41c6bb"
	testFrom        = "f1tynlzcntp3nmrpbbjjchq666zv6s6clr3pfdt3y"
	testTo          = "f1ksokt4dwvgsscpzaxh2q4vcaoifxd4hswdhid5a"
	testLinkedOffer = "bafybeid25rwr6dhwyztbn74cjq5kjbx3mj352qfzyxwzmjf6dmtmhmh6iq"
)

func TestNewPayOffer(t *testing.T) {
	prv, _ := hex.DecodeString(testPrvString)
	offer, err := NewPayOffer(prv, 1, testTo, big.NewInt(10), big.NewInt(1000), big.NewInt(2000), time.Hour, time.Now().Add(time.Hour).Unix(), cid.Undef)
	assert.Empty(t, err)
	assert.NotEmpty(t, offer)

	assert.NotEmpty(t, offer.ID())
	assert.Equal(t, uint64(1), offer.CurrencyID())
	assert.Equal(t, testFrom, offer.From())
	assert.Equal(t, testTo, offer.To())
	assert.Equal(t, "10", offer.PPP().String())
	assert.Equal(t, "1000", offer.Period().String())
	assert.Equal(t, "2000", offer.MaxAmt().String())
	assert.Equal(t, time.Hour, offer.MaxInactivity())
	assert.NotEmpty(t, offer.CreatedAt())
	assert.NotEmpty(t, offer.Expiration())
	assert.Equal(t, "b", offer.LinkedOffer().String())
	assert.Empty(t, offer.Verify())

	offer.signature[0] += 1
	assert.NotEmpty(t, offer.Verify())
}

func TestSerialization(t *testing.T) {
	prv, _ := hex.DecodeString(testPrvString)
	offer, err := NewPayOffer(prv, 1, testTo, big.NewInt(10), big.NewInt(1000), big.NewInt(2000), time.Hour, time.Now().Add(time.Hour).Unix(), cid.Undef)
	assert.Empty(t, err)
	assert.NotEmpty(t, offer)

	id := offer.ID()
	data := offer.ToBytes()
	newOffer, err := FromBytes(data)
	assert.Empty(t, err)
	assert.Equal(t, id, newOffer.ID())

	linkedOffer, _ := cid.Parse(testLinkedOffer)
	offer, err = NewPayOffer(prv, 1, testTo, big.NewInt(10), big.NewInt(1000), big.NewInt(2000), time.Hour, time.Now().Add(time.Hour).Unix(), linkedOffer)
	assert.Empty(t, err)
	assert.NotEmpty(t, offer)

	id = offer.ID()
	data = offer.ToBytes()
	newOffer, err = FromBytes(data)
	assert.Empty(t, err)
	assert.Equal(t, id, newOffer.ID())
}
