package crypto

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
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-crypto"
	"github.com/minio/blake2b-simd"
)

// filSecp256k1Verifier is the verifier of secp256k1 on filecoin.
//
// @input - data, signature, address.
//
// @output - error.
func filSecp256k1Verifier(data []byte, sig []byte, addr string) error {
	b2sum := blake2b.Sum256(data)
	pub, err := crypto.EcRecover(b2sum[:], sig)
	if err != nil {
		return err
	}
	maybeAddr, err := address.NewSecp256k1Address(pub)
	if err != nil {
		return err
	}
	// Enforce the address to always start with "f".
	maybeAddrStr := "f" + maybeAddr.String()[1:]
	if maybeAddrStr != addr {
		return fmt.Errorf("Signature fails to verify, got %s, expect %s", maybeAddrStr, addr)
	}
	return nil
}

// filSecp256k1Signer is the signer of secp256k1 on filecoin.
//
// @input - private key, data.
//
// @output - signature, error.
func filSecp256k1Signer(prv []byte, data []byte) ([]byte, error) {
	b2sum := blake2b.Sum256(data)
	return crypto.Sign(prv, b2sum[:])
}

// filSecp256k1AddrResolver is the addr resolver of secp256k1 on filecoin.
//
// @input - key, boolean indicates whether private or public key is supplied.
//
// @output - address resolved from given public key, error.
func filSecp256k1AddrResolver(key []byte, prv bool) (string, error) {
	pub := key
	if prv {
		// Check if private key has non-zero element
		valid := false
		for _, b := range key {
			if b != 0 {
				valid = true
				break
			}
		}
		if !valid {
			return "", fmt.Errorf("invalid private key provided")
		}
		pub = crypto.PublicKey(key)
	}
	addr, err := address.NewSecp256k1Address(pub)
	if err != nil {
		return "", err
	}
	// Enforce the address to always start with "f".
	addrStr := "f" + addr.String()[1:]
	return addrStr, nil
}

// init initializes this currency-key crypto methods.
func init() {
	// Init verifiers.
	if sigVerifiers == nil {
		sigVerifiers = map[byte]map[byte]func(data []byte, sig []byte, addr string) error{}
	}
	// Init signers.
	if dataSigners == nil {
		dataSigners = make(map[byte]map[byte]func(prv []byte, data []byte) ([]byte, error))
	}
	// Init addr resolvers.
	if addrResolvers == nil {
		addrResolvers = make(map[byte]map[byte]func(key []byte, prv bool) (string, error))
	}
	// Init FIL.
	if sigVerifiers[FIL] == nil {
		sigVerifiers[FIL] = map[byte]func(data []byte, sig []byte, addr string) error{}
	}
	sigVerifiers[FIL][SECP256K1] = filSecp256k1Verifier

	if dataSigners[FIL] == nil {
		dataSigners[FIL] = make(map[byte]func(prv []byte, data []byte) ([]byte, error))
	}
	dataSigners[FIL][SECP256K1] = filSecp256k1Signer

	if addrResolvers[FIL] == nil {
		addrResolvers[FIL] = make(map[byte]func(key []byte, prv bool) (string, error))
	}
	addrResolvers[FIL][SECP256K1] = filSecp256k1AddrResolver
}
