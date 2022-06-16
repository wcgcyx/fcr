package offerproto

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
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/wcgcyx/fcr/comms"
	"github.com/wcgcyx/fcr/fcroffer"
)

// handleQueryOffer handles the query offer request.
//
// @input - network stream.
func (proto *OfferProtocol) handleQueryOffer(conn network.Stream) {
	defer conn.Close()
	// Read request.
	req, err := comms.NewRequestIn(proto.routineCtx, proto.opTimeout, proto.ioTimeout, conn, proto.signer)
	if err != nil {
		log.Debugf("Fail to establish request from %v: %v", conn.ID(), err.Error())
		return
	}
	// Prepare response.
	var respStatus bool
	var respData []byte
	var respErr string
	defer func() {
		var data []byte
		if respStatus {
			data = respData
		} else {
			data = []byte(respErr)
		}
		err = req.Respond(proto.routineCtx, proto.opTimeout, proto.ioTimeout, respStatus, data)
		if err != nil {
			log.Debugf("Error sending response: %v", err.Error())
		}
	}()
	// Get request.
	reqData, err := req.Receive(proto.routineCtx, proto.ioTimeout)
	if err != nil {
		log.Debugf("Fail to receive request from stream %v: %v", conn.ID(), err.Error())
		return
	}
	// Start processing request.
	subCtx, cancel := context.WithTimeout(proto.routineCtx, proto.opTimeout)
	defer cancel()
	exists, err := proto.peerMgr.HasPeer(subCtx, req.CurrencyID, req.FromAddr)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Fail to check if contains peer %v-%v: %v", req.CurrencyID, req.FromAddr, err.Error())
		return
	}
	if exists {
		// Check if peer is blocked.
		blocked, err := proto.peerMgr.IsBlocked(subCtx, req.CurrencyID, req.FromAddr)
		if err != nil {
			respErr = fmt.Sprintf("Internal error")
			log.Warnf("Fail to check if peer %v-%v is blocked: %v", req.CurrencyID, req.FromAddr, err.Error())
			return
		}
		if blocked {
			respErr = fmt.Sprintf("Peer %v-%v is blocked", req.CurrencyID, req.FromAddr)
			log.Debugf("Peer %v-%v has been blocked, stop processing request", req.CurrencyID, req.FromAddr)
			return
		}
	} else {
		// Insert peer.
		pi := peer.AddrInfo{
			ID:    conn.Conn().RemotePeer(),
			Addrs: []multiaddr.Multiaddr{conn.Conn().RemoteMultiaddr()},
		}
		err = proto.peerMgr.AddPeer(subCtx, req.CurrencyID, req.FromAddr, pi)
		if err != nil {
			respErr = fmt.Sprintf("Internal error")
			log.Warnf("Fail to add peer %v-%v with %v: %v", req.CurrencyID, req.FromAddr, pi, err.Error())
			return
		}
	}
	// Decode request
	type reqJson struct {
		Root cid.Cid `json:"root"`
	}
	decoded := reqJson{}
	err = json.Unmarshal(reqData, &decoded)
	if err != nil {
		respErr = fmt.Sprintf("Fail to decode request")
		log.Debugf("Fail to decode request: %v", err.Error())
		return
	}
	root := decoded.Root
	// Check if this root is served at requested currency.
	exists, ppb, err := proto.cservMgr.Inspect(subCtx, root, req.CurrencyID)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Fail to inspect piece manager for %v-%v: %v", root.String(), req.CurrencyID, err.Error())
		return
	}
	if !exists {
		respErr = fmt.Sprintf("Not currently served")
		return
	}
	// Check if this piece is not removed.
	exists, _, _, size, _, err := proto.pieceMgr.Inspect(subCtx, root)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Fail to inspect piece manager for %v-%v: %v", root.String(), req.CurrencyID, err.Error())
		return
	}
	if !exists {
		respErr = fmt.Sprintf("Not currently served")
		go func() {
			err := proto.cservMgr.Stop(proto.routineCtx, root, req.CurrencyID)
			if err != nil {
				log.Warnf("Fail to stop serving piece for removed %v-%v: %v", root.String(), req.CurrencyID, err.Error())
			}
		}()
		return
	}
	// Get offer nonce for potential offer creation.
	nonce, err := proto.offerMgr.GetNonce(subCtx)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Error getting nonce from offer manager: %v", err.Error())
		return
	}
	expiration := time.Now().Add(proto.offerExpiry)
	offer := &fcroffer.PieceOffer{
		ID:            root,
		Size:          size,
		CurrencyID:    req.CurrencyID,
		PPB:           ppb,
		RecipientAddr: req.ToAddr,
		Expiration:    expiration,
		Inactivity:    proto.offerInactivity,
		Nonce:         nonce,
	}
	// Load miner proof
	exists, keyType, minerAddr, proof, err := proto.mps.GetMinerProof(subCtx, req.CurrencyID)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Fail to get miner proof for %v: %v", req.CurrencyID, err.Error())
		return
	}
	if exists {
		offer.LinkedMinerKeyType = keyType
		offer.LinkedMinerAddr = minerAddr
		offer.LinkedMinerProof = proof
	}
	// Sign offer
	data, err := offer.GetToBeSigned()
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Errorf("Fail to get signing data for offer: %v", err.Error())
		return
	}
	sigType, sig, err := proto.signer.Sign(subCtx, req.CurrencyID, data)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Error signing piece offer for %v: %v", req.CurrencyID, err.Error())
		return
	}
	offer.AddSignature(sigType, sig)
	// Create response.
	respData, err = offer.Encode()
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Errorf("Error encoding piece offer for %v-%v: %v", root.String(), req.CurrencyID, err.Error())
		return
	}
	respStatus = true
}
