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

	"github.com/wcgcyx/fcr/comms"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/peermgr"
)

// QueryAdd queries a peer for a paych offer before creating and adding a payment channel.
//
// @input - context, currency id, recipient address.
//
// @output - paych offer, error.
func (proto *PaychProtocol) QueryAdd(ctx context.Context, currencyID byte, toAddr string) (fcroffer.PaychOffer, error) {
	// Check if peer is blocked.
	subCtx, cancel := context.WithTimeout(ctx, proto.opTimeout)
	defer cancel()
	exists, err := proto.peerMgr.HasPeer(subCtx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to check if contains peer %v-%v: %v", currencyID, toAddr, err.Error())
		return fcroffer.PaychOffer{}, err
	}
	if exists {
		blocked, err := proto.peerMgr.IsBlocked(subCtx, currencyID, toAddr)
		if err != nil {
			log.Warnf("Fail to check if peer %v-%v is blocked: %v", currencyID, toAddr, err.Error())
			return fcroffer.PaychOffer{}, err
		}
		if blocked {
			return fcroffer.PaychOffer{}, fmt.Errorf("peer %v-%v has been blocked", currencyID, toAddr)
		}
	}
	// Connect to peer
	pid, err := proto.addrProto.ConnectToPeer(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to connect to %v-%v: %v", currencyID, toAddr, err.Error())
		return fcroffer.PaychOffer{}, err
	}
	subCtx, cancel = context.WithTimeout(ctx, proto.ioTimeout)
	defer cancel()
	conn, err := proto.h.NewStream(subCtx, pid, QueryAddProtocol+ProtocolVersion)
	if err != nil {
		log.Debugf("Fail open stream to %v: %v", pid, err.Error())
		return fcroffer.PaychOffer{}, err
	}
	defer conn.Close()
	// Create request.
	req, err := comms.NewRequestOut(ctx, proto.opTimeout, proto.ioTimeout, conn, proto.signer, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to create request: %v", err.Error())
		return fcroffer.PaychOffer{}, err
	}
	// Send request
	err = req.Send(ctx, proto.opTimeout, proto.ioTimeout, []byte{})
	if err != nil {
		log.Debugf("Fail to send request: %v", err.Error())
		return fcroffer.PaychOffer{}, err
	}
	// Get response
	succeed, data, err := req.GetResponse(ctx, proto.ioTimeout)
	if err != nil {
		log.Debugf("Fail to receive response: %v", err.Error())
		return fcroffer.PaychOffer{}, err
	}
	if !succeed {
		return fcroffer.PaychOffer{}, fmt.Errorf("Peer fail to process request: %v", string(data))
	}
	// Decode data
	offer := fcroffer.PaychOffer{}
	err = offer.Decode(data)
	if err != nil {
		log.Debugf("Fail to decode response: %v", err.Error())
		return fcroffer.PaychOffer{}, err
	}
	// Verify offer
	offerData, err := offer.GetToBeSigned()
	if err != nil {
		log.Errorf("Fail to get signing data for offer: %v", err.Error())
		return fcroffer.PaychOffer{}, err
	}
	err = crypto.Verify(offer.CurrencyID, offerData, offer.SignatureType, offer.Signature, offer.ToAddr)
	if err != nil {
		log.Debugf("Fail to verify signature: %v", err.Error())
		return fcroffer.PaychOffer{}, err
	}
	// Check offer
	if offer.CurrencyID != currencyID {
		return fcroffer.PaychOffer{}, fmt.Errorf("Invalid offer, expect currency id %v, got %v", currencyID, offer.CurrencyID)
	} else if offer.FromAddr != req.FromAddr {
		return fcroffer.PaychOffer{}, fmt.Errorf("Invalid offer, expect from %v, got %v", req.FromAddr, offer.FromAddr)
	} else if offer.ToAddr != toAddr {
		return fcroffer.PaychOffer{}, fmt.Errorf("Invalid offer, expect from %v, got %v", toAddr, offer.ToAddr)
	} else if time.Now().After(offer.Settlement) {
		return fcroffer.PaychOffer{}, fmt.Errorf("Invalid offer, settlement in the past")
	} else if time.Now().After(offer.Expiration) {
		return fcroffer.PaychOffer{}, fmt.Errorf("Invalid offer, expiration in the past")
	}
	return offer, nil
}

