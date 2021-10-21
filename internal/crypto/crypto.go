package crypto

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-crypto"
	"github.com/minio/blake2b-simd"
)

// GenerateKeyPair is used to generate a secp256k1 KeyPair.
// It returns the private key, associated filecoin address and error.
func GenerateKeyPair() ([]byte, string, error) {
	prv, err := crypto.GenerateKey()
	if err != nil {
		return nil, "", err
	}
	addrStr, err := GetAddress(prv)
	if err != nil {
		return nil, "", err
	}
	return prv, addrStr, nil
}

// GetAddress is used to obtain the address of a private key.
// It takes the private key as the argument.
// It returns the associated filecoin address and error.
func GetAddress(prv []byte) (string, error) {
	pub := crypto.PublicKey(prv)
	addr, err := address.NewSecp256k1Address(pub)
	if err != nil {
		return "", err
	}
	// Enforce the address to always start with "f".
	addrStr := "f" + addr.String()[1:]
	return addrStr, nil
}

// Sign is used to generate signature for given private key and data.
// It takes the private key and the data as arguments.
// It returns the signature and error.
func Sign(prv []byte, data []byte) ([]byte, error) {
	b2sum := blake2b.Sum256(data)
	return crypto.Sign(prv, b2sum[:])
}

// Verify is used to verify data against its signature and associated filecoin address.
// It takes the address, signature and data as arguments.
// It returns the error.
func Verify(addr string, sig []byte, data []byte) error {
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
		return fmt.Errorf("Signature fail to verify, got %s, expect %s", maybeAddrStr, addr)
	}
	return nil
}
