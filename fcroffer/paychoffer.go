package fcroffer

/*
 * Dual-licensed under Apache-2.0 and MIT.
 *
 * You can get a copy of the Apache License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * You can also get a copy of the MIT License at
 *
 * http://opensource.org/licenses/MIT
 *
 * @wcgcyx - https://github.com/wcgcyx
 */

import (
	"encoding/json"
	"time"
)

// PaychOffer represents an offer for a paych add request.
type PaychOffer struct {
	// Currency ID is the currency of this paych offer.
	CurrencyID byte

	// From Addr is the offer recipient address and also the sender of future payment channels.
	FromAddr string

	// To Addr is the source address of this offer and who creates this offer.
	ToAddr string

	// Settlement is the offered minimum settlement time.
	Settlement time.Time

	// Expiration is the expiration time of this offer.
	Expiration time.Time

	// Nonce is the nonce of the offer.
	Nonce uint64

	// SignatureType is the signature type.
	SignatureType byte

	// Signature is the signature over the offer by Src Addr.
	Signature []byte
}

// GetToBeSigned gets the data to be signed of this offer.
//
// @output - data to be signed, error.
func (o PaychOffer) GetToBeSigned() ([]byte, error) {
	type valJson struct {
		CurrencyID byte      `json:"currency_id"`
		FromAddr   string    `json:"from_addr"`
		ToAddr     string    `json:"to_addr"`
		Settlement time.Time `json:"settlement"`
		Expiration time.Time `json:"expiration"`
		Nonce      uint64    `json:"nonce"`
	}
	return json.Marshal(valJson{
		CurrencyID: o.CurrencyID,
		FromAddr:   o.FromAddr,
		ToAddr:     o.ToAddr,
		Settlement: o.Settlement,
		Expiration: o.Expiration,
		Nonce:      o.Nonce,
	})
}

// AddSignature adds a signature to the offer.
//
// @input - signature type, signature.
func (o *PaychOffer) AddSignature(sigType byte, signature []byte) {
	o.SignatureType = sigType
	o.Signature = signature
}

// Encode encodes the offer.
//
// @output - data, error.
func (o PaychOffer) Encode() ([]byte, error) {
	type valJson struct {
		CurrencyID    byte      `json:"currency_id"`
		FromAddr      string    `json:"from_addr"`
		ToAddr        string    `json:"to_addr"`
		Settlement    time.Time `json:"settlement"`
		Expiration    time.Time `json:"expiration"`
		Nonce         uint64    `json:"nonce"`
		SignatureType byte      `json:"signature_type"`
		Signature     []byte    `json:"signature"`
	}
	return json.Marshal(valJson{
		CurrencyID:    o.CurrencyID,
		FromAddr:      o.FromAddr,
		ToAddr:        o.ToAddr,
		Settlement:    o.Settlement,
		Expiration:    o.Expiration,
		Nonce:         o.Nonce,
		SignatureType: o.SignatureType,
		Signature:     o.Signature,
	})
}

// Decode decodes a offer data bytes.
//
// @input - data.
//
// @output - error.
func (o *PaychOffer) Decode(data []byte) error {
	type valJson struct {
		CurrencyID    byte      `json:"currency_id"`
		FromAddr      string    `json:"from_addr"`
		ToAddr        string    `json:"to_addr"`
		Settlement    time.Time `json:"settlement"`
		Expiration    time.Time `json:"expiration"`
		Nonce         uint64    `json:"nonce"`
		SignatureType byte      `json:"signature_type"`
		Signature     []byte    `json:"signature"`
	}
	valDec := valJson{}
	err := json.Unmarshal(data, &valDec)
	if err != nil {
		return err
	}
	o.CurrencyID = valDec.CurrencyID
	o.FromAddr = valDec.FromAddr
	o.ToAddr = valDec.ToAddr
	o.Settlement = valDec.Settlement
	o.Expiration = valDec.Expiration
	o.Nonce = valDec.Nonce
	o.SignatureType = valDec.SignatureType
	o.Signature = valDec.Signature
	return nil
}
