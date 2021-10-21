package paychoffer

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/wcgcyx/fcr/internal/crypto"
)

// PaychOffer represents an offer for the time to keep an incoming paych alive.
type PaychOffer struct {
	// Currency ID is the currency of this paych offer.
	currencyID uint64

	// Addr is the address sending this offer, or the recipient of about to created paych.
	addr string

	// Settlement is the earliest time of settlement of this paych.
	settlement int64

	// Expiration specifies the expiration of this offer, it is in unix second.
	expiration int64

	// Signature is the signature by toAddr.
	signature []byte
}

// paychOfferJson is used for serialization.
type paychOfferJson struct {
	CurrencyID uint64 `json:"currency_id"`
	Addr       string `json:"addr"`
	Settlement int64  `json:"settlement"`
	Expiration int64  `json:"expiration"`
	Signature  string `json:"signature"`
}

// NewPaychOffer is used to create and sign a new paych offer.
// It takes a private key, currency id, time to settle, time to live as arguments.
// It returns the paych offer and error.
func NewPaychOffer(prv []byte, currencyID uint64, tts time.Duration, ttl time.Duration) (*PaychOffer, error) {
	addr, err := crypto.GetAddress(prv)
	if err != nil {
		return nil, err
	}
	offer := &PaychOffer{
		currencyID: currencyID,
		addr:       addr,
		settlement: time.Now().Add(tts).Unix(),
		expiration: time.Now().Add(ttl).Unix(),
		signature:  []byte{},
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

// CurrencyID is used to obtain the currency of this paych offer.
// It returns the currency id of this offer.
func (offer *PaychOffer) CurrencyID() uint64 {
	return offer.currencyID
}

// Addr is used to obtain the address of this paych offer.
// It returns the address of this offer.
func (offer *PaychOffer) Addr() string {
	return offer.addr
}

// Settlement is used to obtain the settlement time of this paych offer.
// It returns the settlement time of this offer.
func (offer *PaychOffer) Settlement() int64 {
	return offer.settlement
}

// Expiration is used to obtain the expiration of thie paych offer.
// It returns the expiration of this offer.
func (offer *PaychOffer) Expiration() int64 {
	return offer.expiration
}

// Verify is used to verify this paych offer against the from address.
// It returns the error in verification.
func (offer *PaychOffer) Verify() error {
	sig := offer.signature
	offer.signature = []byte{}
	data := offer.ToBytes()
	// Restore signature.
	offer.signature = sig
	return crypto.Verify(offer.addr, sig, data)
}

// ToBytes is used to serialize this paych offer.
// It returns the bytes serialized from this offer.
func (offer *PaychOffer) ToBytes() []byte {
	data, _ := json.Marshal(paychOfferJson{
		CurrencyID: offer.currencyID,
		Addr:       offer.addr,
		Settlement: offer.settlement,
		Expiration: offer.expiration,
		Signature:  hex.EncodeToString(offer.signature),
	})
	return data
}

// FromBytes is used to deserialize bytes into this paych offer.
// It takes a byte array as the argument.
// It returns the pay offer and the error in deserialization.
func FromBytes(data []byte) (*PaychOffer, error) {
	offer := &PaychOffer{}
	offerJson := paychOfferJson{}
	err := json.Unmarshal(data, &offerJson)
	if err != nil {
		return nil, err
	}
	signature, err := hex.DecodeString(offerJson.Signature)
	if err != nil {
		return nil, err
	}
	offer.currencyID = offerJson.CurrencyID
	offer.addr = offerJson.Addr
	offer.settlement = offerJson.Settlement
	offer.expiration = offerJson.Expiration
	offer.signature = signature
	return offer, nil
}
