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
	"context"
	"crypto/rand"
	"encoding/json"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/io"
)

// QueryAddr is used to query the recipient address of given currency id for a given peer address.
//
// @input - context, currency id, peer address.
//
// @output - recipient address, error.
func (proto *AddrProtocol) QueryAddr(ctx context.Context, currencyID byte, pi peer.AddrInfo) (string, error) {
	subCtx, cancel := context.WithTimeout(ctx, proto.ioTimeout)
	defer cancel()
	err := proto.h.Connect(subCtx, pi)
	if err != nil {
		log.Debugf("Fail to connect to %v: %v", pi, err.Error())
		return "", err
	}
	conn, err := proto.h.NewStream(subCtx, pi.ID, QueryAddrProtocol+ProtocolVersion)
	if err != nil {
		log.Debugf("Fail to open stream to %v: %v", pi, err.Error())
		return "", err
	}
	defer conn.Close()
	var secret [32]byte
	_, err = rand.Read(secret[:])
	if err != nil {
		log.Debugf("Fail to read secret: %v", err.Error())
		return "", err
	}
	type reqJson struct {
		CurrencyID byte     `json:"currency_id"`
		Secret     [32]byte `json:"secret"`
	}
	data, err := json.Marshal(reqJson{
		CurrencyID: currencyID,
		Secret:     secret,
	})
	if err != nil {
		log.Errorf("Fail to encode request: %v", err.Error())
		return "", err
	}
	err = io.Write(subCtx, conn, data)
	if err != nil {
		log.Debugf("Fail to send request: %v", err.Error())
		return "", err
	}
	data, err = io.Read(subCtx, conn)
	if err != nil {
		log.Debugf("Fail to receive response: %v", err.Error())
		return "", err
	}
	// Decode data
	type respJson struct {
		ToAddr    string `json:"to_addr"`
		SigType   byte   `json:"sig_type"`
		Signature []byte `json:"sig"`
	}
	decoded := respJson{}
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		log.Debugf("Fail to decode response: %v", err.Error())
		return "", err
	}
	// Verify signature
	err = crypto.Verify(currencyID, secret[:], decoded.SigType, decoded.Signature, decoded.ToAddr)
	if err != nil {
		log.Debugf("Fail to verify signature: %v", err.Error())
		return "", err
	}
	// Upsert the to address.
	subCtx, cancel = context.WithTimeout(ctx, proto.opTimeout)
	defer cancel()
	err = proto.peerMgr.AddPeer(subCtx, currencyID, decoded.ToAddr, pi)
	if err != nil {
		log.Warnf("Fail to add peer %v-%v with %v: %v", currencyID, decoded.ToAddr, pi, err.Error())
		return "", err
	}
	return decoded.ToAddr, nil
}
