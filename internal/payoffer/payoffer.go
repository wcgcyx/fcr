package payoffer

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/wcgcyx/fcr/internal/crypto"
)

const (
	randLength = 32
)

// PayOffer represents an offer for a proxy payment request.
type PayOffer struct {
	// ID is the id of this offer.
	id cid.Cid

	// Currency ID is the currency of this payment offer.
	currencyID uint64

	// From addr is the source address of this proxy payment.
	fromAddr string

	// To addr is the destination address of this proxy payment.
	toAddr string

	// PPP stands for payment-per-period, is the surchage of this proxy payment.
	ppp *big.Int

	// Period is the payment interval.
	period *big.Int

	// Max amt is the maximum amount of payment this offer covers.
	maxAmt *big.Int

	// Max inactivity specifies the max inactivity time of this offer before it expires.
	maxInactivity time.Duration

	// Created at specifies the unix time in seconds when this offer is created.
	createdAt int64

	// Expiration specifies when this offer will be expired.
	expiration int64

	// Linked offer is the linked offer of this offer.
	// For example, for a payment route of A -> B -> C -> D.
	// When A queries B for an offer, B will query C for offer C -> D and after
	// B received the this offer, B will generate an offer B -> D, since B -> D
	// depends on C -> D, the cid of offer C -> D will be attached here.
	linkedOffer cid.Cid

	// randBytes is used to differentiate offers. 32 bytes long.
	randBytes []byte

	// Signature is the signature by fromAddr.
	signature []byte
}

// payOfferJson is used for serialization.
type payOfferJson struct {
	CurrencyID    uint64 `json:"currency_id"`
	FromAddr      string `json:"from_addr"`
	ToAddr        string `json:"to_addr"`
	PPP           string `json:"ppp"`
	Period        string `json:"period"`
	MaxAmt        string `json:"max_amt"`
	MaxInactivity string `json:"max_inactivity"`
	CreatedAt     int64  `json:"created_at"`
	Expiration    int64  `json:"expiration"`
	LinkedOffer   string `json:"linked_offer"`
	RandBytes     string `json:"rand_bytes"`
	Signature     string `json:"signature"`
}

// NewPayOffer is used to create and sign a new pay offer.
// It takes a private key, currency id, to address, price per period, period, max amount, max inactivity time, expiration, linked offer cid as arguments.
// It returns the pay offer and error.
func NewPayOffer(prv []byte, currencyID uint64, toAddr string, ppp *big.Int, period *big.Int, maxAmt *big.Int, maxInactivity time.Duration, expiration int64, linkedOffer cid.Cid) (*PayOffer, error) {
	fromAddr, err := crypto.GetAddress(prv)
	if err != nil {
		return nil, err
	}
	randBytes := make([]byte, randLength)
	rand.Read(randBytes)
	offer := &PayOffer{
		currencyID:    currencyID,
		fromAddr:      fromAddr,
		toAddr:        toAddr,
		ppp:           ppp,
		period:        period,
		maxAmt:        maxAmt,
		maxInactivity: maxInactivity,
		createdAt:     time.Now().Unix(),
		expiration:    expiration,
		linkedOffer:   linkedOffer,
		randBytes:     randBytes,
		signature:     []byte{},
	}
	// Sign offer.
	data := offer.ToBytes()
	sig, err := crypto.Sign(prv, data)
	if err != nil {
		return nil, err
	}
	offer.signature = sig
	// Set ID.
	data = offer.ToBytes()
	pref := cid.Prefix{
		Version:  1,
		Codec:    cid.DagProtobuf,
		MhType:   multihash.SHA2_256,
		MhLength: -1,
	}
	id, err := pref.Sum(data)
	if err != nil {
		return nil, err
	}
	offer.id = id
	return offer, nil
}

// ID is used to obtain the id of this pay offer.
// It returns the cid of this offer.
func (offer *PayOffer) ID() cid.Cid {
	return offer.id
}

// CurrencyID is used to obtain the currency of this pay offer.
// It returns the currency id of this offer.
func (offer *PayOffer) CurrencyID() uint64 {
	return offer.currencyID
}

// From is used to obtain the from address of this pay offer.
// It returns the from address of this offer.
func (offer *PayOffer) From() string {
	return offer.fromAddr
}

// To is used to obtain the to address of this pay offer.
// It returns the to address of this offer.
func (offer *PayOffer) To() string {
	return offer.toAddr
}

// PPP is used to obtain the price per period of this pay offer.
// It returns the price per period of this offer.
func (offer *PayOffer) PPP() *big.Int {
	return big.NewInt(0).Set(offer.ppp)
}

// Period is used to obtain the period of this pay offer.
// It returns the period of this offer.
func (offer *PayOffer) Period() *big.Int {
	return big.NewInt(0).Set(offer.period)
}

