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

	logging "github.com/ipfs/go-log"
)

// Logger
var log = logging.Logger("transactor")

// Transactor is the interface for an on-chain transactor. For each currency id, it can be used to access
// payment channel related on-chain transactions. The transactor uses the private key stored in the signer
// to sign and send transactions.
type Transactor interface {
	// Create is used to create a payment chanel with given recipient and amount.
	//
	// @input - context, currency id, recipient address, amount.
	//
	// @output - channel address, error.
	Create(ctx context.Context, currencyID byte, toAddr string, amt *big.Int) (string, error)

	// Topup is used to topup a given payment channel with given amount.
	//
	// @input - context, currency id, channel address, amount.
	//
	// @output - error.
	Topup(ctx context.Context, currencyID byte, chAddr string, amt *big.Int) error

	// Check is used to check the status of a given payment channel.
	//
	// @input - context, currency id, channel address.
	//
	// @output - settling height, min settling height, current height, channel redeemed, channel balance, sender address, recipient address and error.
	Check(ctx context.Context, currencyID byte, chAddr string) (int64, int64, int64, *big.Int, *big.Int, string, string, error)

	// Update is used to update channel state with a voucher.
	//
	// @input - context, currency id, channel address, voucher.
	//
	// @output - error.
	Update(ctx context.Context, currencyID byte, chAddr string, voucher string) error

	// Settle is used to settle a payment channel.
	//
	// @input - context, currency id, channel address.
	//
	// @output - error.
	Settle(ctx context.Context, currencyID byte, chAddr string) error

	// Collect is used to collect a payment channel.
	//
	// @input - context, currency id, channel address.
	//
	// @output - error.
	Collect(ctx context.Context, currencyID byte, chAddr string) error

	// GenerateVoucher is used to generate a voucher.
	//
	// @input - context, currency id, channel address, lane number, nonce, redeemed amount.
	//
	// @output - voucher, error.
	GenerateVoucher(ctx context.Context, currencyID byte, chAddr string, lane uint64, nonce uint64, redeemed *big.Int) (string, error)

	// VerifyVoucher is used to decode a given voucher.
	//
	// @input - currency id, voucher.
	//
	// @output - sender address, channel address, lane number, nonce, redeemed, error.
	VerifyVoucher(currencyID byte, voucher string) (string, string, uint64, uint64, *big.Int, error)

	// GetHeight is used to get the current height of the chain.
	//
	// @input - context, currency id.
	//
	// @output - height, error.
	GetHeight(ctx context.Context, currencyID byte) (int64, error)

	// GetBalance is used to get the balance of a given address.
	//
	// @input - context, currency id, address.
	//
	// @output - balance, error.
	GetBalance(ctx context.Context, currencyID byte, addr string) (*big.Int, error)
}
