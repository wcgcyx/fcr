package paynetmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wcgcyx/fcr/internal/io"
	"github.com/wcgcyx/fcr/internal/messages"
	"github.com/wcgcyx/fcr/internal/payservstore"
	"github.com/wcgcyx/fcr/internal/pubsubstore"
	"github.com/wcgcyx/fcr/internal/routestore"
)

// RecordProtocol is the protocol to publish and subscribe for payment routes.
type RecordProtocol struct {
	// Parent context
	ctx context.Context
	// Host addr info
	hAddr peer.AddrInfo
	// Host
	h host.Host
	// Route store
	rs routestore.RouteStore
	// Serving store
	ss payservstore.PayServStore
	// PubSub store
	pss pubsubstore.PubSubStore
	// Force publish channel
	fpc chan bool
}

// NewRecordProtocol creates a new record protocol.
// It takes a context, host addr, host, route store, serv store, pubsub store as arguments.
// It returns record protocol.
func NewRecordProtocol(
	ctx context.Context,
	hAddr peer.AddrInfo,
	h host.Host,
	rs routestore.RouteStore,
	ss payservstore.PayServStore,
	pss pubsubstore.PubSubStore,
) *RecordProtocol {
	proto := &RecordProtocol{
		ctx:   ctx,
		hAddr: hAddr,
		h:     h,
		rs:    rs,
		ss:    ss,
		pss:   pss,
		fpc:   make(chan bool),
	}
	h.SetStreamHandler(subProtocol, proto.handleSubscribe)
	h.SetStreamHandler(pubProtocol, proto.handlePublish)
	go proto.subRoutine()
	go proto.pubRoutine()
	return proto
}

// Subscribe will subscribe to a given peer for given currency id.
// It takes a currency id and peer id as arguments.
func (proto *RecordProtocol) Subscribe(currencyID uint64, pid peer.ID) error {
	err := proto.pss.AddSubscribing(currencyID, pid)
	if err != nil {
		return err
	}
	// Start a routine to subscribe immediately
	go proto.subscribe(currencyID, pid)
	return nil
}

