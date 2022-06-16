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
	"math/big"
)

// adapter is the interface for an adapter to use payment channel funcionalities.
type adapter interface {
	// Create is used to create a payment chanel with given recipient and amount.
	//
	// @input - context, recipient address, amount.
	//
	// @output - channel address, error.
	Create(ctx context.Context, toAddr string, amt *big.Int) (string, error)

	// Topup is used to topup a given payment channel with given amount.
	//
	// @input - context, channel address, amount.
	//
	// @output - error.
	Topup(ctx context.Context, chAddr string, amt *big.Int) error

	// Check is used to check the status of a given payment channel.
	//
	// @input - context, channel address.
	//
	// @output - settling height, min settling height, current height, channel redeemed, channel balance, sender address, recipient address and error.
	Check(ctx context.Context, chAddr string) (int64, int64, int64, *big.Int, *big.Int, string, string, error)

	// Update is used to update channel state with a voucher.
	//
	// @input - context, channel address, voucher.
	//
	// @output - error.
	Update(ctx context.Context, chAddr string, voucher string) error

	// Settle is used to settle a payment channel.
	//
	// @input - context, channel address.
	//
	// @output - error.
	Settle(ctx context.Context, chAddr string) error

	// Collect is used to collect a payment channel.
	//
	// @input - context, channel address.
	//
	// @output - error.
	Collect(ctx context.Context, chAddr string) error

	// GetHeight is used to get the current height of the chain.
	//
	// @input - context.
	//
	// @output - height, error.
	GetHeight(ctx context.Context) (int64, error)

	// GetBalance is used to get the balance of a given address.
	//
	// @input - context, address.
	//
	// @output - balance, error.
	GetBalance(ctx context.Context, addr string) (*big.Int, error)

	// GenerateVoucher is used to generate a voucher.
	//
	// @input - context, channel address, lane number, nonce, redeemed amount.
	//
	// @output - voucher, error.
	GenerateVoucher(ctx context.Context, chAddr string, lane uint64, nonce uint64, redeemed *big.Int) (string, error)

	// VerifyVoucher is used to decode a given voucher.
	//
	// @input - voucher.
	//
	// @output - sender address, channel address, lane number, nonce, redeemed, error.
	VerifyVoucher(voucher string) (string, string, uint64, uint64, *big.Int, error)
}
