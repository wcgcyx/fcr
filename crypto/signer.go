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
	"context"
	"fmt"
	"time"

	logging "github.com/ipfs/go-log"
)

// Logger
var log = logging.Logger("signer")

// Signer is the interface for a secured signing tool. It can save one key for a given currency id.
// The key is used for signing offers and for transacting on chain.
type Signer interface {
	// SetKey is used to set a private key for a given currency id. It will return error if there is
	// an existing key already associated with the given currency id.
	//
	// @input - context, currency id, key type, private key.
	//
	// @output - error.
	SetKey(ctx context.Context, currencyID byte, keyType byte, prv []byte) error

	// GetAddr is used to obtain the key type and the address derived from the private key associated
	// with the given currency id.
	//
	// @input - context, currency id.
	//
	// @output - key type, address, error.
	GetAddr(ctx context.Context, currencyID byte) (byte, string, error)

	// Sign is used to sign given data with the private key associated with given currency id.
	//
	// @input - context, currency id, data.
	//
	// @output - signature type, signature, error.
	Sign(ctx context.Context, currencyID byte, data []byte) (byte, []byte, error)

	// RetireKey is used to retire a key for a given currency id. If within timeout the key is touched
	// it will return error.
	//
	// @input - context, currency id, timeout.
	//
	// @output - error channel out.
	RetireKey(ctx context.Context, currencyID byte, timeout time.Duration) <-chan error

	// StopRetire is used to stop retiring a key for a given currency id.
	//
	// @input - context, currency id.
	//
	// @output - error.
	StopRetire(ctx context.Context, currencyID byte) error

	// ListCurrencyIDs lists all currencies.
	//
	// @input - context.
	//
	// @output - currency chan out, error chan out.
	ListCurrencyIDs(ctx context.Context) (<-chan byte, <-chan error)
}

// Verify is a static method to verify given data with given signature from given address.
//
// @input - currency id, data, key type, signature, signer's address.
//
// @output - error.
func Verify(currencyID byte, data []byte, keyType byte, sig []byte, addr string) error {
	verifier, err := getVerifier(currencyID, keyType)
	if err != nil {
		return err
	}
	return verifier(data, sig, addr)
}

// sigVerifiers contains all registered verifiers.
// It is a map of (currency id -> map of (key type -> verifier)).
var sigVerifiers map[byte]map[byte]func(data []byte, sig []byte, addr string) error

// getVerifier gets verifier for given currency id and key type.
//
// @input - currency id, key type.
//
// @output - verifier function, error.
func getVerifier(currencyID byte, keyType byte) (func(data []byte, sig []byte, addr string) error, error) {
	if sigVerifiers == nil {
		return nil, fmt.Errorf("no verifier is initialized")
	}
	verifiers, ok := sigVerifiers[currencyID]
	if !ok {
		return nil, fmt.Errorf("no verifier for currency id %v is initialized", currencyID)
	}
	verifier, ok := verifiers[keyType]
	if !ok {
		return nil, fmt.Errorf("no verifier for currency id %v and key type %v is initialized", currencyID, keyType)
	}
	return verifier, nil
}

// dataSigners contains all registered signers.
// It is a map of (currency id -> map of (key type -> signer)).
var dataSigners map[byte]map[byte]func(prv []byte, data []byte) ([]byte, error)

// getSigner gets signer for given currency id and key type.
//
// @input - currency id, key type.
//
// @output - signer function, error.
func getSigner(currencyID byte, keyType byte) (func(prv []byte, data []byte) ([]byte, error), error) {
	if dataSigners == nil {
		return nil, fmt.Errorf("no signer is initialized")
	}
	signers, ok := dataSigners[currencyID]
	if !ok {
		return nil, fmt.Errorf("no signer for currency id %v is initialized", currencyID)
	}
	signer, ok := signers[keyType]
	if !ok {
		return nil, fmt.Errorf("no signer for currency id %v and key type %v is initialized", currencyID, keyType)
	}
	return signer, nil
}

// addrResolvers contains all registered address resolvers.
// It is a map of (currency id -> map of (key type -> resolver)).
var addrResolvers map[byte]map[byte]func(key []byte, prv bool) (string, error)

// getResolver gets resolver for given currency id and key type.
//
// @input - currency id, key type.
//
// @output - resolver function, error.
func getResolver(currencyID byte, keyType byte) (func(key []byte, prv bool) (string, error), error) {
	if addrResolvers == nil {
		return nil, fmt.Errorf("no resolver is initialized")
	}
	resolvers, ok := addrResolvers[currencyID]
	if !ok {
		return nil, fmt.Errorf("no resolver for currency id %v is initialized", currencyID)
	}
	resolver, ok := resolvers[keyType]
	if !ok {
		return nil, fmt.Errorf("no resolver for currency id %v and key type %v is initialized", currencyID, keyType)
	}
	return resolver, nil
}
