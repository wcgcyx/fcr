package paychproto

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

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/wcgcyx/fcr/comms"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/fcroffer"
)

// handleQueryAdd handles the query add request.
//
// @input - network stream.
func (proto *PaychProtocol) handleQueryAdd(conn network.Stream) {
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
	_, err = req.Receive(proto.routineCtx, proto.ioTimeout)
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
	// Get settlement policy
	settlement, err := proto.settleMgr.GetSenderPolicy(subCtx, req.CurrencyID, req.FromAddr)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Fail to get policy from settlement manager for %v-%v: %v", req.CurrencyID, req.FromAddr, err.Error())
		return
	}
	if settlement == 0 {
		respErr = fmt.Sprintf("Settlement not supported for %v-%v", req.CurrencyID, req.FromAddr)
		log.Debugf("Policy for %v-%v is 0, stop processing request", req.CurrencyID, req.FromAddr)
		return
	}
	nonce, err := proto.offerMgr.GetNonce(subCtx)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Error getting nonce from offer manager: %v", err.Error())
		return
	}
	// Create offer
	offer := fcroffer.PaychOffer{
		CurrencyID: req.CurrencyID,
		FromAddr:   req.FromAddr,
		ToAddr:     req.ToAddr,
		Settlement: time.Now().Add(settlement),
		Expiration: time.Now().Add(proto.offerExpiry),
		Nonce:      nonce,
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
		log.Warnf("Error signing paych offer for %v: %v", req.CurrencyID, err.Error())
		return
	}
	offer.AddSignature(sigType, sig)
	// Create response.
	respData, err = offer.Encode()
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Errorf("Error encoding pay offer for %v: %v", req.CurrencyID, err.Error())
		return
	}
	respStatus = true
}

// handleAdd handles the add request.
//
// @input - network stream.
func (proto *PaychProtocol) handleAdd(conn network.Stream) {
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
		ChAddr    string `json:"ch_addr"`
		OfferData []byte `json:"offer_data"`
	}
	decoded := reqJson{}
	err = json.Unmarshal(reqData, &decoded)
	if err != nil {
		respErr = fmt.Sprintf("Fail to decode request")
		log.Debugf("Fail to decode request: %v", err.Error())
		return
	}
	offer := fcroffer.PaychOffer{}
	err = offer.Decode(decoded.OfferData)
	if err != nil {
		respErr = fmt.Sprintf("Fail to decode request")
		log.Debugf("Fail to decode offer in request from %v-%v: %v", req.CurrencyID, req.FromAddr, err.Error())
		return
	}
	// Verify offer
	offerData, err := offer.GetToBeSigned()
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Errorf("Fail to get signing data for offer: %v", err.Error())
		return
	}
	err = crypto.Verify(offer.CurrencyID, offerData, offer.SignatureType, offer.Signature, offer.ToAddr)
	if err != nil {
		respErr = fmt.Sprintf("Offer fail to verify")
		log.Debugf("Fail to verify signature: %v", err.Error())
		return
	}
	if offer.CurrencyID != req.CurrencyID {
		respErr = fmt.Sprintf("Invalid offer received")
		log.Debugf("Invalid offer received expect sent by %v-%v has currency id %v", req.CurrencyID, req.FromAddr, offer.CurrencyID)
		return
	} else if offer.FromAddr != req.FromAddr {
		respErr = fmt.Sprintf("Invalid offer received")
		log.Debugf("Invalid offer received expect sent by %v-%v has from addr %v", req.CurrencyID, req.FromAddr, offer.FromAddr)
		return
	} else if offer.ToAddr != req.ToAddr {
		respErr = fmt.Sprintf("Invalid offer received")
		log.Debugf("Invalid offer received expect sent by %v-%v expect %v, got %v", req.CurrencyID, req.FromAddr, req.ToAddr, offer.ToAddr)
		return
	} else if time.Now().After(offer.Expiration) {
		respErr = fmt.Sprintf("Offer expired")
		log.Debugf("Received expired offer from %v-%v", req.CurrencyID, req.FromAddr)
		return
	} else if time.Now().After(offer.Settlement) {
		respErr = fmt.Sprintf("Offer settlement expired")
		log.Debugf("Received expired offer settlement from %v-%v", req.CurrencyID, req.FromAddr)
		return
	}
	// Add to payment manager
	err = proto.payMgr.AddInboundChannel(subCtx, req.CurrencyID, req.FromAddr, decoded.ChAddr)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Fail to add paych of %v from %v-%v", decoded.ChAddr, req.CurrencyID, req.FromAddr)
		return
	}
	err = proto.monitor.Track(subCtx, false, req.CurrencyID, decoded.ChAddr, offer.Settlement)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Fail to track paych of %v from %v-%v", decoded.ChAddr, req.CurrencyID, req.FromAddr)
		return
	}
	// Create response.
	respStatus = true
	// Do publish immediately
	go func() {
		proto.publishHookLock.RLock()
		defer proto.publishHookLock.RUnlock()
		if proto.publishHook != nil {
			go proto.publishHook(req.CurrencyID, req.FromAddr)
		}
	}()
}

// handleRenew handles the renew request.
//
// @input - network stream.
func (proto *PaychProtocol) handleRenew(conn network.Stream) {
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
	chAddr := string(reqData)
	// Check the current settlement
	updatedAt, settlement, err := proto.monitor.Check(subCtx, false, req.CurrencyID, chAddr)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Error checking current settlement for %v-%v-%v: %v", req.CurrencyID, req.FromAddr, chAddr, err.Error())
		return
	}
	// Check if in renew window
	now := time.Now()
	if uint64((now.Unix()-updatedAt.Unix())*100/(settlement.Unix()-updatedAt.Unix())) < proto.renewWindow {
		respErr = fmt.Sprintf("Does not meet renew window of %v - updated at %v settlement %v", proto.renewWindow, updatedAt, settlement)
		log.Debugf("Renew for %v-%v-%v does not meet renew window of %v", req.CurrencyID, req.FromAddr, chAddr, proto.renewWindow)
		return
	}
	// Get renew policy
	renew, err := proto.renewMgr.GetPaychPolicy(subCtx, req.CurrencyID, req.FromAddr, chAddr)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Error getting policy from renew manager for %v-%v-%v: %v", req.CurrencyID, req.FromAddr, chAddr, err.Error())
		return
	}
	if renew == 0 {
		respErr = fmt.Sprintf("Renew not supported")
		log.Debugf("Policy for %v-%v-%v is 0, stop processing request", req.CurrencyID, req.FromAddr, chAddr)
		return
	}
	newSettlement := now.Add(renew)
	if newSettlement.Before(settlement) {
		respErr = fmt.Sprintf("Renew %v does not exceed current settlement time %v", renew, settlement)
		log.Debugf("Renew %v for %v-%v-%v right now does not exceed current settlement time of %v", renew, req.CurrencyID, req.FromAddr, chAddr, settlement)
		return
	}
	err = proto.monitor.Renew(subCtx, false, req.CurrencyID, chAddr, newSettlement)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Error renew for %v-%v-%v: %v", req.CurrencyID, req.FromAddr, chAddr, err.Error())
		return
	}
	// Create response.
	respData, err = newSettlement.MarshalJSON()
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Errorf("Error encoding resp data for renewal of %v-%v-%v: %v", req.CurrencyID, req.FromAddr, chAddr, err.Error())
		return
	}
	respStatus = true
}
