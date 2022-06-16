package brefstore

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
// @input - size, path, offset, trailer length.
//
// @output - value, error.
func encDSVal(size uint64, path string, offset uint64, trailerLen uint32) ([]byte, error) {
	type valJson struct {
		Size       uint64 `json:"size"`
		Path       string `json:"path"`
		Offset     uint64 `json:"offset"`
		TrailerLen uint32 `json:"trailer"`
	}
	return json.Marshal(valJson{
		Size:       size,
		Path:       path,
		Offset:     offset,
		TrailerLen: trailerLen,
	})
}

// decDSVal decodes the ds value into data.
//
// @input - value.
//
// @output - size, path, offset, trailer length, error.
func decDSVal(val []byte) (uint64, string, uint64, uint32, error) {
	type valJson struct {
		Size       uint64 `json:"size"`
		Path       string `json:"path"`
		Offset     uint64 `json:"offset"`
		TrailerLen uint32 `json:"trailer"`
	}
	valDec := valJson{}
	err := json.Unmarshal(val, &valDec)
	if err != nil {
		return 0, "", 0, 0, err
	}
	return valDec.Size, valDec.Path, valDec.Offset, valDec.TrailerLen, nil
}
