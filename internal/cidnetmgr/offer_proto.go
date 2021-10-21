package cidnetmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wcgcyx/fcr/internal/cidoffer"
	"github.com/wcgcyx/fcr/internal/cidservstore"
	"github.com/wcgcyx/fcr/internal/io"
	"github.com/wcgcyx/fcr/internal/messages"
	"github.com/wcgcyx/fcr/internal/peermgr"
	"github.com/wcgcyx/fcr/internal/piecemgr"
)

// OfferProtocol is the protocol to request and answer offer query.
type OfferProtocol struct {
	// Host addr info
	hAddr peer.AddrInfo
	// Host
	h host.Host
	// Serving store
	ss cidservstore.CIDServStore
	// Piece manager
	ps piecemgr.PieceManager
	// Peer manager
	pem peermgr.PeerManager
	// keystore
	ks datastore.Datastore
	// Linked Miner
	linkedMiner string
	// Linked Miner signature
	linkedMinerSigs map[uint64][]byte
}

// NewOfferProtocol creates a new offer protocol.
// It takes a host addr info, a host, a serving store,
// a piece manager, a key store, a linked miner address,
// a linked miner signature as arguments.
// It returns offer protocol.
func NewOfferProtocol(
	hAddr peer.AddrInfo,
	h host.Host,
	ss cidservstore.CIDServStore,
	ps piecemgr.PieceManager,
	pem peermgr.PeerManager,
	ks datastore.Datastore,
	linkedMiner string,
	linkedMinerSigs map[uint64][]byte,
) *OfferProtocol {
	proto := &OfferProtocol{
		hAddr:           hAddr,
		h:               h,
		ss:              ss,
		ps:              ps,
		pem:             pem,
		ks:              ks,
		linkedMiner:     linkedMiner,
		linkedMinerSigs: linkedMinerSigs,
	}
	h.SetStreamHandler(queryOfferProtocol, proto.handleQueryOffer)
	return proto
}

// QueryOffer is used to query an offer for a given cid.
// It takes a context, a currency id, a cid, a peer id as arguments.
// It returns a cid offer and error.
func (proto *OfferProtocol) QueryOffer(parentCtx context.Context, currencyID uint64, id cid.Cid, pid peer.ID) (*cidoffer.CIDOffer, error) {
	ctx, cancel := context.WithTimeout(parentCtx, ioTimeout)
	defer cancel()
	conn, err := proto.h.NewStream(ctx, pid, queryOfferProtocol)
	if err != nil {
		return nil, fmt.Errorf("error opening stream to peer %v: %v", pid.ShortString(), err.Error())
	}
	defer conn.Close()
	// Write request to stream
	data := messages.EncodeCIDOfferReq(currencyID, id)
	err = io.Write(conn, data, ioTimeout)
	if err != nil {
		return nil, fmt.Errorf("error writing data to stream: %v", err.Error())
	}
	// Read response from stream
	data, err = io.Read(conn, ioTimeout)
	if err != nil {
		return nil, fmt.Errorf("error reading data from stream: %v", err.Error())
	}
	offer, err := messages.DecodeCIDOfferResp(data)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err.Error())
	}
	// Validate offer
	if offer.CurrencyID() != currencyID {
		return nil, fmt.Errorf("invalid offer received: expect currency id of %v got %v", currencyID, offer.CurrencyID())
	}
	if offer.Root() != id {
		return nil, fmt.Errorf("invalid offer received: expect root id of %v got %v", id.String(), offer.Root().String())
	}
	if time.Now().Unix() > offer.Expiration()-int64(cidOfferTTLRoom.Seconds()) {
		return nil, fmt.Errorf("invalid offer received: offer expired at %v, now is %v", offer.Expiration(), time.Now())
	}
	if err = offer.Verify(); err != nil {
		return nil, fmt.Errorf("invalid offer received: fail to verify offer: %v", err.Error())
	}
	_, blocked, _, _, _ := proto.pem.GetPeerInfo(currencyID, offer.ToAddr())
	if blocked {
		return nil, fmt.Errorf("peer has been blocked")
	}
	// Offer validated
	return offer, nil
}

// handleQueryOffer handles a query offer request.
// It takes a connection stream as the argument.
func (proto *OfferProtocol) handleQueryOffer(conn network.Stream) {
	defer conn.Close()
	data, err := io.Read(conn, ioTimeout)
	if err != nil {
		log.Debugf("error reading from stream %v: %v", conn.ID(), err.Error())
		return
	}
	currencyID, id, err := messages.DecodeCIDOfferReq(data)
	if err != nil {
		log.Debugf("error decoding message from stream %v: %v", conn.ID(), err.Error())
		return
	}
	ppb, expiration, err := proto.ss.Inspect(currencyID, id)
	if err != nil {
		log.Debugf("error getting serving for %v: %v", id.String(), err.Error())
		return
	}
	if expiration != 0 {
		// We are retiring this serving.
		log.Debugf("serving for %v is retiring", id.String())
		return
	}
	// Get key
	prv, err := proto.ks.Get(datastore.NewKey(fmt.Sprintf("%v", currencyID)))
	if err != nil {
		log.Debugf("error getting key for currency id %v: %v", currencyID, err.Error())
		return
	}
	// Get size
	_, _, size, _, err := proto.ps.Inspect(id)
	// Creating offer
	var sig []byte
	var ok bool
	if proto.linkedMiner != "" {
		sig, ok = proto.linkedMinerSigs[currencyID]
		if !ok {
			log.Debugf("could not find miner proof for currency id %v", currencyID)
			return
		}
	}
	offer, err := cidoffer.NewCIDOffer(prv, currencyID, proto.hAddr, id, ppb, size, proto.linkedMiner, sig, time.Now().Add(cidOfferTTL).Unix())
	if err != nil {
		log.Debugf("error creating offer: %v", err.Error())
		return
	}
	// Respond with offer.
	data = messages.EncodeCIDOfferResp(*offer)
	err = io.Write(conn, data, ioTimeout)
	if err != nil {
		log.Debugf("error writing data to stream: %v", err.Error())
		return
	}
}
