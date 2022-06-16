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

	"github.com/wcgcyx/fcr/comms"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/peermgr"
)

// QueryOffer queries for a payment offer for a given route.
//
// @input - context, currency id, route, amt required.
//
// @output - payment offer, error.
func (proto *OfferProtocol) QueryOffer(ctx context.Context, currencyID byte, route []string, amt *big.Int) (fcroffer.PayOffer, error) {
	if len(route) < 2 {
		return fcroffer.PayOffer{}, fmt.Errorf("route must be at least 2 in length")
	}
	toAddr := route[0]
	// Check if peer is blocked.
	subCtx, cancel := context.WithTimeout(ctx, proto.opTimeout)
	defer cancel()
	exists, err := proto.peerMgr.HasPeer(subCtx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to check if contains peer %v-%v: %v", currencyID, toAddr, err.Error())
		return fcroffer.PayOffer{}, err
	}
	if exists {
		blocked, err := proto.peerMgr.IsBlocked(subCtx, currencyID, toAddr)
		if err != nil {
			log.Warnf("Fail to check if peer %v-%v is blocked: %v", currencyID, toAddr, err.Error())
			return fcroffer.PayOffer{}, err
		}
		if blocked {
			return fcroffer.PayOffer{}, fmt.Errorf("peer %v-%v has been blocked", currencyID, toAddr)
		}
	}
	// Connect to peer
	pid, err := proto.addrProto.ConnectToPeer(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to connect to %v-%v: %v", currencyID, toAddr, err.Error())
		return fcroffer.PayOffer{}, err
	}
	subCtx, cancel = context.WithTimeout(ctx, proto.ioTimeout)
	defer cancel()
	conn, err := proto.h.NewStream(subCtx, pid, QueryOfferProtocol+ProtocolVersion)
	if err != nil {
		log.Debugf("Fail open stream to %v: %v", pid, err.Error())
		return fcroffer.PayOffer{}, err
	}
	defer conn.Close()
	// Create request.
	req, err := comms.NewRequestOut(ctx, proto.opTimeout, proto.ioTimeout, conn, proto.signer, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to create request: %v", err.Error())
		return fcroffer.PayOffer{}, err
	}
	// Send request
	type reqJson struct {
		Route []string `json:"route"`
		Amt   *big.Int `json:"amt"`
	}
	data, err := json.Marshal(reqJson{
		Route: route,
		Amt:   amt,
	})
	if err != nil {
		log.Errorf("Fail to encode request: %v", err.Error())
		return fcroffer.PayOffer{}, err
	}
	err = req.Send(ctx, proto.opTimeout, proto.ioTimeout, data)
	if err != nil {
		log.Debugf("Fail to send request: %v", err.Error())
		return fcroffer.PayOffer{}, err
	}
	// Get response
	succeed, data, err := req.GetResponse(ctx, proto.ioTimeout)
	if err != nil {
		log.Debugf("Fail to receive response: %v", err.Error())
		return fcroffer.PayOffer{}, err
	}
	if !succeed {
		return fcroffer.PayOffer{}, fmt.Errorf("Peer fail to process request: %v", string(data))
	}
	// Decode data
	offer := fcroffer.PayOffer{}
	err = offer.Decode(data)
	if err != nil {
		log.Debugf("Fail to decode response: %v", err.Error())
		return fcroffer.PayOffer{}, err
	}
	// Verify offer
	offerData, err := offer.GetToBeSigned()
	if err != nil {
		log.Errorf("Fail to get signing data for offer: %v", err.Error())
		return fcroffer.PayOffer{}, err
	}
	err = crypto.Verify(offer.CurrencyID, offerData, offer.SignatureType, offer.Signature, offer.SrcAddr)
	if err != nil {
		log.Debugf("Fail to verify signature: %v", err.Error())
		return fcroffer.PayOffer{}, err
	}
	// Check offer
	if offer.CurrencyID != currencyID {
		return fcroffer.PayOffer{}, fmt.Errorf("Invalid offer, expect currency id %v, got %v", currencyID, offer.CurrencyID)
	}
	if offer.SrcAddr != route[0] {
		return fcroffer.PayOffer{}, fmt.Errorf("Invalid offer, expect src addr %v, got %v", route[0], offer.SrcAddr)
	}
	if offer.DestAddr != route[len(route)-1] {
		return fcroffer.PayOffer{}, fmt.Errorf("Invalid offer, expect dest addr %v, got %v", route[len(route)-1], offer.DestAddr)
	}
	if offer.PPP.Cmp(big.NewInt(0)) < 0 {
		return fcroffer.PayOffer{}, fmt.Errorf("Invalid offer, expect non-negative ppp got %v", offer.PPP)
	}
	if offer.Period.Cmp(big.NewInt(0)) < 0 {
		return fcroffer.PayOffer{}, fmt.Errorf("Invalid offer, expect non-negative period got %v", offer.Period)
	}
	if !((offer.PPP.Cmp(big.NewInt(0)) == 0 && offer.Period.Cmp(big.NewInt(0)) == 0) || (offer.PPP.Cmp(big.NewInt(0)) > 0 && offer.Period.Cmp(big.NewInt(0)) > 0)) {
		return fcroffer.PayOffer{}, fmt.Errorf("ppp and period must be either both 0 or both positive, got ppp %v, period %v", offer.PPP, offer.Period)
	}
	if offer.Amt.Cmp(amt) < 0 {
		return fcroffer.PayOffer{}, fmt.Errorf("Invalid offer, expect minimum amount of %v got %v", amt, offer.Amt)
	}
	if time.Now().After(offer.Expiration) {
		return fcroffer.PayOffer{}, fmt.Errorf("Invalid offer, expiration in the past")
	}
	go func() {
		rec := peermgr.Record{
			Description: fmt.Sprintf("peer succeed to process pay offer query request"),
			CreatedAt:   time.Now(),
		}
		err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
		if err != nil {
			log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
		}
	}()
	return offer, nil
}
