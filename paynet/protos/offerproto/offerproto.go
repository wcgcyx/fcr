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
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/offermgr"
	"github.com/wcgcyx/fcr/paymgr"
	"github.com/wcgcyx/fcr/peermgr"
	"github.com/wcgcyx/fcr/peernet/protos/addrproto"
	"github.com/wcgcyx/fcr/pservmgr"
	"github.com/wcgcyx/fcr/routestore"
)

const (
	QueryOfferProtocol = "/fcr-calibr/paynet/offer/query"
	ProtocolVersion    = "/0.0.1"
)

// Logger
var log = logging.Logger("payofferproto")

// OfferProtocol manages offer protocol.
// It handles payment offer creation.
type OfferProtocol struct {
	// Host
	h host.Host

	// Addr proto
	addrProto *addrproto.AddrProtocol

	// Signer
	signer crypto.Signer

	// Peer manager
	peerMgr peermgr.PeerManager

	// Offer manager
	offerMgr offermgr.OfferManager

	// Serve manager
	pservMgr pservmgr.PaychServingManager

	// Route store
	rs routestore.RouteStore

	// Payment manager
	payMgr paymgr.PaymentManager

	// Process related.
	routineCtx context.Context

	// Shutdown function.
	shutdown func()

	// Options
	ioTimeout       time.Duration
	opTimeout       time.Duration
	offerExpiry     time.Duration
	offerInactivity time.Duration
}

// NewOfferProtocol creates a new offer protocol.
//
// @input - host, addr protocol, signer, peer manager, offer manager, payment serving manager,
//   route store, payment manager, options.
//
// @output - offer protocol.
func NewOfferProtocol(
	h host.Host,
	addrProto *addrproto.AddrProtocol,
	signer crypto.Signer,
	peerMgr peermgr.PeerManager,
	offerMgr offermgr.OfferManager,
	pservMgr pservmgr.PaychServingManager,
	rs routestore.RouteStore,
	payMgr paymgr.PaymentManager,
	opts Opts,
) *OfferProtocol {
	// Parse options
	ioTimeout := opts.IOTimeout
	if ioTimeout == 0 {
		ioTimeout = defaultIOTimeout
	}
	opTimeout := opts.OpTimeout
	if opTimeout == 0 {
		opTimeout = defaultOpTimeout
	}
	offerExpiry := opts.OfferExpiry
	if offerExpiry == 0 {
		offerExpiry = defaultOfferExpiry
	}
	offerInactivity := opts.OfferInactivity
	if offerInactivity == 0 {
		offerInactivity = defaultOfferInactivity
	}
	log.Infof("Start pay offer protocol...")
	// Create routine
	routineCtx, cancel := context.WithCancel(context.Background())
	proto := &OfferProtocol{
		h:               h,
		addrProto:       addrProto,
		signer:          signer,
		peerMgr:         peerMgr,
		offerMgr:        offerMgr,
		pservMgr:        pservMgr,
		rs:              rs,
		payMgr:          payMgr,
		routineCtx:      routineCtx,
		ioTimeout:       ioTimeout,
		opTimeout:       opTimeout,
		offerExpiry:     offerExpiry,
		offerInactivity: offerInactivity,
	}
	h.SetStreamHandler(QueryOfferProtocol+ProtocolVersion, proto.handleQueryOffer)
	proto.shutdown = func() {
		log.Infof("Remove stream handler...")
		// Remove stream handler
		h.RemoveStreamHandler(QueryOfferProtocol + ProtocolVersion)
		// Close routines
		log.Infof("Stop all routines...")
		cancel()
	}
	return proto
}

// Shutdown safely shuts down the component.
func (p *OfferProtocol) Shutdown() {
	log.Infof("Start shutdown...")
	p.shutdown()
}
