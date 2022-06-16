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
	"encoding/json"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/wcgcyx/fcr/io"
)

// handleQueryAddr handles query addr request.
//
// @input - network stream.
func (proto *AddrProtocol) handleQueryAddr(conn network.Stream) {
	defer conn.Close()
	// Read request.
	subCtx, cancel := context.WithTimeout(proto.routineCtx, proto.ioTimeout)
	defer cancel()
	data, err := io.Read(subCtx, conn)
	if err != nil {
		log.Debugf("Fail to read from %v: %v", conn, err.Error())
		return
	}
	type reqJson struct {
		CurrencyID byte     `json:"currency_id"`
		Secret     [32]byte `json:"secret"`
	}
	decoded := reqJson{}
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		log.Debugf("Fail to decode request %v: %v", data, err.Error())
		return
	}
	subCtx, cancel = context.WithTimeout(proto.routineCtx, proto.opTimeout)
	defer cancel()
	_, toAddr, err := proto.signer.GetAddr(subCtx, decoded.CurrencyID)
	if err != nil {
		log.Debugf("Fail to get addr for %v: %v", decoded.CurrencyID, err.Error())
		return
	}
	// Sign secret
	sigType, sig, err := proto.signer.Sign(subCtx, decoded.CurrencyID, decoded.Secret[:])
	if err != nil {
		log.Warnf("Fail to sign with key linked to currency id %v from signer: %v", decoded.CurrencyID, err.Error())
		return
	}
	type respJson struct {
		ToAddr    string `json:"to_addr"`
		SigType   byte   `json:"sig_type"`
		Signature []byte `json:"sig"`
	}
	data, err = json.Marshal(respJson{
		ToAddr:    toAddr,
		SigType:   sigType,
		Signature: sig,
	})
	if err != nil {
		log.Errorf("Fail to encode request: %v", err.Error())
		return
	}
	subCtx, cancel = context.WithTimeout(proto.routineCtx, proto.ioTimeout)
	defer cancel()
	err = io.Write(subCtx, conn, data)
	if err != nil {
		log.Debugf("Fail to respond %v: %v", conn, err.Error())
	}
}
