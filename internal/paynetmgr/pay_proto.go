package paynetmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/wcgcyx/fcr/internal/crypto"
	"github.com/wcgcyx/fcr/internal/io"
	"github.com/wcgcyx/fcr/internal/messages"
	"github.com/wcgcyx/fcr/internal/paymgr"
	"github.com/wcgcyx/fcr/internal/payoffer"
	"github.com/wcgcyx/fcr/internal/payofferstore"
	"github.com/wcgcyx/fcr/internal/peermgr"
	"github.com/wcgcyx/fcr/internal/routestore"
)

// PayProtocol is the protocol to request and answer pay.
type PayProtocol struct {
	// Parent context
	ctx context.Context
	// Host
	h host.Host
	// Route store
	rs routestore.RouteStore
	// Offer store
	os payofferstore.PayOfferStore
	// Peer manager
	pem peermgr.PeerManager
	// Payment manager
	pam paymgr.PaymentManager
}

// NewPayProtocol creates a new pay protocol.
// It takes a context, a host, route store, offer store,
// peer manager and payment manager as arguments.
// It returns pay protocol.
func NewPayProtocol(
	ctx context.Context,
	h host.Host,
	rs routestore.RouteStore,
	os payofferstore.PayOfferStore,
	pem peermgr.PeerManager,
	pam paymgr.PaymentManager,
) *PayProtocol {
	proto := &PayProtocol{
		ctx: ctx,
		h:   h,
		rs:  rs,
		os:  os,
		pem: pem,
		pam: pam,
	}
	h.SetStreamHandler(payProtocol, proto.handlePay)
	return proto
}

// Pay pays to given address with amount.
// It takes a currency id, a recipient address and amount as arguments.
// It returns error.
func (proto *PayProtocol) Pay(parentCtx context.Context, currencyID uint64, toAddr string, amt *big.Int) error {
	fromAddr, err := proto.pam.GetRootAddress(currencyID)
	if err != nil {
		return fmt.Errorf("error getting root address for currency id %v: %v", currencyID, err.Error())
	}
	pid, blocked, _, _, err := proto.pem.GetPeerInfo(currencyID, toAddr)
	if err != nil {
		return fmt.Errorf("error in getting peer information for addr %v: %v", toAddr, err.Error())
	}
	if blocked {
		return fmt.Errorf("peer %v with addr %v has been blocked", pid.ShortString(), toAddr)
	}
	ctx, cancel := context.WithTimeout(parentCtx, payTTL)
	defer cancel()
	conn, err := proto.h.NewStream(ctx, pid, payProtocol)
	if err != nil {
		return fmt.Errorf("error opening stream to peer %v: %v", pid.ShortString(), err.Error())
	}
	voucher, commit, revert, err := proto.pam.PayForSelf(currencyID, toAddr, amt)
	if err != nil {
		return fmt.Errorf("error generating voucher for %v: %v", toAddr, err.Error())
	}
	// Generating secret
	var secret [32]byte
	rand.Read(secret[:])
	// Sending voucher to recipient
	err = io.Write(conn, messages.EncodePayReq(currencyID, fromAddr, secret, amt, fromAddr, voucher, nil), ioTimeout)
	if err != nil {
		proto.pem.Record(currencyID, toAddr, false)
		revert()
		return fmt.Errorf("error writing data to stream: %v", err.Error())
	}
	data, err := io.Read(conn, payTTL)
	if err != nil {
		proto.pem.Record(currencyID, toAddr, false)
		revert()
		return fmt.Errorf("error reading data from stream: %v", err.Error())
	}
	receipt, err := messages.DecodePayResp(data)
	if err != nil {
		proto.pem.Record(currencyID, toAddr, false)
		revert()
		return fmt.Errorf("error decoding message: %v", err.Error())
	}
	// Verify receipt
	err = crypto.Verify(toAddr, receipt[:], append(secret[:], amt.Bytes()...))
	if err != nil {
		proto.pem.Record(currencyID, toAddr, false)
		revert()
		return fmt.Errorf("fail to verify receipt: %v", err.Error())
	}
	proto.pem.Record(currencyID, toAddr, true)
	commit()
	return nil
}

