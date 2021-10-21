package paynetmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/wcgcyx/fcr/internal/io"
	"github.com/wcgcyx/fcr/internal/messages"
	"github.com/wcgcyx/fcr/internal/paymgr"
	"github.com/wcgcyx/fcr/internal/payoffer"
	"github.com/wcgcyx/fcr/internal/payofferstore"
	"github.com/wcgcyx/fcr/internal/payservstore"
	"github.com/wcgcyx/fcr/internal/peermgr"
	"github.com/wcgcyx/fcr/internal/routestore"
)

// OfferProtocol is the protocol to request and answer offer query.
type OfferProtocol struct {
	// Parent context
	ctx context.Context
	// Host
	h host.Host
	// Route store
	rs routestore.RouteStore
	// Serving store
	ss payservstore.PayServStore
	// Offer store
	os payofferstore.PayOfferStore
	// Peer manager
	pem peermgr.PeerManager
	// Payment manager
	pam paymgr.PaymentManager
}

// NewOfferProtocol creates a new offer protocol.
// It takes a context, a host, a route store, a serv store,
// a offer store, a peer manager, a payment manager as arguments.
// It returns offer protocol.
func NewOfferProtocol(
	ctx context.Context,
	h host.Host,
	rs routestore.RouteStore,
	ss payservstore.PayServStore,
	os payofferstore.PayOfferStore,
	pem peermgr.PeerManager,
	pam paymgr.PaymentManager,
) *OfferProtocol {
	proto := &OfferProtocol{
		ctx: ctx,
		h:   h,
		rs:  rs,
		ss:  ss,
		os:  os,
		pem: pem,
		pam: pam,
	}
	h.SetStreamHandler(queryOfferProtocol, proto.handleQueryOffer)
	return proto
}

