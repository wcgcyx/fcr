package cidoffer

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
	"github.com/wcgcyx/fcr/internal/crypto"
)

const (
	testPrvString1 = "cfbaf80c86fa46bf1f4ddbc85461bc6561b82f319eaea4fabf4a312a5f41c6bb"
	testPrvString2 = "cfbaf80c86fa46bf1f4ddbc85461bc6561b82f319eaea4fabf4a312a5f41c6bc"
	testPeerID     = "QmXm6tWTUnaa5aep6vAwYkEAMpn8C5YFMgX9bh9BRW8Vi1"
	testIP         = "/ip4/127.0.0.1/tcp/9010"
	testRoot       = "bafybeid25rwr6dhwyztbn74cjq5kjbx3mj352qfzyxwzmjf6dmtmhmh6iq"
)

func TestNewCIDOffer(t *testing.T) {
	prv, _ := hex.DecodeString(testPrvString1)
	pid, err := peer.Decode(testPeerID)
	assert.Empty(t, err)
	maddr, err := multiaddr.NewMultiaddr(testIP)
	assert.Empty(t, err)
	root, err := cid.Parse(testRoot)
	assert.Empty(t, err)

	offer, err := NewCIDOffer(prv, 1, peer.AddrInfo{ID: pid, Addrs: []multiaddr.Multiaddr{maddr}}, root, big.NewInt(20), 11223, "", nil, time.Now().Add(time.Hour).Unix())
	assert.Empty(t, err)
	assert.NotEmpty(t, offer)

	assert.Equal(t, peer.AddrInfo{
		ID:    pid,
		Addrs: []multiaddr.Multiaddr{maddr}}, offer.PeerAddr())
	assert.Equal(t, testRoot, offer.Root().String())
	assert.Equal(t, uint64(1), offer.CurrencyID())
	assert.Equal(t, big.NewInt(20), offer.PPB())
	assert.Equal(t, "f1tynlzcntp3nmrpbbjjchq666zv6s6clr3pfdt3y", offer.ToAddr())
	assert.Equal(t, int64(11223), offer.Size())
	assert.Equal(t, "", offer.LinkedMiner())
	assert.Empty(t, offer.Verify())
	assert.NotEmpty(t, offer.Expiration())

	offer.signature[0] += 1
	assert.NotEmpty(t, offer.Verify())
}

func TestSerialization(t *testing.T) {
	prv1, _ := hex.DecodeString(testPrvString1)
	prv2, _ := hex.DecodeString(testPrvString2)
	pid, err := peer.Decode(testPeerID)
	assert.Empty(t, err)
	maddr, err := multiaddr.NewMultiaddr(testIP)
	assert.Empty(t, err)
	root, err := cid.Parse(testRoot)
	assert.Empty(t, err)
	addr1, err := crypto.GetAddress(prv1)
	assert.Empty(t, err)
	addr2, err := crypto.GetAddress(prv2)
	assert.Empty(t, err)

	sig, err := crypto.Sign(prv2, []byte(addr1))
	assert.Empty(t, err)

	offer, err := NewCIDOffer(prv1, 1, peer.AddrInfo{ID: pid, Addrs: []multiaddr.Multiaddr{maddr}}, root, big.NewInt(20), 11223, addr2, sig, time.Now().Add(time.Hour).Unix())
	assert.Empty(t, err)
	assert.NotEmpty(t, offer)
	assert.Empty(t, offer.Verify())

	data := offer.ToBytes()
	newOffer, err := FromBytes(data)
	assert.Empty(t, err)
	assert.Equal(t, offer.PeerAddr(), newOffer.PeerAddr())
	assert.Equal(t, offer.Root(), newOffer.Root())
	assert.Equal(t, offer.CurrencyID(), newOffer.CurrencyID())
	assert.Equal(t, offer.PPB(), newOffer.PPB())
	assert.Equal(t, offer.Size(), newOffer.Size())
	assert.Equal(t, offer.ToAddr(), newOffer.ToAddr())
	assert.Equal(t, offer.LinkedMiner(), newOffer.LinkedMiner())
	assert.Equal(t, offer.Expiration(), newOffer.Expiration())
	assert.Empty(t, newOffer.Verify())
	assert.Equal(t, data, newOffer.ToBytes())
}
