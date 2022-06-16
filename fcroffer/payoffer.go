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
	"math/big"
	"time"
)

// PayOffer represents an offer for a proxy payment.
type PayOffer struct {
	// Currency ID is the currency of this payment offer.
	CurrencyID byte

	// Src Addr is the source address of this proxy payment also who creates this offer.
	SrcAddr string

	// Dest Addr is the destination address of this proxy payment.
	DestAddr string

	// PPP stands for payment per period, is the surcharge of this proxy payment.
	PPP *big.Int

	// Period is the payment interval.
	Period *big.Int

	// Amt is the maximum amount covered by this pay offer.
	Amt *big.Int

	// Expiration is the expiration time of this offer if not being used.
	Expiration time.Time

	// Inactivity is the inactivity time allowed by this offer every time after it is being used.
	Inactivity time.Duration

	// ResTo is the reserved next recipient.
	ResTo string

	// ResCh is the reservation channel.
	ResCh string

	// ResID is the reservation id.
	ResID uint64

	// Nonce is the nonce of the payment offer.
	Nonce uint64

	// SignatureType is the signature type of the signature.
	SignatureType byte

	// Signature is the signature over the offer by Src Addr.
	Signature []byte
}

// GetToBeSigned gets the data to be signed of this payment offer.
//
// @output - data to be signed, error.
func (o PayOffer) GetToBeSigned() ([]byte, error) {
	type valJson struct {
		CurrencyID byte          `json:"currency_id"`
		SrcAddr    string        `json:"src_addr"`
		DestAddr   string        `json:"dest_addr"`
		PPP        *big.Int      `json:"ppp"`
		Period     *big.Int      `json:"period"`
		Amt        *big.Int      `json:"amt"`
		Expiration time.Time     `json:"expiration"`
		Inactivity time.Duration `json:"inactivity"`
		ResTo      string        `json:"res_to"`
		ResCh      string        `json:"res_ch"`
		ResID      uint64        `json:"res_id"`
		Nonce      uint64        `json:"nonce"`
	}
	return json.Marshal(valJson{
		CurrencyID: o.CurrencyID,
		SrcAddr:    o.SrcAddr,
		DestAddr:   o.DestAddr,
		PPP:        o.PPP,
		Period:     o.Period,
		Amt:        o.Amt,
		Expiration: o.Expiration,
		Inactivity: o.Inactivity,
		ResTo:      o.ResTo,
		ResCh:      o.ResCh,
		ResID:      o.ResID,
		Nonce:      o.Nonce,
	})
}

// AddSignature adds a signature to the payment offer.
//
// @input - signature type, signature.
func (o *PayOffer) AddSignature(sigType byte, signature []byte) {
	o.SignatureType = sigType
	o.Signature = signature
}

// Encode encodes the payment offer.
//
// @output - data, error.
func (o PayOffer) Encode() ([]byte, error) {
	type valJson struct {
		CurrencyID    byte          `json:"currency_id"`
		SrcAddr       string        `json:"src_addr"`
		DestAddr      string        `json:"dest_addr"`
		PPP           *big.Int      `json:"ppp"`
		Period        *big.Int      `json:"period"`
		Amt           *big.Int      `json:"amt"`
		CreatedAt     time.Time     `json:"created_at"`
		Expiration    time.Time     `json:"expiration"`
		Inactivity    time.Duration `json:"inactivity"`
		ResTo         string        `json:"res_to"`
		ResCh         string        `json:"res_ch"`
		ResID         uint64        `json:"res_id"`
		Nonce         uint64        `json:"nonce"`
		SignatureType byte          `json:"signature_type"`
		Signature     []byte        `json:"signature"`
	}
	return json.Marshal(valJson{
		CurrencyID:    o.CurrencyID,
		SrcAddr:       o.SrcAddr,
		DestAddr:      o.DestAddr,
		PPP:           o.PPP,
		Period:        o.Period,
		Amt:           o.Amt,
		Expiration:    o.Expiration,
		Inactivity:    o.Inactivity,
		ResTo:         o.ResTo,
		ResCh:         o.ResCh,
		ResID:         o.ResID,
		Nonce:         o.Nonce,
		SignatureType: o.SignatureType,
		Signature:     o.Signature,
	})
}

// Decode decodes a payment offer data bytes.
//
// @input - data.
//
// @output - error.
func (o *PayOffer) Decode(data []byte) error {
	type valJson struct {
		CurrencyID    byte          `json:"currency_id"`
		SrcAddr       string        `json:"src_addr"`
		DestAddr      string        `json:"dest_addr"`
		PPP           *big.Int      `json:"ppp"`
		Period        *big.Int      `json:"period"`
		Amt           *big.Int      `json:"amt"`
		Expiration    time.Time     `json:"expiration"`
		Inactivity    time.Duration `json:"inactivity"`
		ResTo         string        `json:"res_to"`
		ResCh         string        `json:"res_ch"`
		ResID         uint64        `json:"res_id"`
		Nonce         uint64        `json:"nonce"`
		SignatureType byte          `json:"signature_type"`
		Signature     []byte        `json:"signature"`
	}
	valDec := valJson{}
	err := json.Unmarshal(data, &valDec)
	if err != nil {
		return err
	}
	o.CurrencyID = valDec.CurrencyID
	o.SrcAddr = valDec.SrcAddr
	o.DestAddr = valDec.DestAddr
	o.PPP = valDec.PPP
	o.Period = valDec.Period
	o.Amt = valDec.Amt
	o.Expiration = valDec.Expiration
	o.Inactivity = valDec.Inactivity
	o.ResTo = valDec.ResTo
	o.ResCh = valDec.ResCh
	o.ResID = valDec.ResID
	o.Nonce = valDec.Nonce
	o.SignatureType = valDec.SignatureType
	o.Signature = valDec.Signature
	return nil
}
