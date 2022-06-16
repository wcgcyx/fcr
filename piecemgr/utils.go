package piecemgr

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
// @input - path, index, piece size, if copied.
//
// @output - value, error.
func encDSVal(path string, index int, size uint64, copy bool) ([]byte, error) {
	type valJson struct {
		Path  string `json:"path"`
		Index int    `json:"index"`
		Size  uint64 `json:"size"`
		Copy  bool   `json:"copy"`
	}
	return json.Marshal(valJson{
		Path:  path,
		Index: index,
		Size:  size,
		Copy:  copy,
	})
}

// decDSVal decodes the ds value into data.
//
// @input - value.
//
// @output - path, index, piece size, if copied, error.
func decDSVal(val []byte) (string, int, uint64, bool, error) {
	type valJson struct {
		Path  string `json:"path"`
		Index int    `json:"index"`
		Size  uint64 `json:"size"`
		Copy  bool   `json:"copy"`
	}
	valDec := valJson{}
	err := json.Unmarshal(val, &valDec)
	if err != nil {
		return "", 0, 0, false, err
	}
	return valDec.Path, valDec.Index, valDec.Size, valDec.Copy, nil
}
