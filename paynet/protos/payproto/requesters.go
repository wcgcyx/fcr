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
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/wcgcyx/fcr/comms"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/peermgr"
)

// PayForSelf does a payment for self.
//
// @input - context, currency id, recipient address, reserved channel, reserved id, petty amount required.
//
// @output - error.
func (proto *PayProtocol) PayForSelf(ctx context.Context, currencyID byte, toAddr string, resCh string, resID uint64, pettyAmtRequired *big.Int) error {
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
	conn, err := proto.h.NewStream(subCtx, pid, DirectPayProtocol+ProtocolVersion)
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
	subCtx, cancel = context.WithTimeout(ctx, proto.opTimeout)
	defer cancel()
	voucher, _, commit, revert, err := proto.payMgr.PayForSelf(subCtx, currencyID, toAddr, resCh, resID, pettyAmtRequired)
	if err != nil {
		log.Warnf("Fail to pay for self: %v", err.Error())
		return err
	}
	// Generate secret
	var secret [32]byte
	_, err = rand.Read(secret[:])
	if err != nil {
		revert("")
		log.Errorf("Fail to generate secret: %v", err.Error())
		return err
	}
	type reqJson struct {
		OriginalAddr   string   `json:"original_addr"`
		OriginalSecret [32]byte `json:"original_secret"`
		OriginalAmt    *big.Int `json:"original_amt"`
		Voucher        string   `json:"voucher"`
	}
	data, err := json.Marshal(reqJson{
		OriginalAddr:   req.FromAddr,
		OriginalSecret: secret,
		OriginalAmt:    pettyAmtRequired,
		Voucher:        voucher,
	})
	if err != nil {
		revert("")
		log.Errorf("Fail to encode request: %v", err.Error())
		return err
	}
	err = req.Send(ctx, proto.opTimeout, proto.ioTimeout, data)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer fail to receive pay request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Debugf("Fail to send request: %v", err.Error())
		return err
	}
	// Get response
	succeed, data, err := req.GetResponse(ctx, proto.ioTimeout)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer fail to respond pay request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Debugf("Fail to receive response: %v", err.Error())
		return err
	}
	if !succeed {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer failed to process pay request: %v", string(data)),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		return fmt.Errorf("Peer fail to process request: %v", string(data))
	}
	// Decode data
	type respJson struct {
		NetworkLossVoucher string `json:"network_loss"`
		SigType            byte   `json:"sig_type"`
		Signature          []byte `json:"signature"`
	}
	decoded := respJson{}
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer failed to provide valid request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Debugf("Fail to decode response: %v", err.Error())
		return err
	}
	if decoded.NetworkLossVoucher != "" {
		revert(decoded.NetworkLossVoucher)
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer got network loss"),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Warnf("Fail to pay as network loss presents")
		return fmt.Errorf("fail to pay, network loss presents")
	}
	err = crypto.Verify(currencyID, append(pettyAmtRequired.Bytes(), secret[:]...), decoded.SigType, decoded.Signature, toAddr)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer fail to provide valid receipt for pay request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Warnf("Fail to verify receipt signature: %v", err.Error())
		return err
	}
	// Good.
	// TODO: Should the nodes sync access time or each one records its own?
	// Here, each one records its own access time.
	go func() {
		rec := peermgr.Record{
			Description: fmt.Sprintf("peer succeed to process pay request"),
			CreatedAt:   time.Now(),
		}
		err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
		if err != nil {
			log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
		}
	}()
	commit(time.Now())
	return nil
}

