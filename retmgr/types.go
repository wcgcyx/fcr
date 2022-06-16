package retmgr

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
	"context"
	"encoding/json"
	"math/big"
	"time"

	golock "github.com/viney-shih/go-lock"
	"github.com/wcgcyx/fcr/fcroffer"
)

// outgoingRetrievalState is the state of an outgoing retrieval request.
type outgoingRetrievalState struct {
	// Currency ID.
	currencyID byte

	// To address.
	toAddr string

	// Inactivity of this state.
	inactivity time.Duration

	// Payment per byte of this state.
	ppb *big.Int

	// State expiration time.
	expiredAt time.Time

	// State Lock
	lock golock.RWMutex

	// Context indicating if state is being processed.
	ctx context.Context

	// Optional pay offer
	payOffer *fcroffer.PayOffer

	// Reserved payment channel
	resCh string

	// Reservation id
	resID uint64
}

// incomingRetrievalState is the state of an incoming retrieval request.
type incomingRetrievalState struct {
	// Currency ID.
	currencyID byte

	// From address.
	fromAddr string

	// Inactivity of this state.
	inactivity time.Duration

	// Payment per byte of this state.
	ppb *big.Int

	// Block index paid
	index int64

	// Current payment required.
	paymentRequired *big.Int

	// State expiration time.
	expiredAt time.Time

	// State Lock
	lock golock.RWMutex

	// Boolean indicating if state is being processed.
	processed bool
}

// encode encodes the retrieval state to bytes.
//
// @output - data, error.
func (s *incomingRetrievalState) encode() ([]byte, error) {
	type stateJson struct {
		CurrencyID      byte          `json:"currency_id"`
		FromAddr        string        `json:"from_addr"`
		Inactivity      time.Duration `json:"inactivity"`
		PPB             *big.Int      `json:"ppb"`
		Index           int64         `json:"index"`
		PaymentRequired *big.Int      `json:"payment_required"`
		ExpiredAt       time.Time     `json:"expired_at"`
	}
	return json.Marshal(stateJson{
		CurrencyID:      s.currencyID,
		FromAddr:        s.fromAddr,
		Inactivity:      s.inactivity,
		PPB:             s.ppb,
		Index:           s.index,
		PaymentRequired: s.paymentRequired,
		ExpiredAt:       s.expiredAt,
	})
}

// decode decodes the retrieval state from bytes.
//
// @input - data.
//
// @output - error.
func (s *incomingRetrievalState) decode(data []byte) error {
	type stateJson struct {
		CurrencyID      byte          `json:"currency_id"`
		FromAddr        string        `json:"from_addr"`
		Inactivity      time.Duration `json:"inactivity"`
		PPB             *big.Int      `json:"ppb"`
		Index           int64         `json:"index"`
		PaymentRequired *big.Int      `json:"payment_required"`
		ExpiredAt       time.Time     `json:"expired_at"`
	}
	decoded := stateJson{}
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return err
	}
	s.currencyID = decoded.CurrencyID
	s.fromAddr = decoded.FromAddr
	s.inactivity = decoded.Inactivity
	s.ppb = decoded.PPB
	s.index = decoded.Index
	s.paymentRequired = decoded.PaymentRequired
	s.expiredAt = decoded.ExpiredAt
	s.lock = golock.NewCASMutex()
	s.processed = false
	return nil
}
