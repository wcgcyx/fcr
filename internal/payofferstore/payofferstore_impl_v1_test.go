package payofferstore

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"encoding/hex"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/internal/payoffer"
)

const (
	testDB          = "./testdb"
	testPrvString   = "cfbaf80c86fa46bf1f4ddbc85461bc6561b82f319eaea4fabf4a312a5f41c6bb"
	testFrom        = "f1tynlzcntp3nmrpbbjjchq666zv6s6clr3pfdt3y"
	testTo          = "f1ksokt4dwvgsscpzaxh2q4vcaoifxd4hswdhid5a"
	testLinkedOffer = "bafybeid25rwr6dhwyztbn74cjq5kjbx3mj352qfzyxwzmjf6dmtmhmh6iq"
)

func TestNewPayOfferStore(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	os, err := NewPayOfferStoreImplV1(ctx, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, os)

	prv, _ := hex.DecodeString(testPrvString)
	offer, err := payoffer.NewPayOffer(prv, 1, testTo, big.NewInt(10), big.NewInt(1000), big.NewInt(2000), time.Hour, time.Now().Add(time.Second).Unix(), cid.Undef)
	assert.Empty(t, err)
	assert.NotEmpty(t, offer)
	time.Sleep(2 * time.Second)
	err = os.AddOffer(offer)
	assert.NotEmpty(t, err)

	offer, err = payoffer.NewPayOffer(prv, 1, testTo, big.NewInt(10), big.NewInt(1000), big.NewInt(2000), time.Hour, time.Now().Add(time.Hour).Unix(), cid.Undef)
	assert.Empty(t, err)
	assert.NotEmpty(t, offer)

	err = os.AddOffer(offer)
	assert.Empty(t, err)

	id, err := cid.Parse(testLinkedOffer)
	assert.Empty(t, err)
	_, err = os.GetOffer(id)
	assert.NotEmpty(t, err)

	offerLoad, err := os.GetOffer(offer.ID())
	assert.Empty(t, err)
	assert.Equal(t, offer.ID(), offerLoad.ID())

	cancel()
	time.Sleep(1 * time.Second)
}