// Add asks peer to add a created payment channel based on received offer.
//
// @input - context, paych address, paych offer.
//
// @output - error.
func (proto *PaychProtocol) Add(ctx context.Context, chAddr string, offer fcroffer.PaychOffer) error {
	// Check if peer is blocked.
	subCtx, cancel := context.WithTimeout(ctx, proto.opTimeout)
	defer cancel()
	exists, err := proto.peerMgr.HasPeer(subCtx, offer.CurrencyID, offer.ToAddr)
	if err != nil {
		log.Warnf("Fail to check if contains peer %v-%v: %v", offer.CurrencyID, offer.ToAddr, err.Error())
		return err
	}
	if exists {
		blocked, err := proto.peerMgr.IsBlocked(subCtx, offer.CurrencyID, offer.ToAddr)
		if err != nil {
			log.Warnf("Fail to check if peer %v-%v is blocked: %v", offer.CurrencyID, offer.ToAddr, err.Error())
			return err
		}
		if blocked {
			return fmt.Errorf("peer %v-%v has been blocked", offer.CurrencyID, offer.ToAddr)
		}
	}
	// Connect to peer
	pid, err := proto.addrProto.ConnectToPeer(ctx, offer.CurrencyID, offer.ToAddr)
	if err != nil {
		log.Debugf("Fail to connect to %v-%v: %v", offer.CurrencyID, offer.ToAddr, err.Error())
		return err
	}
	subCtx, cancel = context.WithTimeout(ctx, proto.ioTimeout)
	defer cancel()
	conn, err := proto.h.NewStream(subCtx, pid, AddProtocol+ProtocolVersion)
	if err != nil {
		log.Debugf("Fail open stream to %v: %v", pid, err.Error())
		return err
	}
	defer conn.Close()
	// Create request.
	req, err := comms.NewRequestOut(ctx, proto.opTimeout, proto.ioTimeout, conn, proto.signer, offer.CurrencyID, offer.ToAddr)
	if err != nil {
		log.Debugf("Fail to create request: %v", err.Error())
		return err
	}
	// Send request
	type reqJson struct {
		ChAddr    string `json:"ch_addr"`
		OfferData []byte `json:"offer_data"`
	}
	offerData, err := offer.Encode()
	if err != nil {
		log.Errorf("Error encoding paych offer for %v: %v", req.CurrencyID, err.Error())
		return err
	}
	data, err := json.Marshal(reqJson{
		ChAddr:    chAddr,
		OfferData: offerData,
	})
	if err != nil {
		log.Errorf("Fail to encode request: %v", err.Error())
		return err
	}
	err = req.Send(ctx, proto.opTimeout, proto.ioTimeout, data)
	if err != nil {
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer fail to receive paych add request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, offer.CurrencyID, offer.ToAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, offer.CurrencyID, offer.ToAddr, err.Error())
			}
		}()
		log.Debugf("Fail to send request: %v", err.Error())
		return err
	}
	// Get response
	succeed, data, err := req.GetResponse(ctx, proto.ioTimeout)
	if err != nil {
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer fail to respond paych add request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, offer.CurrencyID, offer.ToAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, offer.CurrencyID, offer.ToAddr, err.Error())
			}
		}()
		log.Debugf("Fail to receive response: %v", err.Error())
		return err
	}
	if !succeed {
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer fail to process paych add request: %v", string(data)),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, offer.CurrencyID, offer.ToAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, offer.CurrencyID, offer.ToAddr, err.Error())
			}
		}()
		return fmt.Errorf("Peer fail to process request: %v", string(data))
	}
	go func() {
		rec := peermgr.Record{
			Description: fmt.Sprintf("peer succeed to process paych add request"),
			CreatedAt:   time.Now(),
		}
		err := proto.peerMgr.AddToHistory(proto.routineCtx, offer.CurrencyID, offer.ToAddr, rec)
		if err != nil {
			log.Warnf("Fail add %v to history of %v-%v: %v", rec, offer.CurrencyID, offer.ToAddr, err.Error())
		}
	}()
	// Add to payment manager
	err = proto.payMgr.AddOutboundChannel(ctx, req.CurrencyID, req.ToAddr, chAddr)
	if err != nil {
		log.Warnf("Fail to add outbound channel %v-%v-%v: %v", req.CurrencyID, req.ToAddr, chAddr, err.Error())
		return err
	}
	err = proto.monitor.Track(ctx, true, req.CurrencyID, chAddr, offer.Settlement)
	if err != nil {
		log.Warnf("Fail to track outbound channel %v-%v-%v: %v", req.CurrencyID, req.ToAddr, chAddr, err.Error())
		return err
	}
	return nil
}

