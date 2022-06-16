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
	"sync"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/offermgr"
	"github.com/wcgcyx/fcr/paychmon"
	"github.com/wcgcyx/fcr/paymgr"
	"github.com/wcgcyx/fcr/peermgr"
	"github.com/wcgcyx/fcr/peernet/protos/addrproto"
	"github.com/wcgcyx/fcr/renewmgr"
	"github.com/wcgcyx/fcr/settlemgr"
)

const (
	QueryAddProtocol = "/fcr-calibr/paynet/paych/query"
	AddProtocol      = "/fcr-calibr/paynet/paych/add"
	RenewProtocol    = "/fcr-calibr/paynet/paych/renew"
	ProtocolVersion  = "/0.0.1"
)

// Logger
var log = logging.Logger("paychproto")

// PaychProtocol manages paych protocol.
// It handles paych creation and renew.
type PaychProtocol struct {
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

	// Settlement manager
	settleMgr settlemgr.SettlementManager

	// Renew manager
	renewMgr renewmgr.RenewManager

	// Payment manager
	payMgr paymgr.PaymentManager

	// Monitor
	monitor paychmon.PaychMonitor

	// Publish hook
	publishHookLock sync.RWMutex
	publishHook     func(byte, string)

	// Process related.
	routineCtx context.Context

	// Shutdown function.
	shutdown func()

	// Options
	ioTimeout   time.Duration
	opTimeout   time.Duration
	offerExpiry time.Duration
	renewWindow uint64
}

// NewPaychProtocol creates a new paych protocol.
//
// @input - host, addr protocol, signer, peer manager, offer manager, settlement manager,
//   renew manager, payment manager, options.
//
// @output - paych protocol.
func NewPaychProtocol(
	h host.Host,
	addrProto *addrproto.AddrProtocol,
	signer crypto.Signer,
	peerMgr peermgr.PeerManager,
	offerMgr offermgr.OfferManager,
	settleMgr settlemgr.SettlementManager,
	renewMgr renewmgr.RenewManager,
	payMgr paymgr.PaymentManager,
	monitor paychmon.PaychMonitor,
	opts Opts,
) *PaychProtocol {
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
	renewWindow := opts.RenewWindow
	if renewWindow == 0 || renewWindow >= 100 {
		renewWindow = defaultRenewWindow
	}
	log.Infof("Start paych protocol...")
	// Create routine
	routineCtx, cancel := context.WithCancel(context.Background())
	proto := &PaychProtocol{
		h:               h,
		addrProto:       addrProto,
		signer:          signer,
		peerMgr:         peerMgr,
		offerMgr:        offerMgr,
		settleMgr:       settleMgr,
		renewMgr:        renewMgr,
		payMgr:          payMgr,
		monitor:         monitor,
		publishHookLock: sync.RWMutex{},
		routineCtx:      routineCtx,
		ioTimeout:       ioTimeout,
		opTimeout:       opTimeout,
		offerExpiry:     offerExpiry,
		renewWindow:     renewWindow,
	}
	h.SetStreamHandler(QueryAddProtocol+ProtocolVersion, proto.handleQueryAdd)
	h.SetStreamHandler(AddProtocol+ProtocolVersion, proto.handleAdd)
	h.SetStreamHandler(RenewProtocol+ProtocolVersion, proto.handleRenew)
	proto.shutdown = func() {
		log.Infof("Remove stream handlers...")
		// Remove stream handlers
		h.RemoveStreamHandler(QueryAddProtocol + ProtocolVersion)
		h.RemoveStreamHandler(AddProtocol + ProtocolVersion)
		h.RemoveStreamHandler(RenewProtocol + ProtocolVersion)
		// Close routines
		log.Infof("Stop all routines...")
		cancel()
	}
	return proto
}

// Shutdown safely shuts down the component.
func (p *PaychProtocol) Shutdown() {
	log.Infof("Start shutdown...")
	p.shutdown()
}

// RegisterPublishHook adds a hook function to call when a new subscriber presents.
//
// @input - hook function.
func (p *PaychProtocol) RegisterPublishFunc(hook func(byte, string)) {
	p.publishHookLock.Lock()
	defer p.publishHookLock.Unlock()
	p.publishHook = hook
}

// DeregisterPublishHook deregister the hook function.
func (p *PaychProtocol) DeregisterPublishHook() {
	p.publishHookLock.Lock()
	defer p.publishHookLock.Unlock()
	p.publishHook = nil
}