// MaxAmt is used to obtain the max amount of this pay offer.
// It returns the max amount of this offer.
func (offer *PayOffer) MaxAmt() *big.Int {
	return big.NewInt(0).Set(offer.maxAmt)
}

// Expiration is used to obtain the expiration of thie pay offer.
// It returns the expiration of this offer.
func (offer *PayOffer) MaxInactivity() time.Duration {
	return offer.maxInactivity
}

// CreatedAt is used to obtain the created time of this pay offer.
// It returns the creation time of this offer.
func (offer *PayOffer) CreatedAt() int64 {
	return offer.createdAt
}

// Expiration is used to obtain the expiration time of this pay offer.
// It returns the expiration time of this offer.
func (offer *PayOffer) Expiration() int64 {
	return offer.expiration
}

// LinkedOffer is used to obatin the linked offer id of this pay offer.
// It returns the linked offer id of this offer.
func (offer *PayOffer) LinkedOffer() cid.Cid {
	return offer.linkedOffer
}

// Verify is used to verify this pay offer against the from address.
// It returns the error in verification.
func (offer *PayOffer) Verify() error {
	sig := offer.signature
	offer.signature = []byte{}
	data := offer.ToBytes()
	// Restore signature.
	offer.signature = sig
	return crypto.Verify(offer.fromAddr, sig, data)
}

// ToBytes is used to serialize this pay offer.
// It returns the bytes serialized from this offer.
func (offer *PayOffer) ToBytes() []byte {
	linkedOffer := ""
	if offer.linkedOffer != cid.Undef {
		linkedOffer = offer.linkedOffer.String()
	}
	data, _ := json.Marshal(payOfferJson{
		CurrencyID:    offer.currencyID,
		FromAddr:      offer.fromAddr,
		ToAddr:        offer.toAddr,
		PPP:           offer.ppp.String(),
		Period:        offer.period.String(),
		MaxAmt:        offer.maxAmt.String(),
		MaxInactivity: offer.maxInactivity.String(),
		CreatedAt:     offer.createdAt,
		Expiration:    offer.expiration,
		LinkedOffer:   linkedOffer,
		RandBytes:     hex.EncodeToString(offer.randBytes),
		Signature:     hex.EncodeToString(offer.signature),
	})
	return data
}

// FromBytes is used to deserialize bytes into this pay offer.
// It takes a byte array as the argument.
// It returns the pay offer and the error in deserialization.
func FromBytes(data []byte) (*PayOffer, error) {
	offer := &PayOffer{}
	offerJson := payOfferJson{}
	err := json.Unmarshal(data, &offerJson)
	if err != nil {
		return nil, err
	}
	ppp, ok := big.NewInt(0).SetString(offerJson.PPP, 10)
	if !ok {
		return nil, fmt.Errorf("fail to set price per period")
	}
	period, ok := big.NewInt(0).SetString(offerJson.Period, 10)
	if !ok {
		return nil, fmt.Errorf("fail to set period")
	}
	maxAmt, ok := big.NewInt(0).SetString(offerJson.MaxAmt, 10)
	if !ok {
		return nil, fmt.Errorf("fail to set max amount")
	}
	linkedOffer := cid.Undef
	if offerJson.LinkedOffer != "" {
		linkedOffer, err = cid.Parse(offerJson.LinkedOffer)
		if err != nil {
			return nil, err
		}
	}
	randBytes, err := hex.DecodeString(offerJson.RandBytes)
	if err != nil {
		return nil, err
	}
	randBytesPadded := make([]byte, randLength)
	copy(randBytesPadded, randBytes)
	signature, err := hex.DecodeString(offerJson.Signature)
	if err != nil {
		return nil, err
	}
	maxInactivity, err := time.ParseDuration(offerJson.MaxInactivity)
	if err != nil {
		return nil, err
	}
	offer.currencyID = offerJson.CurrencyID
	offer.fromAddr = offerJson.FromAddr
	offer.toAddr = offerJson.ToAddr
	offer.ppp = ppp
	offer.period = period
	offer.maxAmt = maxAmt
	offer.maxInactivity = maxInactivity
	offer.createdAt = offerJson.CreatedAt
	offer.expiration = offerJson.Expiration
	offer.linkedOffer = linkedOffer
	offer.randBytes = randBytesPadded
	offer.signature = signature
	// Set ID.
	pref := cid.Prefix{
		Version:  1,
		Codec:    cid.DagProtobuf,
		MhType:   multihash.SHA2_256,
		MhLength: -1,
	}
	id, err := pref.Sum(data)
	if err != nil {
		return nil, err
	}
	offer.id = id
	return offer, nil
}
