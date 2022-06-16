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
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/cservmgr"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/mpstore"
	"github.com/wcgcyx/fcr/offermgr"
	"github.com/wcgcyx/fcr/peermgr"
	"github.com/wcgcyx/fcr/peernet/protos/addrproto"
	"github.com/wcgcyx/fcr/piecemgr"
)

const (
	QueryOfferProtocol = "/fcr-calibr/cidnet/offer/query"
	ProtocolVersion    = "/0.0.1"
)

// Logger
var log = logging.Logger("cidofferproto")

// OfferProtocol manages offer protocol.
// It handles piece offer creation.
type OfferProtocol struct {
	// Host
	h host.Host

	// Addr proto
	addrProto *addrproto.AddrProtocol

	// DHT Network
	dht *dual.DHT

	// Signer
	signer crypto.Signer

	// Peer manager
	peerMgr peermgr.PeerManager

	// Offer manager
	offerMgr offermgr.OfferManager

	// Piece manager
	pieceMgr piecemgr.PieceManager

	// Serving manager
	cservMgr cservmgr.PieceServingManager

	// Linked miner proof store.
	mps mpstore.MinerProofStore

	// Process related
	routineCtx context.Context

	// Shutdown function.
	shutdown func()

	// Options
	ioTimeout       time.Duration
	opTimeout       time.Duration
	offerExpiry     time.Duration
	offerInactivity time.Duration
	publishFreq     time.Duration
}

// NewOfferProtocol creates a new offer protocol.
//
// @input - host, dht, addr protocol, signer, peer manager, offer manager, piece manager,
//   piece serving manager, miner proof store, options.
//
// @output - offer protocol.
func NewOfferProtocol(
	h host.Host,
	dht *dual.DHT,
	addrProto *addrproto.AddrProtocol,
	signer crypto.Signer,
	peerMgr peermgr.PeerManager,
	offerMgr offermgr.OfferManager,
	pieceMgr piecemgr.PieceManager,
	cservMgr cservmgr.PieceServingManager,
	mps mpstore.MinerProofStore,
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
	publishFreq := opts.PublishFreq
	if publishFreq == 0 {
		publishFreq = defaultPublishFrequency
	}
	log.Infof("Start piece offer protocol...")
	// Create routine
	routineCtx, cancel := context.WithCancel(context.Background())
	proto := &OfferProtocol{
		h:               h,
		dht:             dht,
		addrProto:       addrProto,
		signer:          signer,
		peerMgr:         peerMgr,
		offerMgr:        offerMgr,
		pieceMgr:        pieceMgr,
		cservMgr:        cservMgr,
		mps:             mps,
		routineCtx:      routineCtx,
		ioTimeout:       ioTimeout,
		opTimeout:       opTimeout,
		offerExpiry:     offerExpiry,
		offerInactivity: offerInactivity,
		publishFreq:     publishFreq,
	}
	// Register publish hook
	cservMgr.RegisterPublishFunc(func(id cid.Cid) {
		// TODO: Maybe also publish to indexing service.
		err := proto.dht.Provide(proto.routineCtx, id, true)
		if err != nil {
			if err != nil {
				log.Warnf("Publish routine - Fail to provide %v over DHT: %v", id, err.Error())
			}
		}
	})
	h.SetStreamHandler(QueryOfferProtocol+ProtocolVersion, proto.handleQueryOffer)
	proto.shutdown = func() {
		log.Infof("Remove stream handler...")
		// Remove stream handler
		h.RemoveStreamHandler(QueryOfferProtocol + ProtocolVersion)
		// Close routines
		log.Infof("Stop all routines...")
		cancel()
	}
	go proto.publishRoutine()
	return proto
}

// Shutdown safely shuts down the component.
func (p *OfferProtocol) Shutdown() {
	log.Infof("Start shutdown...")
	p.shutdown()
}

// FindOffersAsync tries to find offers for given cid and given currency.
//
// @input - context, currency id, piece cid, max offers required.
//
// @output - offer chan out.
func (p *OfferProtocol) FindOffersAsync(ctx context.Context, currencyID byte, id cid.Cid, max int) <-chan fcroffer.PieceOffer {
	out := make(chan fcroffer.PieceOffer)
	go func() {
		defer func() {
			close(out)
		}()
		piOut := p.dht.FindProvidersAsync(ctx, id, 0)
		discovered := 0
		lock := sync.RWMutex{}
		var wg sync.WaitGroup
		for pi := range piOut {
			wg.Add(1)
			go func(pi peer.AddrInfo) {
				defer wg.Done()
				toAddr, err := p.addrProto.QueryAddr(ctx, currencyID, pi)
				if err != nil {
					log.Warnf("Error querying address for %v: %v", pi.ID, err.Error())
					return
				}
				offer, err := p.QueryOffer(ctx, currencyID, toAddr, id)
				if err != nil {
					log.Warnf("Error querying offer for %v from %v-%v: %v", id, currencyID, toAddr, err.Error())
					return
				}
				lock.Lock()
				if max != 0 && discovered == max {
					lock.Unlock()
					return
				}
				discovered++
				lock.Unlock()
				select {
				case out <- offer:
				case <-ctx.Done():
					return
				}
			}(pi)
		}
		wg.Wait()
	}()
	return out
}

// Publish offers to the network immediately.
//
// @input - context.
//
// @output - error.
func (p *OfferProtocol) Publish(ctx context.Context) error {
	log.Infof("Publish routine - Start publish")
	pieceChan, errChan := p.cservMgr.ListPieceIDs(ctx)
	for piece := range pieceChan {
		// TODO: Maybe also publish to indexing service.
		err := p.dht.Provide(ctx, piece, true)
		if err != nil {
			return err
		}
	}
	err := <-errChan
	if err != nil {
		return err
	}
	log.Infof("Publish routine - Finished publish")
	return nil
}
