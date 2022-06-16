package peermgr

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

	dsquery "github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p-core/peer"
)

// encDSVal encodes the data to the ds value.
//
// @input - if blocked, peer address, rec id.
//
// @output - value, error.
func encDSVal(blocked bool, peerAddr peer.AddrInfo, recID *big.Int) ([]byte, error) {
	type valJson struct {
		Blocked  bool          `json:"blocked"`
		PeerAddr peer.AddrInfo `json:"peer_addr"`
		RecID    *big.Int      `json:"rec_id"`
	}
	return json.Marshal(valJson{
		Blocked:  blocked,
		PeerAddr: peerAddr,
		RecID:    recID,
	})
}

// decDSVal decodes the ds value into data.
//
// @input - value.
//
// @output - if blocked, peer address, rec id, error.
func decDSVal(val []byte) (bool, peer.AddrInfo, *big.Int, error) {
	type valJson struct {
		Blocked  bool          `json:"blocked"`
		PeerAddr peer.AddrInfo `json:"peer_addr"`
		RecID    *big.Int      `json:"rec_id"`
	}
	valDec := valJson{}
	err := json.Unmarshal(val, &valDec)
	if err != nil {
		return false, peer.AddrInfo{}, nil, err
	}
	return valDec.Blocked, valDec.PeerAddr, valDec.RecID, nil
}

// history order is used to order history
type historyOrder struct {
}

// Compare compares two entries.
//
// @input - entry a, entry b.
//
// @output - 1 if a is newer than b, 0 if same, -1 if a is older than b.
func (o historyOrder) Compare(a, b dsquery.Entry) int {
	// Need to get the true value out of raw ds value.
	// This is copied from mdlstore.
	type valJson struct {
		Data     []byte          `json:"data"`
		Children map[string]bool `json:"children"`
	}
	valDec1 := valJson{}
	valDec2 := valJson{}
	err1 := json.Unmarshal(a.Value, &valDec1)
	err2 := json.Unmarshal(b.Value, &valDec2)
	if err1 != nil && err2 != nil {
		return 0
	} else if err1 != nil && err2 == nil {
		return 1
	} else if err1 == nil && err2 != nil {
		return -1
	}
	rec1, err1 := DecodeRecord(valDec1.Data)
	rec2, err2 := DecodeRecord(valDec2.Data)
	if err1 != nil && err2 != nil {
		return 0
	} else if err1 != nil && err2 == nil {
		return 1
	} else if err1 == nil && err2 != nil {
		return -1
	}
	if rec1.CreatedAt.Equal(rec2.CreatedAt) {
		return 0
	} else if rec1.CreatedAt.Before(rec2.CreatedAt) {
		return 1
	}
	return -1
}