// handleSubscribe handles subscribe request.
// It takes a connection stream as the argument.
func (proto *RecordProtocol) handleSubscribe(conn network.Stream) {
	defer conn.Close()
	data, err := io.Read(conn, ioTimeout)
	if err != nil {
		log.Debugf("error reading data from stream: %v", err.Error())
		return
	}
	currencyID, pi, err := messages.DecodeSubscribe(data)
	if err != nil {
		log.Debugf("error decoding message: %v", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(proto.ctx, ioTimeout)
	defer cancel()
	err = proto.h.Connect(ctx, pi)
	if err != nil {
		log.Debugf("error in connecting to peer: %v", err.Error())
		return
	}
	// Succeed, save to storage
	err = proto.pss.AddSubscriber(currencyID, pi.ID, subTTL)
	if err != nil {
		log.Errorf("error adding subscriber: %v", err.Error())
		return
	}
	// Do a publish immediately to the new subsriber.
	routesMap := make(map[uint64][][]string)
	activeMap, err := proto.ss.ListActive(proto.ctx)
	if err != nil {
		log.Errorf("error in listing active servings: %v", err.Error())
	} else {
		for currencyID, servings := range activeMap {
			routes := make([][]string, 0)
			for _, serving := range servings {
				temp, err := proto.rs.GetRoutesFrom(currencyID, serving)
				if err != nil {
					log.Errorf("error getting routes from currency id %v, serving %v", currencyID, serving)
					continue
				}
				routes = append(routes, temp...)
			}
			routesMap[currencyID] = routes
		}
	}
	go proto.publish(currencyID, pi.ID, routesMap)
}

// Publish will publish a new serving to subscribers.
// It takes a currency id and new serving address as arguments.
func (proto *RecordProtocol) Publish(currencyID uint64, newServedAddr string) {
	routesMap := make(map[uint64][][]string, 0)
	routes, err := proto.rs.GetRoutesFrom(currencyID, newServedAddr)
	if err != nil {
		log.Errorf("error getting routes from %v with currency id %v: %v", newServedAddr, currencyID, err.Error())
		return
	}
	routesMap[currencyID] = routes
	// Get subscribers for this currency id
	peersMap, err := proto.pss.ListSubscribers(proto.ctx)
	if err != nil {
		log.Errorf("error listing subscribers: %v", err.Error())
		return
	}
	peers, ok := peersMap[currencyID]
	if ok {
		for _, peer := range peers {
			// Start a routine to publish immediately
			go proto.publish(currencyID, peer, routesMap)
		}
	}
}

// ForcePublish will force to publish all servings immediately.
func (proto *RecordProtocol) ForcePublish() {
	go func() {
		proto.fpc <- true
	}()
}

// handlePublish handles publish request.
// It takes a connection stream as the argument.
func (proto *RecordProtocol) handlePublish(conn network.Stream) {
	defer conn.Close()
	data, err := io.Read(conn, ioTimeout)
	if err != nil {
		log.Debugf("error reading data from stream: %v", err.Error())
		return
	}
	currencyID, routes, err := messages.DecodePublish(data)
	if err != nil {
		log.Debugf("error decoding message: %v", err.Error())
		return
	}
	for _, route := range routes {
		proto.rs.AddRoute(currencyID, route, pubTTL)
	}
}

// subRoutine is a routine to publish subscribe messages.
func (proto *RecordProtocol) subRoutine() {
	for {
		// resub
		subMap, err := proto.pss.ListSubscribers(proto.ctx)
		if err != nil {
			log.Errorf("error listing subscribers: %v", err.Error())
			return
		}
		for currencyID, subscribings := range subMap {
			for _, subscribing := range subscribings {
				go proto.subscribe(currencyID, subscribing)
			}
		}
		tc := time.After(resubInterval)
		select {
		case <-proto.ctx.Done():
			return
		case <-tc:
		}
	}
}

// pubRoutine is a routine to publish routes.
func (proto *RecordProtocol) pubRoutine() {
	for {
		// repub
		routesMap := make(map[uint64][][]string)
		activeMap, err := proto.ss.ListActive(proto.ctx)
		if err != nil {
			log.Errorf("error in listing active servings: %v", err.Error())
		} else {
			for currencyID, servings := range activeMap {
				routes := make([][]string, 0)
				for _, serving := range servings {
					temp, err := proto.rs.GetRoutesFrom(currencyID, serving)
					if err != nil {
						log.Errorf("error getting routes from currency id %v, serving %v", currencyID, serving)
						continue
					}
					routes = append(routes, temp...)
				}
				routesMap[currencyID] = routes
			}
			subMap, err := proto.pss.ListSubscribers(proto.ctx)
			if err != nil {
				log.Errorf("error listing subscribers: %v", err.Error())
				return
			}
			for currencyID, subscribers := range subMap {
				for _, subscriber := range subscribers {
					go proto.publish(currencyID, subscriber, routesMap)
				}
			}
		}
		tc := time.After(repubInterval)
		select {
		case <-proto.ctx.Done():
			return
		case <-tc:
		case <-proto.fpc:
		}
	}
}

// subscribe sends a subscribe message to given peer.
// It takes a currency id and peer id as arguments.
func (proto *RecordProtocol) subscribe(currencyID uint64, pid peer.ID) {
	ctx, cancel := context.WithTimeout(proto.ctx, ioTimeout)
	defer cancel()
	conn, err := proto.h.NewStream(ctx, pid, subProtocol)
	if err != nil {
		log.Debugf("error opening stream to peer %v: %v", pid.ShortString(), err.Error())
		return
	}
	defer conn.Close()
	// Write request to stream
	data := messages.EncodeSubscribe(currencyID, proto.hAddr)
	err = io.Write(conn, data, ioTimeout)
	if err != nil {
		log.Debugf("error writing data to stream: %v", err.Error())
		return
	}
}

// publish sends a publish message to given peer.
// It takes a currency id, peer id, routes map as arguments.
func (proto *RecordProtocol) publish(currencyID uint64, pid peer.ID, routesMap map[uint64][][]string) {
	routes, ok := routesMap[currencyID]
	if !ok {
		log.Debugf("currency id %v not supported", currencyID)
		return
	}
	if len(routes) == 0 {
		log.Debugf("currency id %v does not have any route in serve", currencyID)
		return
	}
	ctx, cancel := context.WithTimeout(proto.ctx, ioTimeout)
	defer cancel()
	conn, err := proto.h.NewStream(ctx, pid, pubProtocol)
	if err != nil {
		log.Debugf("error opening stream to peer %v: %v", pid.ShortString(), err.Error())
		return
	}
	defer conn.Close()
	// Write request to stream
	data := messages.EncodePublish(currencyID, routes)
	err = io.Write(conn, data, ioTimeout)
	if err != nil {
		log.Debugf("error writing data to stream: %v", err.Error())
		return
	}
}
