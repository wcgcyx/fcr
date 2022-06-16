package trans

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
	"fmt"
	"math/big"

	"github.com/wcgcyx/fcr/crypto"
)

// TransactorImpl is the implementation of the Transactor interface.
type TransactorImpl struct {
	adapters map[byte]adapter
}

// NewTransactorImpl creates a new TransactorImpl.
//
// @input - context, signer, options.
//
// @output - store, error.
func NewTransactorImpl(ctx context.Context, signer crypto.Signer, opts Opts) (*TransactorImpl, error) {
	log.Infof("Start transactor...")
	adapters := make(map[byte]adapter)
	// Parse options
	if opts.FilecoinEnabled {
		if opts.FilecoinAPI == "" {
			return nil, fmt.Errorf("empty filecoin api provided")
		}
		filConfidence := defaultFilecoinConfidence
		if opts.FilecoinConfidence != nil {
			filConfidence = *opts.FilecoinConfidence
		}
		filAdapter, err := newFilAdapter(ctx, signer, opts.FilecoinAPI, opts.FilecoinAuthToken, filConfidence)
		if err != nil {
			return nil, err
		}
		adapters[crypto.FIL] = filAdapter
	}
	if len(adapters) == 0 {
		return nil, fmt.Errorf("no adapter has been initialized")
	}
	return &TransactorImpl{adapters: adapters}, nil
}

// Create is used to create a payment chanel with given recipient and amount.
//
// @input - context, currency id, recipient address, amount.
//
// @output - channel address, error.
func (t *TransactorImpl) Create(ctx context.Context, currencyID byte, toAddr string, amt *big.Int) (string, error) {
	adapter, ok := t.adapters[currencyID]
	if !ok {
		return "", fmt.Errorf("currency id %v not supported", currencyID)
	}
	return adapter.Create(ctx, toAddr, amt)
}

// Topup is used to topup a given payment channel with given amount.
//
// @input - context, currency id, channel address, amount.
//
// @output - error.
func (t *TransactorImpl) Topup(ctx context.Context, currencyID byte, chAddr string, amt *big.Int) error {
	adapter, ok := t.adapters[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v not supported", currencyID)
	}
	return adapter.Topup(ctx, chAddr, amt)
}

// Check is used to check the status of a given payment channel.
//
// @input - context, currency id, channel address.
//
// @output - settling height, min settling height, current height, channel redeemed, channel balance, sender address, recipient address and error.
func (t *TransactorImpl) Check(ctx context.Context, currencyID byte, chAddr string) (int64, int64, int64, *big.Int, *big.Int, string, string, error) {
	adapter, ok := t.adapters[currencyID]
	if !ok {
		return 0, 0, 0, nil, nil, "", "", fmt.Errorf("currency id %v not supported", currencyID)
	}
	return adapter.Check(ctx, chAddr)
}

// Update is used to update channel state with a voucher.
//
// @input - context, currency id, channel address, voucher.
//
// @output - error.
func (t *TransactorImpl) Update(ctx context.Context, currencyID byte, chAddr string, voucher string) error {
	adapter, ok := t.adapters[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v not supported", currencyID)
	}
	return adapter.Update(ctx, chAddr, voucher)
}

// Settle is used to settle a payment channel.
//
// @input - context, currency id, channel address.
//
// @output - error.
func (t *TransactorImpl) Settle(ctx context.Context, currencyID byte, chAddr string) error {
	adapter, ok := t.adapters[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v not supported", currencyID)
	}
	return adapter.Settle(ctx, chAddr)
}

// Collect is used to collect a payment channel.
//
// @input - context, currency id, channel address.
//
// @output - error.
func (t *TransactorImpl) Collect(ctx context.Context, currencyID byte, chAddr string) error {
	adapter, ok := t.adapters[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v not supported", currencyID)
	}
	return adapter.Collect(ctx, chAddr)
}

// GenerateVoucher is used to generate a voucher.
//
// @input - context, currency id, channel address, lane number, nonce, redeemed amount.
//
// @output - voucher, error.
func (t *TransactorImpl) GenerateVoucher(ctx context.Context, currencyID byte, chAddr string, lane uint64, nonce uint64, redeemed *big.Int) (string, error) {
	adapter, ok := t.adapters[currencyID]
	if !ok {
		return "", fmt.Errorf("currency id %v not supported", currencyID)
	}
	return adapter.GenerateVoucher(ctx, chAddr, lane, nonce, redeemed)
}

// VerifyVoucher is used to decode a given voucher.
//
// @input - currency id, voucher.
//
// @output - sender address, channel address, lane number, nonce, redeemed, error.
func (t *TransactorImpl) VerifyVoucher(currencyID byte, voucher string) (string, string, uint64, uint64, *big.Int, error) {
	adapter, ok := t.adapters[currencyID]
	if !ok {
		return "", "", 0, 0, nil, fmt.Errorf("currency id %v not supported", currencyID)
	}
	return adapter.VerifyVoucher(voucher)
}

// GetHeight is used to get the current height of the chain.
//
// @input - context, currency id.
//
// @output - height, error.
func (t *TransactorImpl) GetHeight(ctx context.Context, currencyID byte) (int64, error) {
	adapter, ok := t.adapters[currencyID]
	if !ok {
		return 0, fmt.Errorf("currency id %v not supported", currencyID)
	}
	return adapter.GetHeight(ctx)
}

// GetBalance is used to get the balance of a given address.
//
// @input - context, currency id, address.
//
// @output - balance, error.
func (t *TransactorImpl) GetBalance(ctx context.Context, currencyID byte, addr string) (*big.Int, error) {
	adapter, ok := t.adapters[currencyID]
	if !ok {
		return nil, fmt.Errorf("currency id %v not supported", currencyID)
	}
	return adapter.GetBalance(ctx, addr)
}
