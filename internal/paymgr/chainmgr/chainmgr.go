package chainmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"math/big"
)

// ChainManager is used to interact with the chain regarding payment channel onchain operations.
type ChainManager interface {
	// Create is used to create a payment channel.
	// It takes a context, private key, recipient address and amount as arguments.
	// It returns the channel address and error.
	Create(ctx context.Context, prv []byte, toAddr string, amt *big.Int) (string, error)

	// Topup is used to topup a payment channel.
	// It takes a context, private key, channel address and amount as arguments.
	// It returns the error.
	Topup(ctx context.Context, prv []byte, chAddr string, amt *big.Int) error

	// Settle is used to settle a payment channel.
	// It takes a context, private key, channel address and final voucher as arguments.
	// It returns the error.
	Settle(ctx context.Context, prv []byte, chAddr string, voucher string) error

	// Collect is used to collect a payment channel.
	// It takes a context, private key, channel address as arguments.
	// It returns the error.
	Collect(ctx context.Context, prv []byte, chAddr string) error

	// Check is used to check the status of a payment channel.
	// It takes a context, channel address as arguments.
	// It returns a settling height, current height, channel balance, sender address, recipient address and error.
	Check(ctx context.Context, chAddr string) (int64, int64, *big.Int, string, string, error)

	// GenerateVoucher is used to generate a voucher.
	// It takes the private key, channel address, lane number, nonce, redeemed amount as arguments.
	// It returns voucher and error.
	GenerateVoucher(prv []byte, chAddr string, lane uint64, nonce uint64, redeemed *big.Int) (string, error)

	// VerifyVoucher is used to verify a voucher.
	// It takes the voucher as the argument.
	// It returns the sender's address, channel address, lane number, nonce, redeemed and error.
	VerifyVoucher(voucher string) (string, string, uint64, uint64, *big.Int, error)
}

// CurrencyID for chains
// 0 - Mocked Chain
// 1 - FIL Mainnet
// 2 - FIL Calibration
// 3 - ETH Mainnet
// 4 - ETH Rinkeby

const (
	MockedChainCurrencyID    = 0
	FILMainnetCurrencyID     = 1
	CurrencyID1Name          = "FIL Mainnet"
	FILCalibrationCurrencyID = 2
	CurrencyID2Name          = "FIL Calibration"
	ETHMainnetCurrencyID     = 3
	CurrencyID3Name          = "ETH Mainnet"
	ETHRinkebyCurrencyID     = 4
	CurrencyID4Name          = "ETH Rinkeby"
)
