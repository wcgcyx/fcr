package payproto

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
	"math/big"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/wcgcyx/fcr/comms"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/peermgr"
)

// handleDirectPay handles a direct payment request.
//
// @input - network stream.
func (proto *PayProtocol) handleDirectPay(conn network.Stream) {
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
		OriginalAddr   string   `json:"original_addr"`
		OriginalSecret [32]byte `json:"original_secret"`
		OriginalAmt    *big.Int `json:"original_amt"`
		Voucher        string   `json:"voucher"`
	}
	decoded := reqJson{}
	err = json.Unmarshal(reqData, &decoded)
	if err != nil {
		respErr = fmt.Sprintf("Fail to decode request")
		log.Debugf("Fail to decode request: %v", err.Error())
		return
	}
	type respJson struct {
		NetworkLoss string `json:"network_loss"`
		SigType     byte   `json:"sig_type"`
		Signature   []byte `json:"signature"`
	}
	// This node is the final recipient.
	grossReceived, commit, revert, networkLoss, err := proto.payMgr.Receive(subCtx, req.CurrencyID, decoded.Voucher)
	if err != nil {
		if networkLoss != "" {
			// There is network loss.
			respData, err = json.Marshal(respJson{
				NetworkLoss: networkLoss,
			})
			if err != nil {
				respErr = fmt.Sprintf("Internal error")
				revert()
				log.Errorf("Error encoding resp data for request from %v-%v: %v", req.CurrencyID, req.FromAddr, err.Error())
				return
			}
			respStatus = true
			return
		}
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Fail to receive from %v-%v: %v", req.CurrencyID, req.FromAddr, err.Error())
		return
	}
	// No network loss.
	if grossReceived.Cmp(decoded.OriginalAmt) < 0 {
		respErr = fmt.Sprintf("Amount not sufficient")
		revert()
		log.Debugf("Amount received not sufficient, expect at least %v, got %v", decoded.OriginalAmt, grossReceived)
		return
	}
	// Sign a receipt back.
	sigType, sig, err := proto.signer.Sign(subCtx, req.CurrencyID, append(decoded.OriginalAmt.Bytes(), decoded.OriginalSecret[:]...))
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		revert()
		log.Warnf("Fail to sign receipt for request from %v-%v: %v", req.CurrencyID, req.FromAddr, err.Error())
		return
	}
	respData, err = json.Marshal(respJson{
		SigType:   sigType,
		Signature: sig,
	})
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		revert()
		log.Errorf("Error encoding resp data for request from %v-%v: %v", req.CurrencyID, req.FromAddr, err.Error())
		return
	}
	// Update received.
	release, err := proto.receivedLock.Lock(subCtx, req.CurrencyID, decoded.OriginalAddr)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		revert()
		log.Warnf("Error locking receiving map: %v", err.Error())
		return
	}
	defer release()
	_, ok := proto.received[req.CurrencyID]
	if !ok {
		proto.received[req.CurrencyID] = make(map[string]*big.Int)
		proto.receivedLock.Insert(req.CurrencyID, decoded.OriginalAddr)
	}
	_, ok = proto.received[req.CurrencyID][decoded.OriginalAddr]
	if !ok {
		proto.received[req.CurrencyID][decoded.OriginalAddr] = decoded.OriginalAmt
		proto.receivedLock.Insert(req.CurrencyID, decoded.OriginalAddr)
	} else {
		proto.received[req.CurrencyID][decoded.OriginalAddr].Add(proto.received[req.CurrencyID][decoded.OriginalAddr], decoded.OriginalAmt)
	}
	commit()
	respStatus = true
}

