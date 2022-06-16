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
	"math/big"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/wcgcyx/fcr/comms"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/peermgr"
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
		Route []string `json:"route"`
		Amt   *big.Int `json:"amt"`
	}
	decoded := reqJson{}
	err = json.Unmarshal(reqData, &decoded)
	if err != nil {
		respErr = fmt.Sprintf("Fail to decode request")
		log.Debugf("Fail to decode request: %v", err.Error())
		return
	}
	route := decoded.Route
	amt := decoded.Amt
	if len(route) < 2 {
		respErr = fmt.Sprintf("Route must be at least 2 in length")
		log.Debugf("Route in request does not have at least 2 in length got %v", len(route))
		return
	}
	maxHop, err := proto.rs.MaxHop(subCtx, req.CurrencyID)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Fail to get max hop for %v: %v", req.CurrencyID, err.Error())
		return
	}
	if uint64(len(route)) > maxHop {
		respErr = fmt.Sprintf("Route too long, not supported")
		log.Debugf("Route in request does not have at most %v in length got %v", maxHop, len(route))
		return
	}
	if route[0] != req.ToAddr {
		respErr = fmt.Sprintf("Invalid route received")
		log.Debugf("Invalid route received expect sent by %v-%v expect start from %v, got %v", req.CurrencyID, req.FromAddr, req.ToAddr, route[0])
		return
	}
	// Check if the next node in route is currently served.
	served := false
	subsubCtx, cancel := context.WithCancel(subCtx)
	defer cancel()
	chanOut, errOut := proto.pservMgr.ListServings(subsubCtx, req.CurrencyID, route[1])
	for range chanOut {
		cancel()
		served = true
		break
	}
	err = <-errOut
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Fail to list servings for %v-%v: %v", req.CurrencyID, route[1], err.Error())
		return
	}
	if !served {
		respErr = fmt.Sprintf("No active serving found for route")
		log.Debugf("No active serving found for %v-%v", req.CurrencyID, route[1])
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
	offer := &fcroffer.PayOffer{
		CurrencyID: req.CurrencyID,
		SrcAddr:    req.ToAddr,
		DestAddr:   route[len(route)-1],
		Amt:        amt,
		Expiration: expiration,
		Inactivity: proto.offerInactivity,
		Nonce:      nonce,
	}
	// Check if this is the final node in serving.
	if len(route) == 2 {
		// This is the final node.
		ppp, period, resCh, resID, err := proto.payMgr.ReserveForOthersFinal(subCtx, req.CurrencyID, route[1], amt, expiration, proto.offerInactivity, req.FromAddr)
		if err != nil {
			respErr = fmt.Sprintf("Internal error")
			log.Warnf("Fail to reserve for %v-%v with %v: %v", req.CurrencyID, route[1], amt, err.Error())
			return
		}
		offer.PPP = ppp
		offer.Period = period
		offer.ResTo = route[1]
		offer.ResCh = resCh
		offer.ResID = resID
	} else {
		// This is not the final node, query the next node
		receivedOffer, err := proto.QueryOffer(proto.routineCtx, req.CurrencyID, route[1:], amt)
		if err != nil {
			respErr = fmt.Sprintf("Error query offer from next node in route")
			log.Debugf("Fail to query offer from %v-%v with %v: %v", req.CurrencyID, route[1], amt, err.Error())
			return
		}
		subCtx, cancel = context.WithTimeout(proto.routineCtx, proto.opTimeout)
		defer cancel()
		ppp, period, resCh, resID, err := proto.payMgr.ReserveForOthersIntermediate(subCtx, receivedOffer, req.FromAddr)
		if err != nil {
			respErr = fmt.Sprintf("Internal error")
			log.Warnf("Fail to reserve with received offer from %v-%v: %v", req.CurrencyID, route[1], err.Error())
			return
		}
		offer.PPP = ppp
		offer.Period = period
		offer.ResTo = route[1]
		offer.ResCh = resCh
		offer.ResID = resID
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
		log.Warnf("Error signing pay offer for %v: %v", req.CurrencyID, err.Error())
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
	go func() {
		rec := peermgr.Record{
			Description: fmt.Sprintf("peer requested a reservation of %v for %v-%v", amt, offer.ResCh, offer.ResID),
			CreatedAt:   time.Now(),
		}
		err := proto.peerMgr.AddToHistory(proto.routineCtx, req.CurrencyID, req.FromAddr, rec)
		if err != nil {
			log.Warnf("Fail add %v to history of %v-%v: %v", rec, req.CurrencyID, req.FromAddr, err.Error())
		}
	}()
	respStatus = true
}