// PayWithOffer pays amount with an offer.
// It takes amount and offer as arguments.
// It returns error.
func (proto *PayProtocol) PayWithOffer(parentCtx context.Context, amt *big.Int, offer *payoffer.PayOffer) error {
	if offer == nil {
		return fmt.Errorf("nil offer supplied")
	}
	currencyID := offer.CurrencyID()
	toAddr := offer.From()
	fromAddr, err := proto.pam.GetRootAddress(currencyID)
	if err != nil {
		return fmt.Errorf("error getting root address for currency id %v: %v", currencyID, err.Error())
	}
	pid, blocked, _, _, err := proto.pem.GetPeerInfo(currencyID, toAddr)
	if err != nil {
		return fmt.Errorf("error in getting peer information for addr %v: %v", toAddr, err.Error())
	}
	if blocked {
		return fmt.Errorf("peer %v with addr %v has been blocked", pid.ShortString(), toAddr)
	}
	ctx, cancel := context.WithTimeout(parentCtx, payTTL)
	defer cancel()
	conn, err := proto.h.NewStream(ctx, pid, payProtocol)
	if err != nil {
		return fmt.Errorf("error opening stream to peer %v: %v", pid.ShortString(), err.Error())
	}
	voucher, commit, revert, err := proto.pam.PayForSelfWithOffer(amt, offer)
	if err != nil {
		return fmt.Errorf("error generating voucher for %v: %v", toAddr, err.Error())
	}
	// Generating secret
	var secret [32]byte
	rand.Read(secret[:])
	// Sending voucher to recipient
	err = io.Write(conn, messages.EncodePayReq(currencyID, fromAddr, secret, amt, fromAddr, voucher, offer), ioTimeout)
	if err != nil {
		proto.pem.Record(currencyID, toAddr, false)
		revert()
		return fmt.Errorf("error writing data to stream: %v", err.Error())
	}
	data, err := io.Read(conn, payTTL)
	if err != nil {
		proto.pem.Record(currencyID, toAddr, false)
		revert()
		return fmt.Errorf("error reading data from stream: %v", err.Error())
	}
	receipt, err := messages.DecodePayResp(data)
	if err != nil {
		proto.pem.Record(currencyID, toAddr, false)
		revert()
		return fmt.Errorf("error decoding message: %v", err.Error())
	}
	// Verify receipt
	err = crypto.Verify(offer.To(), receipt[:], append(secret[:], amt.Bytes()...))
	if err != nil {
		proto.pem.Record(currencyID, toAddr, false)
		revert()
		return fmt.Errorf("fail to verify receipt: %v", err.Error())
	}
	proto.pem.Record(currencyID, toAddr, true)
	commit()
	return nil
}