// handleProxyPay handles a proxy payment request.
//
// @input - network stream.
func (proto *PayProtocol) handleProxyPay(conn network.Stream) {
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
		log.Warnf("Error checking if peer exists for %v-%v: %v", req.CurrencyID, req.FromAddr, err.Error())
		return
	}
	if exists {
		// Check if peer is blocked.
		blocked, err := proto.peerMgr.IsBlocked(subCtx, req.CurrencyID, req.FromAddr)
		if err != nil {
			respErr = fmt.Sprintf("Internal error")
			log.Warnf("Fail to check if contains peer %v-%v: %v", req.CurrencyID, req.FromAddr, err.Error())
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
		OriginalAddr   string   `json:"original_addr"`
		OriginalSecret [32]byte `json:"original_secret"`
		OriginalAmt    *big.Int `json:"original_amt"`
		OfferData      []byte   `json:"offer_data"`
		Voucher        string   `json:"voucher"`
	}
	decoded := reqJson{}
	err = json.Unmarshal(reqData, &decoded)
	if err != nil {
		respErr = fmt.Sprintf("Fail to decode request")
		log.Debugf("Fail to decode request: %v", err.Error())
		return
	}
	sentOffer := fcroffer.PayOffer{}
	err = sentOffer.Decode(decoded.OfferData)
	if err != nil {
		respErr = fmt.Sprintf("Fail to decode offer")
		log.Debugf("Fail to decode offer from %v-%v: %v", req.CurrencyID, req.FromAddr, err.Error())
		return
	}
	offerData, err := sentOffer.GetToBeSigned()
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Errorf("Fail to get signing data for offer: %v", err.Error())
		return
	}
	err = crypto.Verify(sentOffer.CurrencyID, offerData, sentOffer.SignatureType, sentOffer.Signature, sentOffer.SrcAddr)
	if err != nil {
		respErr = fmt.Sprintf("Offer fail to verify")
		log.Debugf("Fail to verify signature: %v", err.Error())
		return
	}
	// Check offer
	if sentOffer.CurrencyID != req.CurrencyID {
		respErr = fmt.Sprintf("offer currency id mismatch")
		log.Debugf("Invalid offer, expect currency id %v, got %v", req.CurrencyID, sentOffer.CurrencyID)
		return
	}
	if sentOffer.SrcAddr != req.ToAddr {
		respErr = fmt.Sprintf("offer src addr mismatch")
		log.Debugf("Invalid offer, expect src addr %v, got %v", req.ToAddr, sentOffer.SrcAddr)
		return
	}
	type respJson struct {
		NetworkLoss string `json:"network_loss"`
		SigType     byte   `json:"sig_type"`
		Signature   []byte `json:"signature"`
	}
	// This node is the intermediate recipient.
	grossReceived, commit, revert, networkLoss, err := proto.payMgr.Receive(subCtx, req.CurrencyID, decoded.Voucher)
	if err != nil {
		if networkLoss != "" {
			// There is network loss.
			respData, err = json.Marshal(respJson{
				NetworkLoss: networkLoss,
			})
			if err != nil {
				respErr = fmt.Sprintf("Internal error")
				revert()
				log.Errorf("Error encoding resp data for request from %v-%v: %v", req.CurrencyID, req.FromAddr, err.Error())
				return
			}
			respStatus = true
			return
		}
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Fail to receive from %v-%v: %v", req.CurrencyID, req.FromAddr, err.Error())
		return
	}
	go func() {
		rec := peermgr.Record{
			Description: fmt.Sprintf("peer requested a use of reservation of %v for %v-%v", decoded.OriginalAmt, sentOffer.ResCh, sentOffer.ResID),
			CreatedAt:   time.Now(),
		}
		err := proto.peerMgr.AddToHistory(proto.routineCtx, req.CurrencyID, req.FromAddr, rec)
		if err != nil {
			log.Warnf("Fail add %v to history of %v-%v: %v", rec, req.CurrencyID, req.FromAddr, err.Error())
		}
	}()
	// No network loss
	var sigType byte
	var sig []byte
	if sentOffer.DestAddr == sentOffer.ResTo {
		// This is the final node
		sigType, sig, err = proto.payForOthersFinal(proto.routineCtx, sentOffer, grossReceived, decoded.OriginalAmt, decoded.OriginalAddr, decoded.OriginalSecret)
	} else {
		// This is not the final node
		sigType, sig, err = proto.payForOthersIntermediate(proto.routineCtx, sentOffer, grossReceived, decoded.OriginalAmt, decoded.OriginalAddr, decoded.OriginalSecret)
	}
	if err != nil {
		respErr = fmt.Sprintf("Proxy payment error")
		revert()
		log.Warnf("Error proxy paying: %v", err.Error())
		return
	}
	respData, err = json.Marshal(respJson{
		SigType:   sigType,
		Signature: sig,
	})
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		revert()
		log.Errorf("Error encoding resp data for request from %v-%v: %v", req.CurrencyID, req.FromAddr, err.Error())
		return
	}
	commit()
	respStatus = true
}