// PayForSelfWithOffer does a payment for self with offer.
//
// @input - context, received offer, reserved channel, reserved id, petty amount required.
func (proto *PayProtocol) PayForSelfWithOffer(ctx context.Context, receivedOffer fcroffer.PayOffer, resCh string, resID uint64, pettyAmtRequired *big.Int) error {
	currencyID := receivedOffer.CurrencyID
	toAddr := receivedOffer.SrcAddr
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
	conn, err := proto.h.NewStream(subCtx, pid, ProxyPayProtocol+ProtocolVersion)
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
	subCtx, cancel = context.WithTimeout(ctx, proto.opTimeout)
	defer cancel()
	voucher, _, commit, revert, err := proto.payMgr.PayForSelf(subCtx, currencyID, toAddr, resCh, resID, pettyAmtRequired)
	if err != nil {
		log.Warnf("Fail to pay for self: %v", err.Error())
		return err
	}
	// Generate secret
	var secret [32]byte
	_, err = rand.Read(secret[:])
	if err != nil {
		revert("")
		log.Errorf("Fail to generate secret: %v", err.Error())
		return err
	}
	type reqJson struct {
		OriginalAddr   string   `json:"original_addr"`
		OriginalSecret [32]byte `json:"original_secret"`
		OriginalAmt    *big.Int `json:"original_amt"`
		OfferData      []byte   `json:"offer_data"`
		Voucher        string   `json:"voucher"`
	}
	offerData, err := receivedOffer.Encode()
	if err != nil {
		revert("")
		log.Errorf("Fail to encode offer: %v", err.Error())
		return err
	}
	data, err := json.Marshal(reqJson{
		OriginalAddr:   req.FromAddr,
		OriginalSecret: secret,
		OriginalAmt:    pettyAmtRequired,
		OfferData:      offerData,
		Voucher:        voucher,
	})
	if err != nil {
		revert("")
		log.Errorf("Fail to encode request: %v", err.Error())
		return err
	}
	err = req.Send(ctx, proto.opTimeout, proto.ioTimeout, data)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer fail to receive pay request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Debugf("Fail to send request: %v", err.Error())
		return err
	}
	// Get response
	succeed, data, err := req.GetResponse(ctx, proto.ioTimeout)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer fail to respond pay request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Debugf("Fail to receive response: %v", err.Error())
		return err
	}
	if !succeed {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer failed to process pay request: %v", string(data)),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		return fmt.Errorf("Peer fail to process request: %v", string(data))
	}
	// Decode data
	type respJson struct {
		NetworkLossVoucher string `json:"network_loss"`
		SigType            byte   `json:"sig_type"`
		Signature          []byte `json:"signature"`
	}
	decoded := respJson{}
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer failed to provide valid request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Debugf("Fail to decode response: %v", err.Error())
		return err
	}
	if decoded.NetworkLossVoucher != "" {
		revert(decoded.NetworkLossVoucher)
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer got network loss"),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Warnf("Fail to pay as network loss presents")
		return fmt.Errorf("fail to pay, network loss presents")
	}
	err = crypto.Verify(currencyID, append(pettyAmtRequired.Bytes(), secret[:]...), decoded.SigType, decoded.Signature, receivedOffer.DestAddr)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer fail to provide valid receipt for pay request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Warnf("Fail to verify receipt signature: %v", err.Error())
		return err
	}
	go func() {
		rec := peermgr.Record{
			Description: fmt.Sprintf("peer succeed to process pay request"),
			CreatedAt:   time.Now(),
		}
		err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
		if err != nil {
			log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
		}
	}()
	commit(time.Now())
	return nil
}

