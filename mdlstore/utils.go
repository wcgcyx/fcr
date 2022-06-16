package mdlstore

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
	"strings"

	"github.com/ipfs/go-datastore"
)

// getDSKey gets the datastore key for a given path.
//
// @input - path.
//
// @output - datastore key, error.
func getDSKey(path ...interface{}) (datastore.Key, error) {
	mdKey := ""
	for _, sub := range path {
		key := fmt.Sprintf("%v", sub)
		if strings.Contains(key, DatastoreKeySeperator) {
			return datastore.Key{}, fmt.Errorf("%v contains illegal character %v", key, DatastoreKeySeperator)
		}
		mdKey += key + DatastoreKeySeperator
	}
	if mdKey != "" {
		// Get rid of extra separator in the end.
		mdKey = mdKey[:len(mdKey)-1]
	}
	return datastore.NewKey(mdKey), nil
}

// encDSVal encodes the data and children to the ds value.
//
// @input - data, children.
//
// @output - ds value, error.
func encDSVal(data []byte, children map[string]bool) ([]byte, error) {
	type valJson struct {
		Data     []byte          `json:"data"`
		Children map[string]bool `json:"children"`
	}
	return json.Marshal(valJson{
		Data:     data,
		Children: children,
	})
}

// decDSVal decodes the ds value into the node's data and children.
//
// @input - ds value.
//
// @output - data, children, error.
func decDSVal(val []byte) ([]byte, map[string]bool, error) {
	type valJson struct {
		Data     []byte          `json:"data"`
		Children map[string]bool `json:"children"`
	}
	valDec := valJson{}
	err := json.Unmarshal(val, &valDec)
	return valDec.Data, valDec.Children, err
}
