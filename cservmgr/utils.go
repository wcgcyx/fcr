package cservmgr

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
	"strconv"
	"strings"

	"github.com/wcgcyx/fcr/mdlstore"
)

// decDSKey is the ds key decoder.
//
// @input - key.
//
// @output - number of sub keys, key hash, currency id, error.
func decDSKey(key string) (int, string, byte, error) {
	res := strings.Split(key, mdlstore.DatastoreKeySeperator)
	keyHash := ""
	currencyID := byte(0)
	if len(res) == 0 {
		return 0, "", 0, fmt.Errorf("invalid ds key")
	}
	if len(res) >= 1 {
		keyHash = res[0]
	}
	if len(res) >= 2 {
		cur, err := strconv.Atoi(res[1])
		if err != nil {
			return 0, "", 0, fmt.Errorf("fail to parse %v", res[1])
		}
		currencyID = byte(cur)
	}
	if len(res) > 2 {
		return 0, "", 0, fmt.Errorf("invalid ds key")
	}
	return len(res), keyHash, currencyID, nil
}