// payForOthers does a payment for others as a final node.
//
// @input - context, sent offer, gross amount received, petty amount required, original address, original secret.
func (proto *PayProtocol) payForOthersFinal(ctx context.Context, sentOffer fcroffer.PayOffer, grossAmtReceived *big.Int, originalAmt *big.Int, originalAddr string, originalSecret [32]byte) (byte, []byte, error) {
	currencyID := sentOffer.CurrencyID
	toAddr := sentOffer.DestAddr
	if toAddr != sentOffer.ResTo {
		return 0, nil, fmt.Errorf("invalid offer received: expected to %v, got %v", toAddr, sentOffer.ResTo)
	}
	// Check if peer is blocked.
	subCtx, cancel := context.WithTimeout(ctx, proto.opTimeout)
	defer cancel()
	exists, err := proto.peerMgr.HasPeer(subCtx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to check if contains peer %v-%v: %v", currencyID, toAddr, err.Error())
		return 0, nil, err
	}
	if exists {
		blocked, err := proto.peerMgr.IsBlocked(subCtx, currencyID, toAddr)
		if err != nil {
			log.Warnf("Fail to check if peer %v-%v is blocked: %v", currencyID, toAddr, err.Error())
			return 0, nil, err
		}
		if blocked {
			return 0, nil, fmt.Errorf("peer %v-%v has been blocked", currencyID, toAddr)
		}
	}
	// Connect to peer
	pid, err := proto.addrProto.ConnectToPeer(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to connect to %v-%v: %v", currencyID, toAddr, err.Error())
		return 0, nil, err
	}
	subCtx, cancel = context.WithTimeout(ctx, proto.ioTimeout)
	defer cancel()
	conn, err := proto.h.NewStream(subCtx, pid, DirectPayProtocol+ProtocolVersion)
	if err != nil {
		log.Debugf("Fail open stream to %v: %v", pid, err.Error())
		return 0, nil, err
	}
	defer conn.Close()
	// Create request.
	req, err := comms.NewRequestOut(ctx, proto.opTimeout, proto.ioTimeout, conn, proto.signer, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to create request: %v", err.Error())
		return 0, nil, err
	}
	subCtx, cancel = context.WithTimeout(ctx, proto.opTimeout)
	defer cancel()
	voucher, _, commit, revert, err := proto.payMgr.PayForOthers(subCtx, currencyID, toAddr, sentOffer.ResCh, sentOffer.ResID, grossAmtReceived, originalAmt)
	if err != nil {
		log.Warnf("Fail to pay for others: %v", err.Error())
		return 0, nil, err
	}
	// This is the final node doing proxy payment
	type reqJson struct {
		OriginalAddr   string   `json:"original_addr"`
		OriginalSecret [32]byte `json:"original_secret"`
		OriginalAmt    *big.Int `json:"original_amt"`
		Voucher        string   `json:"voucher"`
	}
	data, err := json.Marshal(reqJson{
		OriginalAddr:   originalAddr,
		OriginalSecret: originalSecret,
		OriginalAmt:    originalAmt,
		Voucher:        voucher,
	})
	if err != nil {
		revert("")
		log.Errorf("Fail to encode request: %v", err.Error())
		return 0, nil, err
	}
	err = req.Send(ctx, proto.opTimeout, proto.ioTimeout, data)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer fail to receive pay request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Debugf("Fail to send request: %v", err.Error())
		return 0, nil, err
	}
	// Get response
	succeed, data, err := req.GetResponse(ctx, proto.ioTimeout)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer fail to respond pay request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Debugf("Fail to receive response: %v", err.Error())
		return 0, nil, err
	}
	if !succeed {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer failed to process pay request: %v", string(data)),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		return 0, nil, fmt.Errorf("Peer fail to process request: %v", string(data))
	}
	// Decode data
	type respJson struct {
		NetworkLossVoucher string `json:"network_loss"`
		SigType            byte   `json:"sig_type"`
		Signature          []byte `json:"signature"`
	}
	decoded := respJson{}
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer failed to provide valid request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Debugf("Fail to decode response: %v", err.Error())
		return 0, nil, err
	}
	if decoded.NetworkLossVoucher != "" {
		revert(decoded.NetworkLossVoucher)
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer got network loss"),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Warnf("Fail to pay as network loss presents")
		return 0, nil, fmt.Errorf("fail to pay, network loss presents")
	}
	err = crypto.Verify(currencyID, append(originalAmt.Bytes(), originalSecret[:]...), decoded.SigType, decoded.Signature, sentOffer.DestAddr)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer fail to provide valid receipt for pay request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Warnf("Fail to verify receipt signature: %v", err.Error())
		return 0, nil, err
	}
	go func() {
		rec := peermgr.Record{
			Description: fmt.Sprintf("peer succeed to process pay request"),
			CreatedAt:   time.Now(),
		}
		err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
		if err != nil {
			log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
		}
	}()
	commit(time.Now())
	return decoded.SigType, decoded.Signature, nil
}

