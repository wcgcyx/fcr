package paychmon

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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/wcgcyx/fcr/mdlstore"
)

// decDSKey is the ds key decoder.
//
// @input - key.
//
// @output - number of sub keys, outbound, currency id, chAddr, error.
func decDSKey(key string) (int, bool, byte, string, error) {
	res := strings.Split(key, mdlstore.DatastoreKeySeperator)
	outbound := false
	currencyID := byte(0)
	chAddr := ""
	if len(res) == 0 {
		return 0, false, 0, "", fmt.Errorf("invalid ds key")
	}
	if len(res) >= 1 {
		if res[0] == "true" {
			outbound = true
		} else if res[0] == "false" {
			outbound = false
		} else {
			return 0, false, 0, "", fmt.Errorf("fail to parse %v", res[0])
		}
	}
	if len(res) >= 2 {
		cur, err := strconv.Atoi(res[1])
		if err != nil {
			return 0, false, 0, "", fmt.Errorf("fail to parse %v", res[1])
		}
		currencyID = byte(cur)
	}
	if len(res) == 3 {
		chAddr = res[2]
	}
	if len(res) > 3 {
		return 0, false, 0, "", fmt.Errorf("invalid ds key")
	}
	return len(res), outbound, currencyID, chAddr, nil
}

// encDSVal encodes the data to the ds value.
//
// @input - updatedAt, settlement.
//
// @output - value, error.
func encDSVal(updatedAt time.Time, settlement time.Time) ([]byte, error) {
	type valJson struct {
		UpdatedAt  time.Time `json:"updated_at"`
		Settlement time.Time `json:"settlement"`
	}
	return json.Marshal(valJson{
		UpdatedAt:  updatedAt,
		Settlement: settlement,
	})
}

// decDSVal decodes the ds value into data.
//
// @input - value.
//
// @output - updatedAt, settlement, error.
func decDSVal(val []byte) (time.Time, time.Time, error) {
	type valJson struct {
		UpdatedAt  time.Time `json:"updated_at"`
		Settlement time.Time `json:"settlement"`
	}
	valDec := valJson{}
	err := json.Unmarshal(val, &valDec)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return valDec.UpdatedAt, valDec.Settlement, nil
}