// Renew renews a payment channel and ask peer to add it.
//
// @input - context, currency id, to address, channel address.
//
// @output - error.
func (proto *PaychProtocol) Renew(ctx context.Context, currencyID byte, toAddr string, chAddr string) error {
	// Check if peer is blocked.
	subCtx, cancel := context.WithTimeout(ctx, proto.opTimeout)
	defer cancel()
	exists, err := proto.peerMgr.HasPeer(subCtx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to check if contains peer %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	if exists {
		blocked, err := proto.peerMgr.IsBlocked(subCtx, currencyID, toAddr)
		if err != nil {
			log.Warnf("Fail to check if peer %v-%v is blocked: %v", currencyID, toAddr, err.Error())
			return err
		}
		if blocked {
			return fmt.Errorf("peer %v-%v has been blocked", currencyID, toAddr)
		}
	}
	// Connect to peer
	pid, err := proto.addrProto.ConnectToPeer(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to connect to %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	subCtx, cancel = context.WithTimeout(ctx, proto.ioTimeout)
	defer cancel()
	conn, err := proto.h.NewStream(subCtx, pid, RenewProtocol+ProtocolVersion)
	if err != nil {
		log.Debugf("Fail open stream to %v: %v", pid, err.Error())
		return err
	}
	defer conn.Close()
	// Create request.
	req, err := comms.NewRequestOut(ctx, proto.opTimeout, proto.ioTimeout, conn, proto.signer, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to create request: %v", err.Error())
		return err
	}
	// Send request
	err = req.Send(ctx, proto.opTimeout, proto.ioTimeout, []byte(chAddr))
	if err != nil {
		log.Debugf("Fail to send request: %v", err.Error())
		return err
	}
	// Get response
	succeed, data, err := req.GetResponse(ctx, proto.ioTimeout)
	if err != nil {
		log.Debugf("Fail to receive response: %v", err.Error())
		return err
	}
	if !succeed {
		return fmt.Errorf("Peer fail to process request: %v", string(data))
	}
	go func() {
		rec := peermgr.Record{
			Description: fmt.Sprintf("peer succeed to process paych renew request"),
			CreatedAt:   time.Now(),
		}
		err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
		if err != nil {
			log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
		}
	}()
	// Decode data
	newSettlement := time.Time{}
	err = newSettlement.UnmarshalJSON(data)
	if err != nil {
		log.Debugf("Fail to decode response: %v", data)
		return err
	}
	// Check settlement
	_, settlement, err := proto.monitor.Check(ctx, true, currencyID, chAddr)
	if err != nil {
		log.Debugf("Fail to check settlement for %v-%v-%v: %v", true, currencyID, chAddr, err.Error())
		return err
	}
	// Check settlement.
	if newSettlement.Before(settlement) {
		return fmt.Errorf("New settlement %v older than current settlement %v", newSettlement, settlement)
	}
	// Renew
	return proto.monitor.Renew(ctx, true, currencyID, chAddr, newSettlement)
}
