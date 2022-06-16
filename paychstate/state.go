package paychstate

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
)

// State is the state of a payment channel.
type State struct {
	// Currency id of the payment channel.
	CurrencyID byte

	// Sender of the payment channel.
	FromAddr string

	// Recipient of the payment channel.
	ToAddr string

	// Address of the payment channel.
	ChAddr string

	// The total balance of the payment channel.
	Balance *big.Int

	// The redeemed amount of the payment channel.
	Redeemed *big.Int

	// Nonce of the payment channel.
	Nonce uint64

	// The latest voucher of the payment channel.
	Voucher string

	// The potential network loss voucher of this payment channel.
	NetworkLossVoucher string

	// The nonce of this state.
	StateNonce uint64
}

// Encode encodes a state into bytes.
//
// @output - data, error.
func (s State) Encode() ([]byte, error) {
	type valJson struct {
		CurrencyID         byte     `json:"currency_id"`
		FromAddr           string   `json:"from_addr"`
		ToAddr             string   `json:"to_addr"`
		ChAddr             string   `json:"ch_addr"`
		Balance            *big.Int `json:"balance"`
		Redeemed           *big.Int `json:"redeemed"`
		Nonce              uint64   `json:"nonce"`
		Voucher            string   `json:"voucher"`
		NetworkLossVoucher string   `json:"network_loss_voucher"`
		StateNonce         uint64   `json:"state_nonce"`
	}
	return json.Marshal(valJson{
		CurrencyID:         s.CurrencyID,
		FromAddr:           s.FromAddr,
		ToAddr:             s.ToAddr,
		ChAddr:             s.ChAddr,
		Balance:            s.Balance,
		Redeemed:           s.Redeemed,
		Nonce:              s.Nonce,
		Voucher:            s.Voucher,
		NetworkLossVoucher: s.NetworkLossVoucher,
		StateNonce:         s.StateNonce,
	})
}

// Decode sets the state to be decoded state from given bytes.
//
// @input - data.
//
// @output - error.
func (s *State) Decode(data []byte) error {
	type valJson struct {
		CurrencyID         byte     `json:"currency_id"`
		FromAddr           string   `json:"from_addr"`
		ToAddr             string   `json:"to_addr"`
		ChAddr             string   `json:"ch_addr"`
		Balance            *big.Int `json:"balance"`
		Redeemed           *big.Int `json:"redeemed"`
		Nonce              uint64   `json:"nonce"`
		Voucher            string   `json:"voucher"`
		NetworkLossVoucher string   `json:"network_loss_voucher"`
		StateNonce         uint64   `json:"state_nonce"`
	}
	valDec := valJson{}
	err := json.Unmarshal(data, &valDec)
	if err != nil {
		return err
	}
	s.CurrencyID = valDec.CurrencyID
	s.FromAddr = valDec.FromAddr
	s.ToAddr = valDec.ToAddr
	s.ChAddr = valDec.ChAddr
	s.Balance = valDec.Balance
	s.Redeemed = valDec.Redeemed
	s.Nonce = valDec.Nonce
	s.Voucher = valDec.Voucher
	s.NetworkLossVoucher = valDec.NetworkLossVoucher
	s.StateNonce = valDec.StateNonce
	return nil
}
