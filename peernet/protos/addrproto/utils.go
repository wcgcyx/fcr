package addrproto

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
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

// getAddrRecord is used to get the addr record of given to addr and currency id.
//
// @input - to address, currency id.
//
// @output - cid, error.
func getAddrRecord(toAddr string, currencyID byte) (cid.Cid, error) {
	pref := cid.Prefix{
		Version:  0,
		Codec:    cid.DagProtobuf,
		MhType:   multihash.SHA2_256,
		MhLength: -1,
	}
	return pref.Sum(append([]byte(toAddr), currencyID))
}
