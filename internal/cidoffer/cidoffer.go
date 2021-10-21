package cidoffer

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wcgcyx/fcr/internal/crypto"
)

// CIDOffer represents an offer for an ipld graph.
type CIDOffer struct {
	// PeerAddr is the p2p address of the offer creator.
	peerAddr peer.AddrInfo

	// Root is the cid of the root node of an ipld graph.
	root cid.Cid

	// CurrencyID is the currency this offer accepted.
	currencyID uint64

	// PPB stands for price-per-byte, the prices of the graph.
	ppb *big.Int

	// Size is the size of this graph.
	size int64

	// ToAddr is the payment address of this offer.
	toAddr string

	// LinkedMinerAddr is the miner address that links to this peer.
	linkedMinerAddr string

	// LinkedMinerSignature is a signature across the toAddr.
	linkedMinerSignature []byte

	// Expiration specifies when this offer will be expired.
	expiration int64

	// Signature is the signatures by the recipient address.
	signature []byte
}

// cidOfferJson is used for serialization.
type cidOfferJson struct {
	PeerAddr             string `json:"peer_addr"`
	Root                 string `json:"root"`
	CurrencyID           uint64 `json:"currency_id"`
	PPB                  string `json:"ppb"`
	Size                 int64  `json:"size"`
	ToAddr               string `json:"to_addr"`
	LinkedMinerAddr      string `json:"linked_miner_addr"`
	LinkedMinerSignature string `json:"linked_miner_signature"`
	Expiration           int64  `json:"expiration"`
	Signature            string `json:"signature"`
}

// NewCIDOffer is used to create and sign a new cid offer.
// It takes a private key, a currency id, peer addr info, root cid, price per byte, size, linked miner address, linked miner signature, expiration as arguments.
// It returns the cid offer and error.
func NewCIDOffer(prv []byte, currencyID uint64, peerAddr peer.AddrInfo, root cid.Cid, ppb *big.Int, size int64, linkedMinerAddr string, linkedMinerSignature []byte, expiration int64) (*CIDOffer, error) {
	toAddr, err := crypto.GetAddress(prv)
	if err != nil {
		return nil, err
	}
	offer := &CIDOffer{
		peerAddr:             peerAddr,
		root:                 root,
		currencyID:           currencyID,
		ppb:                  big.NewInt(0).Set(ppb),
		size:                 size,
		toAddr:               toAddr,
		linkedMinerAddr:      linkedMinerAddr,
		linkedMinerSignature: linkedMinerSignature,
		expiration:           expiration,
	}
	// Sign offer.
	data := offer.ToBytes()
	sig, err := crypto.Sign(prv, data)
	if err != nil {
		return nil, err
	}
	offer.signature = sig
	return offer, nil
}

// PeerAddr is used to obtain the peer addr info of this cid offer.
// It returns the peer addr of this offer.
func (offer *CIDOffer) PeerAddr() peer.AddrInfo {
	return offer.peerAddr
}

// Root is used to obtain the root of this cid offer.
// It returns the root cid of this offer.
func (offer *CIDOffer) Root() cid.Cid {
	return offer.root
}

// CurrencyID is used to obtain the currency id of this cid offer.
// It returns the currency id of this offer.
func (offer *CIDOffer) CurrencyID() uint64 {
	return offer.currencyID
}

// PPB is used to obtain the price per byte of this cid offer.
// It returns the price per byte of this offer.
func (offer *CIDOffer) PPB() *big.Int {
	return big.NewInt(0).Set(offer.ppb)
}

// Size is used to obtain the size of the ipld graph contained in this cid offer.
// It returns the size of the ipld graph of this offer.
func (offer *CIDOffer) Size() int64 {
	return offer.size
}

// ToAddr is used to obtain the to address of this cid offer.
// It returns the to address of this offer.
func (offer *CIDOffer) ToAddr() string {
	return offer.toAddr
}

// LinkedMiner is used to obtain the linked miner address of this cid offer.
// It returns the linked miner address of this offer.
func (offer *CIDOffer) LinkedMiner() string {
	return offer.linkedMinerAddr
}

// Expiration is used to obtain the expiration time of this cid offer.
// It returns the expiration time of this offer.
func (offer *CIDOffer) Expiration() int64 {
	return offer.expiration
}

// Verify is used to verify this cid offer against the to addresses and the miner signature.
// It returns the error in verification.
func (offer *CIDOffer) Verify() error {
	// First verify the offer signatures.
	sig := offer.signature
	offer.signature = []byte{}
	data := offer.ToBytes()
	// Restore signature.
	offer.signature = sig
	if err := crypto.Verify(offer.toAddr, sig, data); err != nil {
		return err
	}
	// Second verify miner signature
	if offer.linkedMinerAddr != "" {
		return crypto.Verify(offer.linkedMinerAddr, offer.linkedMinerSignature, []byte(offer.toAddr))
	}
	return nil
}

// ToBytes is used to serialize this cid offer.
// It returns the bytes serialized from this offer.
func (offer *CIDOffer) ToBytes() []byte {
	piData, _ := offer.peerAddr.MarshalJSON()
	linkedMinerSignature := ""
	if offer.linkedMinerAddr != "" {
		linkedMinerSignature = hex.EncodeToString(offer.linkedMinerSignature)
	}
	data, _ := json.Marshal(cidOfferJson{
		PeerAddr:             hex.EncodeToString(piData),
		Root:                 offer.root.String(),
		CurrencyID:           offer.currencyID,
		PPB:                  offer.ppb.String(),
		Size:                 offer.size,
		ToAddr:               offer.toAddr,
		LinkedMinerAddr:      offer.linkedMinerAddr,
		LinkedMinerSignature: linkedMinerSignature,
		Expiration:           offer.expiration,
		Signature:            hex.EncodeToString(offer.signature),
	})
	return data
}

// FromBytes is used to deserialize bytes into this cid offer.
// It takes a byte array as the argument.
// It returns the cid offer and the error in deserialization.
func FromBytes(data []byte) (*CIDOffer, error) {
	offer := &CIDOffer{}
	offerJson := cidOfferJson{}
	err := json.Unmarshal(data, &offerJson)
	if err != nil {
		return nil, err
	}
	piData, err := hex.DecodeString(offerJson.PeerAddr)
	if err != nil {
		return nil, err
	}
	pi := &peer.AddrInfo{}
	err = pi.UnmarshalJSON(piData)
	if err != nil {
		return nil, err
	}
	root, err := cid.Parse(offerJson.Root)
	if err != nil {
		return nil, err
	}
	ppb, ok := big.NewInt(0).SetString(offerJson.PPB, 10)
	if !ok {
		return nil, fmt.Errorf("fail to set price per byte")
	}
	var linkedMinerSignature []byte
	if offerJson.LinkedMinerAddr != "" {
		linkedMinerSignature, err = hex.DecodeString(offerJson.LinkedMinerSignature)
		if err != nil {
			return nil, err
		}
	}
	sig, err := hex.DecodeString(offerJson.Signature)
	if err != nil {
		return nil, err
	}
	offer.peerAddr = *pi
	offer.root = root
	offer.currencyID = offerJson.CurrencyID
	offer.ppb = ppb
	offer.size = offerJson.Size
	offer.toAddr = offerJson.ToAddr
	offer.linkedMinerAddr = offerJson.LinkedMinerAddr
	offer.linkedMinerSignature = linkedMinerSignature
	offer.expiration = offerJson.Expiration
	offer.signature = sig
	return offer, nil
}
