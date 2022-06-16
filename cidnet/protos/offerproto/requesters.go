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

	"github.com/ipfs/go-cid"
	"github.com/wcgcyx/fcr/comms"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/mpstore"
)

// QueryOffer queries for a piece offer for a given cid, currency id and peer id.
//
// @input - context, piece cid, currency id, peer id.
//
// @output - piece offer, error.
func (proto *OfferProtocol) QueryOffer(ctx context.Context, currencyID byte, toAddr string, root cid.Cid) (fcroffer.PieceOffer, error) {
	// Check if peer is blocked.
	subCtx, cancel := context.WithTimeout(ctx, proto.opTimeout)
	defer cancel()
	exists, err := proto.peerMgr.HasPeer(subCtx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to check if contains peer %v-%v: %v", currencyID, toAddr, err.Error())
		return fcroffer.PieceOffer{}, err
	}
	if exists {
		blocked, err := proto.peerMgr.IsBlocked(subCtx, currencyID, toAddr)
		if err != nil {
			log.Warnf("Fail to check if peer %v-%v is blocked: %v", currencyID, toAddr, err.Error())
			return fcroffer.PieceOffer{}, err
		}
		if blocked {
			return fcroffer.PieceOffer{}, fmt.Errorf("peer %v-%v has been blocked", currencyID, toAddr)
		}
	}
	// Connect to peer
	pid, err := proto.addrProto.ConnectToPeer(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to connect to %v-%v: %v", currencyID, toAddr, err.Error())
		return fcroffer.PieceOffer{}, err
	}
	subCtx, cancel = context.WithTimeout(ctx, proto.ioTimeout)
	defer cancel()
	conn, err := proto.h.NewStream(subCtx, pid, QueryOfferProtocol+ProtocolVersion)
	if err != nil {
		log.Debugf("Fail open stream to %v: %v", pid, err.Error())
		return fcroffer.PieceOffer{}, err
	}
	defer conn.Close()
	// Create request.
	req, err := comms.NewRequestOut(ctx, proto.opTimeout, proto.ioTimeout, conn, proto.signer, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to create request: %v", err.Error())
		return fcroffer.PieceOffer{}, err
	}
	// Send request
	type reqJson struct {
		Root cid.Cid `json:"root"`
	}
	data, err := json.Marshal(reqJson{
		Root: root,
	})
	if err != nil {
		log.Errorf("Fail to encode request: %v", err.Error())
		return fcroffer.PieceOffer{}, err
	}
	err = req.Send(ctx, proto.opTimeout, proto.ioTimeout, data)
	if err != nil {
		log.Debugf("Fail to send request: %v", err.Error())
		return fcroffer.PieceOffer{}, err
	}
	// Get response
	succeed, data, err := req.GetResponse(ctx, proto.ioTimeout)
	if err != nil {
		log.Debugf("Fail to receive response: %v", err.Error())
		return fcroffer.PieceOffer{}, err
	}
	if !succeed {
		return fcroffer.PieceOffer{}, fmt.Errorf("peer fail to process request: %v", string(data))
	}
	// Decode data
	offer := fcroffer.PieceOffer{}
	err = offer.Decode(data)
	if err != nil {
		log.Debugf("Fail to decode response: %v", err.Error())
		return fcroffer.PieceOffer{}, err
	}
	// Verify offer
	offerData, err := offer.GetToBeSigned()
	if err != nil {
		log.Errorf("Fail to get signing data for offer: %v", err.Error())
		return fcroffer.PieceOffer{}, err
	}
	err = crypto.Verify(offer.CurrencyID, offerData, offer.SignatureType, offer.Signature, offer.RecipientAddr)
	if err != nil {
		log.Debugf("Fail to verify signature: %v", err.Error())
		return fcroffer.PieceOffer{}, err
	}
	if offer.LinkedMinerAddr != "" {
		// Check offer miner proof
		err = mpstore.VerifyMinerProof(currencyID, offer.RecipientAddr, offer.LinkedMinerKeyType, offer.LinkedMinerAddr, offer.LinkedMinerProof)
		if err != nil {
			log.Debugf("Fail to verify miner proof: %v", err.Error())
			return fcroffer.PieceOffer{}, err
		}
	}
	// Check offer
	if offer.CurrencyID != currencyID {
		return fcroffer.PieceOffer{}, fmt.Errorf("Invalid offer, expect currency id %v, got %v", currencyID, offer.CurrencyID)
	}
	if !offer.ID.Equals(root) {
		return fcroffer.PieceOffer{}, fmt.Errorf("Invalid offer, expect root %v, got %v", root, offer.ID)
	}
	if offer.PPB.Cmp(big.NewInt(0)) < 0 {
		return fcroffer.PieceOffer{}, fmt.Errorf("Invalid offer, expect non-negative ppb got %v", offer.PPB)
	}
	if offer.RecipientAddr != toAddr {
		return fcroffer.PieceOffer{}, fmt.Errorf("Invalid offer, expect recipient to be %v got %v", toAddr, offer.RecipientAddr)
	}
	if time.Now().After(offer.Expiration) {
		return fcroffer.PieceOffer{}, fmt.Errorf("Invalid offer, expiration in the past")
	}
	return offer, nil
}