// payForOthersIntermediate does a payment for others as an intermediate node.
func (proto *PayProtocol) payForOthersIntermediate(ctx context.Context, sentOffer fcroffer.PayOffer, grossAmtReceived *big.Int, originalAmt *big.Int, originalAddr string, originalSecret [32]byte) (byte, []byte, error) {
	currencyID := sentOffer.CurrencyID
	toAddr := sentOffer.ResTo
	// Check if peer is blocked.
	subCtx, cancel := context.WithTimeout(ctx, proto.opTimeout)
	defer cancel()
	exists, err := proto.peerMgr.HasPeer(subCtx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to check if contains peer %v-%v: %v", currencyID, toAddr, err.Error())
		return 0, nil, err
	}
	if exists {
		blocked, err := proto.peerMgr.IsBlocked(subCtx, currencyID, toAddr)
		if err != nil {
			log.Warnf("Fail to check if peer %v-%v is blocked: %v", currencyID, toAddr, err.Error())
			return 0, nil, err
		}
		if blocked {
			return 0, nil, fmt.Errorf("peer %v-%v has been blocked", currencyID, toAddr)
		}
	}
	// Connect to peer
	pid, err := proto.addrProto.ConnectToPeer(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to connect to %v-%v: %v", currencyID, toAddr, err.Error())
		return 0, nil, err
	}
	subCtx, cancel = context.WithTimeout(ctx, proto.ioTimeout)
	defer cancel()
	conn, err := proto.h.NewStream(subCtx, pid, ProxyPayProtocol+ProtocolVersion)
	if err != nil {
		log.Debugf("Fail open stream to %v: %v", pid, err.Error())
		return 0, nil, err
	}
	defer conn.Close()
	// Create request.
	req, err := comms.NewRequestOut(ctx, proto.opTimeout, proto.ioTimeout, conn, proto.signer, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to create request: %v", err.Error())
		return 0, nil, err
	}
	subCtx, cancel = context.WithTimeout(ctx, proto.opTimeout)
	defer cancel()
	voucher, receivedOffer, commit, revert, err := proto.payMgr.PayForOthers(subCtx, currencyID, toAddr, sentOffer.ResCh, sentOffer.ResID, grossAmtReceived, originalAmt)
	if err != nil {
		log.Warnf("Fail to pay for others: %v", err.Error())
		return 0, nil, err
	}
	if receivedOffer == nil {
		revert("")
		log.Errorf("Fail to have received offer, which should be expected")
		return 0, nil, fmt.Errorf("expect to have received offer for proxy payment but got nil")
	}
	type reqJson struct {
		OriginalAddr   string   `json:"original_addr"`
		OriginalSecret [32]byte `json:"original_secret"`
		OriginalAmt    *big.Int `json:"original_amt"`
		OfferData      []byte   `json:"offer_data"`
		Voucher        string   `json:"voucher"`
	}
	offerData, err := receivedOffer.Encode()
	if err != nil {
		revert("")
		log.Errorf("Fail to encode offer: %v", err.Error())
		return 0, nil, err
	}
	data, err := json.Marshal(reqJson{
		OriginalAddr:   originalAddr,
		OriginalSecret: originalSecret,
		OriginalAmt:    originalAmt,
		OfferData:      offerData,
		Voucher:        voucher,
	})
	if err != nil {
		revert("")
		log.Errorf("Fail to encode request: %v", err.Error())
		return 0, nil, err
	}
	err = req.Send(ctx, proto.opTimeout, proto.ioTimeout, data)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer fail to receive pay request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Debugf("Fail to send request: %v", err.Error())
		return 0, nil, err
	}
	// Get response
	succeed, data, err := req.GetResponse(ctx, proto.ioTimeout)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer fail to respond pay request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Debugf("Fail to receive response: %v", err.Error())
		return 0, nil, err
	}
	if !succeed {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer failed to process pay request: %v", string(data)),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		return 0, nil, fmt.Errorf("Peer fail to process request: %v", string(data))
	}
	// Decode data
	type respJson struct {
		NetworkLossVoucher string `json:"network_loss"`
		SigType            byte   `json:"sig_type"`
		Signature          []byte `json:"signature"`
	}
	decoded := respJson{}
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer failed to provide valid request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Debugf("Fail to decode response: %v", err.Error())
		return 0, nil, err
	}
	if decoded.NetworkLossVoucher != "" {
		revert(decoded.NetworkLossVoucher)
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer got network loss"),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Warnf("Fail to pay as network loss presents")
		return 0, nil, fmt.Errorf("fail to pay, network loss presents")
	}
	err = crypto.Verify(currencyID, append(originalAmt.Bytes(), originalSecret[:]...), decoded.SigType, decoded.Signature, sentOffer.DestAddr)
	if err != nil {
		revert("")
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer fail to provide valid receipt for pay request: %v", err.Error()),
				CreatedAt:   time.Now(),
			}
			err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
			}
		}()
		log.Warnf("Fail to verify receipt signature: %v", err.Error())
		return 0, nil, err
	}
	go func() {
		rec := peermgr.Record{
			Description: fmt.Sprintf("peer succeed to process pay request"),
			CreatedAt:   time.Now(),
		}
		err := proto.peerMgr.AddToHistory(proto.routineCtx, currencyID, toAddr, rec)
		if err != nil {
			log.Warnf("Fail add %v to history of %v-%v: %v", rec, currencyID, toAddr, err.Error())
		}
	}()
	commit(time.Now())
	return decoded.SigType, decoded.Signature, nil
}
