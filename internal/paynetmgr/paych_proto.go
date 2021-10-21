package paynetmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wcgcyx/fcr/internal/io"
	"github.com/wcgcyx/fcr/internal/messages"
	"github.com/wcgcyx/fcr/internal/paychoffer"
	"github.com/wcgcyx/fcr/internal/paymgr"
)

// PaychProtocol is the protocol to ask for paych offer and add paych request.
type PaychProtocol struct {
	// Host
	h host.Host
	// SettlementDelay
	delayLock sync.RWMutex
	delay     time.Duration
	// Payment manager
	pam paymgr.PaymentManager
}

// NewPaychProtocol creates a new paych protocol.
// It takes a host, a settlement delay,
// a payment manager as arguments.
// It returns paych protocol.
func NewPaychProtocol(
	h host.Host,
	delay time.Duration,
	pam paymgr.PaymentManager,
) *PaychProtocol {
	proto := &PaychProtocol{
		h:         h,
		delayLock: sync.RWMutex{},
		delay:     delay,
		pam:       pam,
	}
	h.SetStreamHandler(queryPaychProtocol, proto.handleQueryOffer)
	h.SetStreamHandler(addPaychProtocol, proto.handleAddPaych)
	return proto
}

// UpdateDelay updates the settlement delay.
// It takes a new settlement delay as the argument.
func (proto *PaychProtocol) UpdateDelay(newDelay time.Duration) {
	proto.delayLock.Lock()
	defer proto.delayLock.Unlock()
	proto.delay = newDelay
}

// QueryOffer is used to query an paych offer from a peer.
// It takes a parent context, a currency id, recipient address, peer addr info as arguments.
// It returns a paych offer and error.
func (proto *PaychProtocol) QueryOffer(parentCtx context.Context, currencyID uint64, toAddr string, pi peer.AddrInfo) (*paychoffer.PaychOffer, error) {
	ctx, cancel := context.WithTimeout(parentCtx, ioTimeout)
	defer cancel()
	err := proto.h.Connect(ctx, pi)
	if err != nil {
		return nil, fmt.Errorf("error connecting to peer: %v", err.Error())
	}
	conn, err := proto.h.NewStream(ctx, pi.ID, queryPaychProtocol)
	if err != nil {
		return nil, fmt.Errorf("error opening stream to peer %v: %v", pi.ID.ShortString(), err.Error())
	}
	defer conn.Close()
	// Write request to stream
	data := messages.EncodePaychOfferReq(currencyID, toAddr)
	err = io.Write(conn, data, ioTimeout)
	if err != nil {
		return nil, fmt.Errorf("error writing data to stream: %v", err.Error())
	}
	// Read response from stream
	data, err = io.Read(conn, ioTimeout)
	if err != nil {
		return nil, fmt.Errorf("error reading data from stream: %v", err.Error())
	}
	offer, err := messages.DecodePaychOfferResp(data)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err.Error())
	}
	// Validate offer
	if offer.CurrencyID() != currencyID {
		return nil, fmt.Errorf("invalid offer received: expect currency id of %v got %v", currencyID, offer.CurrencyID())
	}
	if offer.Addr() != toAddr {
		return nil, fmt.Errorf("invalid offer received: expect to %v got %v", toAddr, offer.Addr())
	}
	if time.Now().Unix() > offer.Expiration()-int64(paychOfferTTLRoom.Seconds()) {
		return nil, fmt.Errorf("invalid offer received: offer expired at %v, now is %v", offer.Expiration(), time.Now())
	}
	if err = offer.Verify(); err != nil {
		return nil, fmt.Errorf("invalid offer received: fail to verify offer: %v", err.Error())
	}
	// Offer validated
	return offer, nil
}

// handleQueryOffer handles a query offer request.
// It takes a connection stream as the argument.
func (proto *PaychProtocol) handleQueryOffer(conn network.Stream) {
	defer conn.Close()
	data, err := io.Read(conn, ioTimeout)
	if err != nil {
		log.Debugf("error reading from stream %v: %v", conn.ID(), err.Error())
		return
	}
	currencyID, addr, err := messages.DecodePaychOfferReq(data)
	if err != nil {
		log.Debugf("error decoding message from stream %v: %v", conn.ID(), err.Error())
		return
	}
	root, err := proto.pam.GetRootAddress(currencyID)
	if err != nil {
		log.Debugf("error getting root address for currency id %v: %v", currencyID, err.Error())
		return
	}
	if root != addr {
		log.Debugf("invalid request, expect root to be %v, got %v", root, addr)
		return
	}
	proto.delayLock.RLock()
	defer proto.delayLock.RUnlock()
	if proto.delay <= 0 {
		log.Debugf("invalid delay set, expect positive, got %v", proto.delay)
		return
	}
	prv, err := proto.pam.GetPrvKey(currencyID)
	if err != nil {
		log.Debugf("error getting private key for currency id %v: %v", currencyID, err.Error())
		return
	}
	offer, err := paychoffer.NewPaychOffer(prv, currencyID, proto.delay, paychOfferTTL)
	if err != nil {
		log.Debugf("error creating offer: %v", err.Error())
		return
	}
	// Respond with offer.
	data = messages.EncodePaychOfferResp(*offer)
	err = io.Write(conn, data, ioTimeout)
	if err != nil {
		log.Debugf("error writing data to stream: %v", err.Error())
	}
}

