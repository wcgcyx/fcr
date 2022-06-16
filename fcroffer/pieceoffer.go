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

	"github.com/ipfs/go-cid"
)

// PieceOffer represents an offer for a piece retrieval.
type PieceOffer struct {
	// ID is the root id of the ipld graph of this piece.
	ID cid.Cid

	// Size is the size of the piece.
	Size uint64

	// Currency ID the currency of this piece offer.
	CurrencyID byte

	// PPB stands for payment per byte, is the price of the retireval.
	PPB *big.Int

	// Recipient Addr is the payment recipient address also who creates this offer.
	RecipientAddr string

	// LinkedMinerKeyType is the key type of the linked miner (optional).
	LinkedMinerKeyType byte

	// LinkedMinerAddr is the address of the linked miner (optional).
	LinkedMinerAddr string

	// LinkedMinerProof is the proof provided by the linked miner or a signature of miner on recipient address (optional).
	LinkedMinerProof []byte

	// Expiration is the expiration time of this offer if not being used.
	Expiration time.Time

	// Inactivity is the inactivity time allowed by this offer every time after it is being used.
	Inactivity time.Duration

	// Nonce is the nonce of the payment offer.
	Nonce uint64

	// SignatureType is the signature type of the signature.
	SignatureType byte

	// Signature is the signature over the offer by Recipient Addr.
	Signature []byte
}

// GetToBeSigned gets the data to be signed of this piece offer.
//
// @output - data to be signed, error.
func (o PieceOffer) GetToBeSigned() ([]byte, error) {
	type valJson struct {
		ID                 cid.Cid       `json:"id"`
		Size               uint64        `json:"size"`
		CurrencyID         byte          `json:"currency_id"`
		PPB                *big.Int      `json:"ppb"`
		RecipientAddr      string        `json:"recipient_addr"`
		LinkedMinerKeyType byte          `json:"key_type"`
		LinkedMinerAddr    string        `json:"miner_addr"`
		LinkedMinerProof   []byte        `json:"miner_proof"`
		Expiration         time.Time     `json:"expiration"`
		Inactivity         time.Duration `json:"inactivity"`
		Nonce              uint64        `json:"nonce"`
	}
	return json.Marshal(valJson{
		ID:                 o.ID,
		Size:               o.Size,
		CurrencyID:         o.CurrencyID,
		PPB:                o.PPB,
		RecipientAddr:      o.RecipientAddr,
		LinkedMinerKeyType: o.LinkedMinerKeyType,
		LinkedMinerAddr:    o.LinkedMinerAddr,
		LinkedMinerProof:   o.LinkedMinerProof,
		Expiration:         o.Expiration,
		Inactivity:         o.Inactivity,
		Nonce:              o.Nonce,
	})
}

// AddSignature adds a signature to the piece offer.
//
// @input - signature type, signature.
func (o *PieceOffer) AddSignature(sigType byte, signature []byte) {
	o.SignatureType = sigType
	o.Signature = signature
}

// Encode encodes the piece offer.
//
// @output - data, error.
func (o PieceOffer) Encode() ([]byte, error) {
	type valJson struct {
		ID                 cid.Cid       `json:"id"`
		Size               uint64        `json:"size"`
		CurrencyID         byte          `json:"currency_id"`
		PPB                *big.Int      `json:"ppb"`
		RecipientAddr      string        `json:"recipient_addr"`
		LinkedMinerKeyType byte          `json:"key_type"`
		LinkedMinerAddr    string        `json:"miner_addr"`
		LinkedMinerProof   []byte        `json:"miner_proof"`
		Expiration         time.Time     `json:"expiration"`
		Inactivity         time.Duration `json:"inactivity"`
		Nonce              uint64        `json:"nonce"`
		SignatureType      byte          `json:"signature_type"`
		Signature          []byte        `json:"signature"`
	}
	return json.Marshal(valJson{
		ID:                 o.ID,
		Size:               o.Size,
		CurrencyID:         o.CurrencyID,
		PPB:                o.PPB,
		RecipientAddr:      o.RecipientAddr,
		LinkedMinerKeyType: o.LinkedMinerKeyType,
		LinkedMinerAddr:    o.LinkedMinerAddr,
		LinkedMinerProof:   o.LinkedMinerProof,
		Expiration:         o.Expiration,
		Inactivity:         o.Inactivity,
		Nonce:              o.Nonce,
		SignatureType:      o.SignatureType,
		Signature:          o.Signature,
	})
}

// Decode decodes a piece offer data bytes.
//
// @input - data.
//
// @output - error.
func (o *PieceOffer) Decode(data []byte) error {
	type valJson struct {
		ID                 cid.Cid       `json:"id"`
		Size               uint64        `json:"size"`
		CurrencyID         byte          `json:"currency_id"`
		PPB                *big.Int      `json:"ppb"`
		RecipientAddr      string        `json:"recipient_addr"`
		LinkedMinerKeyType byte          `json:"key_type"`
		LinkedMinerAddr    string        `json:"miner_addr"`
		LinkedMinerProof   []byte        `json:"miner_proof"`
		Expiration         time.Time     `json:"expiration"`
		Inactivity         time.Duration `json:"inactivity"`
		Nonce              uint64        `json:"nonce"`
		SignatureType      byte          `json:"signature_type"`
		Signature          []byte        `json:"signature"`
	}
	valDec := valJson{}
	err := json.Unmarshal(data, &valDec)
	if err != nil {
		return err
	}
	o.ID = valDec.ID
	o.Size = valDec.Size
	o.CurrencyID = valDec.CurrencyID
	o.PPB = valDec.PPB
	o.RecipientAddr = valDec.RecipientAddr
	o.LinkedMinerKeyType = valDec.LinkedMinerKeyType
	o.LinkedMinerAddr = valDec.LinkedMinerAddr
	o.LinkedMinerProof = valDec.LinkedMinerProof
	o.Expiration = valDec.Expiration
	o.Inactivity = valDec.Inactivity
	o.Nonce = valDec.Nonce
	o.SignatureType = valDec.SignatureType
	o.Signature = valDec.Signature
	return nil
}