// handlePay handles pay request.
// It takes a connection stream as the argument.
func (proto *PayProtocol) handlePay(conn network.Stream) {
	defer conn.Close()
	data, err := io.Read(conn, ioTimeout)
	if err != nil {
		log.Debugf("error reading from stream %v: %v", conn.ID(), err.Error())
		return
	}
	currencyID, originAddr, originSecret, originAmt, fromAddr, voucher, offer, err := messages.DecodePayReq(data)
	if err != nil {
		log.Debugf("error decoding message from stream %v: %v", conn.ID(), err.Error())
		return
	}
	if offer == nil {
		// This payment is for myself
		amt, commit, revert, err := proto.pam.RecordReceiveForSelf(currencyID, fromAddr, voucher, originAddr)
		if err != nil {
			log.Debugf("error receving payment: %v", err.Error())
			return
		}
		if amt.Cmp(originAmt) < 0 {
			log.Debugf("received insufficient amount: expect %v got %v", originAmt.String(), amt.String())
			revert()
			return
		}
		// Payment is good, sign secret + amt
		prv, _ := proto.pam.GetPrvKey(currencyID)
		receipt, err := crypto.Sign(prv, append(originSecret[:], originAmt.Bytes()...))
		if err != nil {
			log.Debugf("error signing receipt: %v", err.Error())
			revert()
			return
		}
		if err = io.Write(conn, messages.EncodePayResp(receipt), ioTimeout); err != nil {
			revert()
			log.Debugf("error writing to stream: %v", err.Error())
			return
		}
		commit()
	} else {
		// This payment is for someone else, need to redirect it
		// Need to verify the offer
		if err = offer.Verify(); err != nil {
			log.Debugf("error verifying offer")
			return
		}
		root, err := proto.pam.GetRootAddress(currencyID)
		if err != nil {
			log.Debugf("error getting root address for currency id %v", err.Error())
			return
		}
		if offer.From() != root {
			log.Debug("offer expect to coming from %v but got %v", root, offer.From())
			return
		}
		if time.Now().Unix() > offer.Expiration() {
			log.Debug("offer has expired at %v, now is %v", offer.Expiration(), time.Now().Unix())
			return
		}
		amt, commitR, revertR, err := proto.pam.RecordReceiveForOthers(fromAddr, voucher, offer)
		if err != nil {
			log.Debugf("error receving payment: %v", err.Error())
			return
		}
		if amt.Cmp(originAmt) < 0 {
			log.Debugf("received insufficient amount: expect %v got %v", originAmt.String(), amt.String())
			revertR()
			return
		}
		// Start redirecting this payment
		var toAddr string
		var linkedOffer *payoffer.PayOffer
		if offer.LinkedOffer() == cid.Undef {
			toAddr = offer.To()
		} else {
			linkedOffer, err = proto.os.GetOffer(offer.LinkedOffer())
			if err != nil {
				log.Debugf("error loading linked offer %v: %v", offer.LinkedOffer(), err.Error())
				revertR()
				return
			}
			toAddr = linkedOffer.From()
		}
		voucher, commitP, revertP, err := proto.pam.PayForOthers(originAmt, offer)
		if err != nil {
			log.Debugf("error creating voucher: %v", err.Error())
			revertR()
			return
		}
		pid, blocked, _, _, err := proto.pem.GetPeerInfo(currencyID, toAddr)
		if err != nil {
			log.Debugf("error getting peer information for addr %v: %v", toAddr, err.Error())
			revertR()
			revertP()
			return
		}
		if blocked {
			log.Debugf("peer %v with addr %v has been blocked", pid.ShortString(), toAddr)
			revertR()
			revertP()
			return
		}
		ctx, cancel := context.WithTimeout(proto.ctx, payTTL)
		defer cancel()
		connOut, err := proto.h.NewStream(ctx, pid, payProtocol)
		if err != nil {
			log.Debugf("error opening stream to peer %v: %v", pid.ShortString(), err.Error())
			proto.pem.Record(currencyID, toAddr, false)
			revertR()
			revertP()
			return
		}
		err = io.Write(connOut, messages.EncodePayReq(currencyID, originAddr, originSecret, originAmt, root, voucher, linkedOffer), ioTimeout)
		if err != nil {
			log.Debugf("error writing data to stream: %v", err.Error())
			proto.pem.Record(currencyID, toAddr, false)
			revertR()
			revertP()
			return
		}
		data, err := io.Read(connOut, payTTL)
		if err != nil {
			log.Debugf("error reading data from stream: %v", err.Error())
			proto.pem.Record(currencyID, toAddr, false)
			revertR()
			revertP()
			return
		}
		receipt, err := messages.DecodePayResp(data)
		if err != nil {
			log.Debugf("error decoding message: %v", err.Error())
			proto.pem.Record(currencyID, toAddr, false)
			revertR()
			revertP()
			return
		}
		// Verify receipt
		err = crypto.Verify(offer.To(), receipt[:], append(originSecret[:], originAmt.Bytes()...))
		if err != nil {
			log.Debugf("fail to verify receipt: %v", err.Error())
			proto.pem.Record(currencyID, toAddr, false)
			revertR()
			revertP()
			return
		}
		// Reply to sender
		proto.pem.Record(currencyID, toAddr, true)
		commitR()
		commitP()
		err = io.Write(conn, messages.EncodePayResp(receipt[:]), ioTimeout)
		if err != nil {
			log.Debugf("error in replying: %v", err.Error())
		}
	}
}
