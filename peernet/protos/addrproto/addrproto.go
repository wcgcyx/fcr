package addrproto

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
	"fmt"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/peermgr"
)

const (
	QueryAddrProtocol = "/fcr-calibr/peernet/addr/query"
	ProtocolVersion   = "/0.0.1"
)

// Logger
var log = logging.Logger("addrproto")

// AddrProtocol manages addr protocol.
// It handles resolving peer addr.
type AddrProtocol struct {
	// Host
	h host.Host

	// DHT Network
	dht *dual.DHT

	// Signer
	signer crypto.Signer

	// Peer manager
	peerMgr peermgr.PeerManager

	// Process related
	routineCtx context.Context

	// Shutdown function.
	shutdown func()

	// Options
	ioTimeout   time.Duration
	opTimeout   time.Duration
	publishFreq time.Duration
}

// NewAddrProtocol creates a new addr protocol.
//
// @input - host, dht, signer, peer manager.
func NewAddrProtocol(
	h host.Host,
	dht *dual.DHT,
	signer crypto.Signer,
	peerMgr peermgr.PeerManager,
	opts Opts,
) *AddrProtocol {
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
		publishFreq = defaultPublishFrequency
	}
	log.Infof("Start addr protocol...")
	// Create routine
	routineCtx, cancel := context.WithCancel(context.Background())
	proto := &AddrProtocol{
		h:           h,
		dht:         dht,
		signer:      signer,
		peerMgr:     peerMgr,
		routineCtx:  routineCtx,
		ioTimeout:   ioTimeout,
		opTimeout:   opTimeout,
		publishFreq: publishFreq,
	}
	h.SetStreamHandler(QueryAddrProtocol+ProtocolVersion, proto.handleQueryAddr)
	proto.shutdown = func() {
		log.Infof("Remove stream handler...")
		// Remove stream handler
		h.RemoveStreamHandler(QueryAddrProtocol + ProtocolVersion)
		// Close routines
		log.Infof("Stop all routines...")
		cancel()
	}
	go proto.publishRoutine()
	return proto
}

// Shutdown safely shuts down the component.
func (p *AddrProtocol) Shutdown() {
	log.Infof("Start shutdown...")
	p.shutdown()
}

// ConnectToPeer attempts to establish a connection to peer with given currency id and recipient address pair.
//
// @input - context, operation timeout, io timeout, currency id, to addr.
//
// @output - peer addr id, error.
func (p *AddrProtocol) ConnectToPeer(ctx context.Context, currencyID byte, toAddr string) (peer.ID, error) {
	// First try to load network address from peer manager.
	subCtx, cancel := context.WithTimeout(ctx, p.opTimeout)
	defer cancel()
	// Get peer network address.
	exists, err := p.peerMgr.HasPeer(ctx, currencyID, toAddr)
	if err != nil {
		return "", err
	}
	if exists {
		pi, err := p.peerMgr.PeerAddr(subCtx, currencyID, toAddr)
		if err != nil {
			log.Warnf("Fail to get peer network address for %v-%v: %v", currencyID, toAddr, err.Error())
			return "", err
		}
		// Connect to peer
		subCtx, cancel = context.WithTimeout(ctx, p.ioTimeout)
		defer cancel()
		err = p.h.Connect(subCtx, pi)
		if err == nil {
			return pi.ID, nil
		}
	}
	// Try to serach via DHT and update the peer address.
	rec, err := getAddrRecord(toAddr, currencyID)
	if err != nil {
		log.Warnf("Fail to search peer network address for %v-%v via DHT: %v", currencyID, toAddr, err.Error())
		return "", err
	}
	out := p.dht.FindProvidersAsync(subCtx, rec, 0)
	for pi := range out {
		tempAddr, err := p.QueryAddr(subCtx, currencyID, pi)
		if err != nil {
			return "", err
		}
		if tempAddr == toAddr {
			return pi.ID, nil
		}
	}
	return "", fmt.Errorf("fail to find any peer for %v-%v", currencyID, toAddr)
}

// Publish the presence to the network immediately.
//
// @input - context.
//
// @output - error.
func (p *AddrProtocol) Publish(ctx context.Context) error {
	log.Infof("Publish routine - Start publish")
	curChan, errChan := p.signer.ListCurrencyIDs(ctx)
	for currencyID := range curChan {
		_, toAddr, err := p.signer.GetAddr(ctx, currencyID)
		if err != nil {
			return err
		} else {
			// Publish addr record to DHT. TODO: Maybe also publish to indexing service.
			rec, err := getAddrRecord(toAddr, currencyID)
			if err != nil {
				return err
			} else {
				err := p.dht.Provide(ctx, rec, true)
				if err != nil {
					return err
				}
				log.Infof("Publish routine - Finished publish for %v-%v", currencyID, toAddr)
			}
		}
	}
	return <-errChan
}