// AddPaych is used to send an add paych request to a peer.
// It takes a context, a currency id, channel address, paych offer, peer id as arguments.
// It returns error.
func (proto *PaychProtocol) AddPaych(parentCtx context.Context, currencyID uint64, chAddr string, offer *paychoffer.PaychOffer, pid peer.ID) error {
	ctx, cancel := context.WithTimeout(parentCtx, ioTimeout)
	defer cancel()
	conn, err := proto.h.NewStream(ctx, pid, addPaychProtocol)
	if err != nil {
		return fmt.Errorf("error opening stream to peer %v: %v", pid.ShortString(), err.Error())
	}
	defer conn.Close()
	// Write request to stream
	data := messages.EncodePaychRegister(chAddr, offer)
	err = io.Write(conn, data, ioTimeout)
	if err != nil {
		return fmt.Errorf("error writing data to stream: %v", err.Error())
	}
	data, err = io.Read(conn, ioTimeout)
	if err != nil {
		return fmt.Errorf("error reading data from stream: %v", err.Error())
	}
	if data[0] != 1 {
		return fmt.Errorf("fail to register paych: %v", string(data[1:]))
	}
	return nil
}

// handleAddPaych handles an add paych request.
// It takes a connection stream as the argument.
func (proto *PaychProtocol) handleAddPaych(conn network.Stream) {
	defer conn.Close()
	data, err := io.Read(conn, ioTimeout)
	if err != nil {
		log.Debugf("error reading from stream %v: %v", conn.ID(), err.Error())
		return
	}
	chAddr, offer, err := messages.DecodePaychRegister(data)
	if err != nil {
		err = fmt.Errorf("error reading from stream %v: %v", conn.ID(), err.Error())
		log.Debugf("%v", err.Error())
		if err = io.Write(conn, append([]byte{0}, []byte(err.Error())...), ioTimeout); err != nil {
			log.Debugf("error writing data to stream: %v", err.Error())
		}
		return
	}
	// Verify offer
	if time.Now().Unix() > offer.Expiration() {
		err = fmt.Errorf("offer has expired at %v, now %v", offer.Expiration(), time.Now().Unix())
		log.Debugf("%v", err.Error())
		if err = io.Write(conn, append([]byte{0}, []byte(err.Error())...), ioTimeout); err != nil {
			log.Debugf("error writing data to stream: %v", err.Error())
		}
		return
	}
	root, err := proto.pam.GetRootAddress(offer.CurrencyID())
	if err != nil {
		err = fmt.Errorf("error getting root address for currency id %v", err.Error())
		log.Errorf("%v", err.Error())
		if err = io.Write(conn, append([]byte{0}, []byte("internal error")...), ioTimeout); err != nil {
			log.Debugf("error writing data to stream: %v", err.Error())
		}
		return
	}
	if err = offer.Verify(); err != nil {
		err = fmt.Errorf("invalid offer received: fail to verify offer: %v", err.Error())
		log.Debugf("%v", err.Error())
		if err = io.Write(conn, append([]byte{0}, []byte(err.Error())...), ioTimeout); err != nil {
			log.Debugf("error writing data to stream: %v", err.Error())
		}
		return
	}
	if offer.Addr() != root {
		err = fmt.Errorf("invalid offer received: expect to %v got %v", root, offer.Addr())
		log.Debugf("%v", err.Error())
		if err = io.Write(conn, append([]byte{0}, []byte(err.Error())...), ioTimeout); err != nil {
			log.Debugf("error writing data to stream: %v", err.Error())
		}
		return
	}
	// Add channel
	err = proto.pam.AddInboundCh(offer.CurrencyID(), chAddr, offer.Settlement())
	if err != nil {
		err = fmt.Errorf("error adding inbound channel %v", err.Error())
		log.Errorf("%v", err.Error())
		if err = io.Write(conn, append([]byte{0}, []byte("internal error")...), ioTimeout); err != nil {
			log.Debugf("error writing data to stream: %v", err.Error())
		}
		return
	}
	if err = io.Write(conn, []byte{1}, ioTimeout); err != nil {
		log.Debugf("error writing data to stream: %v", err.Error())
	}
}
