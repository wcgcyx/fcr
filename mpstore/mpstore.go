package mpstore

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

	logging "github.com/ipfs/go-log"
	"github.com/wcgcyx/fcr/crypto"
)

// Logger
var log = logging.Logger("mpstore")

// MinerProofStore is the interface for a store that stores linked miner's proof.
type MinerProofStore interface {
	// UpsertMinerProof is used to set the miner proof for a given currency id.
	//
	// @input - context, currency id, miner key type, miner address, proof.
	//
	// @output - error.
	UpsertMinerProof(ctx context.Context, currencyID byte, minerKeyType byte, minerAddr string, proof []byte) error

	// GetMinerProof is used to get the miner proof for a given currency id.
	//
	// @input - context, currency id.
	//
	// @output - if exists, miner key type, miner address, proof, error.
	GetMinerProof(ctx context.Context, currencyID byte) (bool, byte, string, []byte, error)
}

// VerifyMinerProof is used to verify the miner proof.
//
// @input - currency id, addr (miner signed data), miner address, proof.
//
// @output - error.
func VerifyMinerProof(currencyID byte, addr string, minerKeyType byte, minerAddr string, proof []byte) error {
	return crypto.Verify(currencyID, []byte(addr), minerKeyType, proof, minerAddr)
}