// QueryOffer is used to query an offer for a given route.
// It takes a context, a currency id, a route and amount required as arguments.
// It returns a pay offer and error.
func (proto *OfferProtocol) QueryOffer(parentCtx context.Context, currencyID uint64, route []string, amt *big.Int) (*payoffer.PayOffer, error) {
	// Check route length
	if len(route) < 3 || len(route) > proto.rs.MaxHop(currencyID) {
		return nil, fmt.Errorf("route need to have at least length of 3, and not greater than %v, but got %v", proto.rs.MaxHop(currencyID), len(route))
	}
	// Validate route
	if !validRoute(route) {
		return nil, fmt.Errorf("route %v is not acyclic", route)
	}
	// Check root
	root, err := proto.pam.GetRootAddress(currencyID)
	if err != nil {
		return nil, fmt.Errorf("error getting root address for currency id %v", err.Error())
	}
	if root != route[0] {
		return nil, fmt.Errorf("route expect to start with %v, but got %v", root, route[0])
	}
	pid, blocked, _, _, err := proto.pem.GetPeerInfo(currencyID, route[1])
	if err != nil {
		return nil, fmt.Errorf("error in getting peer information for addr %v: %v", route[1], err.Error())
	}
	if blocked {
		return nil, fmt.Errorf("peer %v with addr %v has been blocked", pid.ShortString(), route[1])
	}
	ctx, cancel := context.WithTimeout(parentCtx, ioTimeout*time.Duration(len(route)))
	defer cancel()
	conn, err := proto.h.NewStream(ctx, pid, queryOfferProtocol)
	if err != nil {
		return nil, fmt.Errorf("error opening stream to peer %v: %v", pid.ShortString(), err.Error())
	}
	defer conn.Close()
	// Write request to stream
	data := messages.EncodePayOfferReq(currencyID, route[1:], amt)
	err = io.Write(conn, data, ioTimeout)
	if err != nil {
		return nil, fmt.Errorf("error writing data to stream: %v", err.Error())
	}
	// Read response from stream
	data, err = io.Read(conn, ioTimeout*time.Duration(len(route)-2))
	if err != nil {
		return nil, fmt.Errorf("error reading data from stream: %v", err.Error())
	}
	offer, err := messages.DecodePayOfferResp(data)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err.Error())
	}
	// Validate offer
	if offer.CurrencyID() != currencyID {
		return nil, fmt.Errorf("invalid offer received: expect currency id of %v got %v", currencyID, offer.CurrencyID())
	}
	if offer.From() != route[1] {
		return nil, fmt.Errorf("invalid offer received: expect from %v got %v", route[1], offer.From())
	}
	if offer.To() != route[len(route)-1] {
		return nil, fmt.Errorf("invalid offer received: expect to %v got %v", route[len(route)-1], offer.To())
	}
	if offer.MaxAmt().Cmp(amt) != 0 {
		return nil, fmt.Errorf("invalid offer received: expect amt %v got %v", amt.String(), offer.MaxAmt().String())
	}
	if time.Now().Unix() > offer.CreatedAt()+int64(offer.MaxInactivity().Seconds())-int64(payOfferInactRoom.Seconds()) {
		return nil, fmt.Errorf("invalid offer received: offer created at %v, max inactivity of %v, now is %v", offer.CreatedAt(), offer.MaxInactivity(), time.Now())
	}
	if time.Now().Unix() > offer.Expiration()-int64(payOfferTTLRoom.Seconds()) {
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
func (proto *OfferProtocol) handleQueryOffer(conn network.Stream) {
	defer conn.Close()
	data, err := io.Read(conn, ioTimeout)
	if err != nil {
		log.Debugf("error reading from stream %v: %v", conn.ID(), err.Error())
		return
	}
	currencyID, route, amtRequired, err := messages.DecodePayOfferReq(data)
	if err != nil {
		log.Debugf("error decoding message from stream %v: %v", conn.ID(), err.Error())
		return
	}
	if len(route) < 2 || len(route) > proto.rs.MaxHop(currencyID) {
		log.Debugf("route need to have at least length of 2, and not greater than %v, but got %v", proto.rs.MaxHop(currencyID), len(route))
		return
	}
	// Check root
	root, err := proto.pam.GetRootAddress(currencyID)
	if err != nil {
		log.Debugf("error getting root address for currency id %v", err.Error())
		return
	}
	if root != route[0] {
		log.Debugf("route expect to start with %v, but got %v", root, route[0])
		return
	}
	if len(route) == 2 {
		// This is the final node to create offer.
		// Check if we serve the paych to destination
		ppp, period, expiration, err := proto.ss.Inspect(currencyID, route[1])
		if err != nil {
			log.Debugf("error getting serving for %v: %v", route[1], err.Error())
			return
		}
		if expiration != 0 {
			// We are retiring this serving.
			log.Debugf("serving for %v is retiring", route[1])
			return
		}
		// Construct a new offer
		// Get key
		prv, err := proto.pam.GetPrvKey(currencyID)
		if err != nil {
			log.Debugf("error getting key for currency id %v: %v", currencyID, err.Error())
			return
		}
		// Creating offer
		offer, err := payoffer.NewPayOffer(prv, currencyID, route[1], ppp, period, amtRequired, payOfferInact, time.Now().Add(payOfferTTL).Unix(), cid.Undef)
		if err != nil {
			log.Debugf("error creating offer: %v", err.Error())
			return
		}
		// Reserve amount
		err = proto.pam.ReserveForOthers(offer)
		if err != nil {
			log.Debugf("error reserving offer %v: %v", offer.ID(), err.Error())
			return
		}
		// Respond with offer.
		data = messages.EncodePayOfferResp(*offer)
		err = io.Write(conn, data, ioTimeout)
		if err != nil {
			log.Debugf("error writing data to stream: %v", err.Error())
			return
		}
		return
	}
	// This is an intermediate node.
	// Check if we serve the next stop
	ppp, period, expiration, err := proto.ss.Inspect(currencyID, route[1])
	if err != nil {
		log.Debugf("error getting serving for %v: %v", route[1], err.Error())
		return
	}
	if expiration != 0 {
		// We are retiring this serving.
		log.Debugf("serving for %v is retiring", route[1])
		return
	}
	// Query linkedOffer for this route
	linkedOffer, err := proto.QueryOffer(proto.ctx, currencyID, route, amtRequired)
	if err != nil {
		log.Debugf("error querying linked offer: %v", err.Error())
		return
	}
	// Construct a new offer
	// Get key
	prv, err := proto.pam.GetPrvKey(currencyID)
	if err != nil {
		log.Debugf("error getting key for currency id %v: %v", currencyID, err.Error())
		return
	}
	// The new offer's ppp and period will depend on the implementation, here we use safe option
	// sum(ppp, linked offer ppp), min(period, linked offer period)
	pppNew := big.NewInt(0).Add(ppp, linkedOffer.PPP())
	periodNew := big.NewInt(0)
	if period.Cmp(linkedOffer.Period()) > 0 {
		periodNew = linkedOffer.Period()
	} else {
		periodNew = period
	}
	maxInactivity := linkedOffer.MaxInactivity() - payOfferInactRoom
	if maxInactivity > payOfferInact {
		maxInactivity = payOfferInact
	}
	if maxInactivity < 0 {
		log.Debugf("error creating offer, max inactivity %v is negative", maxInactivity)
		return
	}
	expiration = linkedOffer.Expiration() - int64(payOfferTTLRoom.Seconds())
	if expiration > time.Now().Add(payOfferTTL).Unix() {
		expiration = time.Now().Add(payOfferTTL).Unix()
	}
	if expiration < time.Now().Unix() {
		log.Debugf("error creating offer, expiration %v earlier than now %v", expiration, time.Now().Unix())
		return
	}
	// Creating offer
	offer, err := payoffer.NewPayOffer(prv, currencyID, linkedOffer.To(), pppNew, periodNew, amtRequired, maxInactivity, expiration, linkedOffer.ID())
	if err != nil {
		log.Debugf("error creating offer: %v", err.Error())
		return
	}
	// Add linked offer
	err = proto.os.AddOffer(linkedOffer)
	if err != nil {
		log.Debugf("error adding linked offer: %v", err.Error())
		return
	}
	// Reserve amount
	err = proto.pam.ReserveForOthers(offer)
	if err != nil {
		log.Debugf("error reserving offer %v: %v", offer.ID(), err.Error())
		return
	}
	// Respond with offer.
	data = messages.EncodePayOfferResp(*offer)
	err = io.Write(conn, data, ioTimeout)
	if err != nil {
		log.Debugf("error writing data to stream: %v", err.Error())
		return
	}
}

// validRoute checks if a route is valid.
// It takes a route as the argument.
// It returns a boolean indicating if the route is valid.
func validRoute(nodes []string) bool {
	visited := make(map[string]bool)
	for _, node := range nodes {
		_, ok := visited[node]
		if ok {
			return false
		}
		visited[node] = true
	}
	return true
}
