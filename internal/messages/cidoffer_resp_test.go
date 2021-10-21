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
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/internal/cidoffer"
)

func TestCIDOfferRespRoundTrip(t *testing.T) {
	testID, err := cid.Parse(testCID)
	assert.Empty(t, err)
	pid, err := peer.Decode(testPeerID)
	assert.Empty(t, err)
	maddr, err := multiaddr.NewMultiaddr(testIP)
	assert.Empty(t, err)
	prv, err := hex.DecodeString(testPrvString)
	assert.Empty(t, err)
	offer, err := cidoffer.NewCIDOffer(prv, 1, peer.AddrInfo{ID: pid, Addrs: []multiaddr.Multiaddr{maddr}}, testID, big.NewInt(20), 11223, "", nil, time.Now().Add(time.Hour).Unix())
	assert.Empty(t, err)
	data := EncodeCIDOfferResp(*offer)
	offerDecoded, err := DecodeCIDOfferResp(data)
	assert.Empty(t, err)
	assert.Equal(t, offer.ToBytes(), offerDecoded.ToBytes())
}
