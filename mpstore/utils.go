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

import "encoding/json"

// encDSVal encodes the data to the ds value.
//
// @input - miner key type, miner address, proof.
//
// @output - value, error.
func encDSVal(minerKeyType byte, minerAddr string, proof []byte) ([]byte, error) {
	type valJson struct {
		KeyType   byte   `json:"key_type"`
		MinerAddr string `json:"miner_addr"`
		Proof     []byte `json:"proof"`
	}
	return json.Marshal(valJson{
		KeyType:   minerKeyType,
		MinerAddr: minerAddr,
		Proof:     proof,
	})
}

// decDSVal decodes the ds value into data.
//
// @input - value.
//
// @output - miner key type, miner address, proof, error.
func decDSVal(val []byte) (byte, string, []byte, error) {
	type valJson struct {
		KeyType   byte   `json:"key_type"`
		MinerAddr string `json:"miner_addr"`
		Proof     []byte `json:"proof"`
	}
	valDec := valJson{}
	err := json.Unmarshal(val, &valDec)
	if err != nil {
		return 0, "", nil, err
	}
	return valDec.KeyType, valDec.MinerAddr, valDec.Proof, nil
}
