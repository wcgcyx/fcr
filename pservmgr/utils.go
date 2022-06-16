package pservmgr

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
	"math/big"
)

// encDSVal encodes the data to the ds value.
//
// @input - price-per-period, period.
//
// @output - value, error.
func encDSVal(ppp *big.Int, period *big.Int) ([]byte, error) {
	type valJson struct {
		PPP    *big.Int `json:"ppp"`
		Period *big.Int `json:"period"`
	}
	return json.Marshal(valJson{
		PPP:    ppp,
		Period: period,
	})
}

// decDSVal decodes the ds value into data.
//
// @input - value.
//
// @output - price-per-period, period, error.
func decDSVal(val []byte) (*big.Int, *big.Int, error) {
	type valJson struct {
		PPP    *big.Int `json:"ppp"`
		Period *big.Int `json:"period"`
	}
	valDec := valJson{}
	err := json.Unmarshal(val, &valDec)
	return valDec.PPP, valDec.Period, err
}
