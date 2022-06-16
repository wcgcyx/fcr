package routeproto

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
	"github.com/wcgcyx/fcr/paynet/protos/paychproto"
	"github.com/wcgcyx/fcr/peermgr"
	"github.com/wcgcyx/fcr/peernet/protos/addrproto"
	"github.com/wcgcyx/fcr/pservmgr"
	"github.com/wcgcyx/fcr/routestore"
	"github.com/wcgcyx/fcr/substore"
)

const (
	PublishProtocol = "/fcr-calibr/paynet/route/publish"
	ProtocolVersion = "/0.0.1"
)

// Logger
var log = logging.Logger("routeproto")

// RouteProtocol manages route protocol.
// It handles route publish and share.
type RouteProtocol struct {
	// Host
	h host.Host

	// Addr proto
	addrProto *addrproto.AddrProtocol

	// Paych proto
	paychProto *paychproto.PaychProtocol

	// Signer
	signer crypto.Signer

	// Peer manager
	peerMgr peermgr.PeerManager

	// Serve manager
	pservMgr pservmgr.PaychServingManager

	// Route store
	rs routestore.RouteStore

	// Subscriber store
	sbs substore.SubStore

	// Process related.
	routineCtx context.Context

	// Shutdown function.
	shutdown func()

	// Options
	ioTimeout   time.Duration
	opTimeout   time.Duration
	publishFreq time.Duration
	routeExpiry time.Duration
	publishWait time.Duration
}

// NewRouteProtocol creates a new route protocol.
//
// @input - host, addr protocol, signer, peer manager, payment serving manager,
// 	route store, subscriber store, options.
//
// @output - route protocol.
func NewRouteProtocol(
	h host.Host,
	addrProto *addrproto.AddrProtocol,
	paychProto *paychproto.PaychProtocol,
	signer crypto.Signer,
	peerMgr peermgr.PeerManager,
	pservMgr pservmgr.PaychServingManager,
	rs routestore.RouteStore,
	sbs substore.SubStore,
	opts Opts,
) *RouteProtocol {
	// Parse options
	ioTimeout := opts.IOTimeout
	if ioTimeout == 0 {
		ioTimeout = defaultIOTimeout
	}
	opTimeout := opts.OpTimeout
	if opTimeout == 0 {
		opTimeout = defaultOpTimeout
	}
	publishFreq := opts.PublishFreq
	if publishFreq == 0 {
		publishFreq = defaultPublishFreq
	}
	routeExpiry := opts.RouteExpiry
	if routeExpiry == 0 {
		routeExpiry = defaultRouteExpiry
	}
	publishWait := opts.PublishWait
	if publishWait == 0 {
		publishWait = defaultPublishWait
	}
	log.Infof("Start route protocol...")
	// Create routine
	routineCtx, cancel := context.WithCancel(context.Background())
	proto := &RouteProtocol{
		h:           h,
		addrProto:   addrProto,
		paychProto:  paychProto,
		signer:      signer,
		peerMgr:     peerMgr,
		pservMgr:    pservMgr,
		rs:          rs,
		sbs:         sbs,
		routineCtx:  routineCtx,
		ioTimeout:   ioTimeout,
		opTimeout:   opTimeout,
		publishFreq: publishFreq,
		routeExpiry: routeExpiry,
		publishWait: publishWait,
	}
	// Register publish hook
	paychProto.RegisterPublishFunc(func(currencyID byte, toAddr string) {
		after := time.After(proto.publishWait)
		select {
		case <-after:
			err := proto.Publish(proto.routineCtx, currencyID, toAddr)
			if err != nil {
				log.Warnf("Fail to publish to new subscriber %v-%v: %v", currencyID, toAddr, err.Error())
			}
		case <-proto.routineCtx.Done():
			// Exit
			log.Infof("Shutdown initial route publish routine")
			return
		}
	})
	h.SetStreamHandler(PublishProtocol+ProtocolVersion, proto.handlePublish)
	proto.shutdown = func() {
		log.Infof("Remove stream handlers...")
		// Remove stream handlers
		h.RemoveStreamHandler(PublishProtocol + ProtocolVersion)
		// Deregister hook
		log.Infof("De-register publish hook...")
		paychProto.DeregisterPublishHook()
		// Close routines
		log.Infof("Stop all routines...")
		cancel()
	}
	go proto.publishRoutine()
	return proto
}

// Shutdown safely shuts down the component.
func (p *RouteProtocol) Shutdown() {
	log.Infof("Start shutdown...")
	p.shutdown()
}

// StartPublish publishes the routes to the network immediately.
//
// @input - context.
//
// @output - error.
func (p *RouteProtocol) StartPublish(ctx context.Context) error {
	curOut, errOut0 := p.sbs.ListCurrencyIDs(ctx)
	wg0 := sync.WaitGroup{}
	for currencyID := range curOut {
		wg0.Add(1)
		go func(currencyID byte) {
			defer wg0.Done()
			subOut, errOut1 := p.sbs.ListSubscribers(ctx, currencyID)
			wg1 := sync.WaitGroup{}
			for sub := range subOut {
				wg1.Add(1)
				go func(sub string) {
					defer wg1.Done()
					err2 := p.Publish(ctx, currencyID, sub)
					if err2 != nil {
						log.Errorf("error publish routes to %v-%v: %v", currencyID, sub, err2.Error())
					}
				}(sub)
			}
			wg1.Wait()
			err1 := <-errOut1
			if err1 != nil {
				log.Errorf("error listing subscribers for %v: %v", currencyID, err1.Error())
			}
		}(currencyID)
	}
	wg0.Wait()
	err0 := <-errOut0
	if err0 != nil {
		return err0
	}
	log.Infof("Publish routine - Finished publish")
	return nil
}
