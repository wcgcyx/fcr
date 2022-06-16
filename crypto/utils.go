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
	"encoding/json"
)

// encDSVal encodes the data to the ds value.
//
// @input - key type, address, private key.
//
// @output - value, error.
func encDSVal(keyType byte, addr string, prv []byte) ([]byte, error) {
	type valJson struct {
		KeyType byte   `json:"key_type"`
		Addr    string `json:"addr"`
		Prv     []byte `json:"prv"`
	}
	return json.Marshal(valJson{
		KeyType: keyType,
		Addr:    addr,
		Prv:     prv,
	})
}

// decDSVal decodes the ds value into data.
//
// @input - value.
//
// @output - key type, address, private key, error.
func decDSVal(val []byte) (byte, string, []byte, error) {
	type valJson struct {
		KeyType byte   `json:"key_type"`
		Addr    string `json:"addr"`
		Prv     []byte `json:"prv"`
	}
	valDec := valJson{}
	err := json.Unmarshal(val, &valDec)
	return valDec.KeyType, valDec.Addr, valDec.Prv, err
}
