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
	"encoding/json"
	"fmt"

	"github.com/wcgcyx/fcr/comms"
)

// Publish publishes served route to a given peer.
//
// @input - context, currency id, recipient address.
//
// @output - error.
func (proto *RouteProtocol) Publish(ctx context.Context, currencyID byte, toAddr string) error {
	// Check if peer is blocked.
	subCtx, cancel := context.WithTimeout(ctx, proto.opTimeout)
	defer cancel()
	exists, err := proto.peerMgr.HasPeer(subCtx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to check if contains peer %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	if exists {
		blocked, err := proto.peerMgr.IsBlocked(subCtx, currencyID, toAddr)
		if err != nil {
			log.Warnf("Fail to check if peer %v-%v is blocked: %v", currencyID, toAddr, err.Error())
			return err
		}
		if blocked {
			return fmt.Errorf("peer %v-%v has been blocked", currencyID, toAddr)
		}
	}
	// Connect to peer
	pid, err := proto.addrProto.ConnectToPeer(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to connect to %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	subCtx, cancel = context.WithTimeout(ctx, proto.ioTimeout)
	defer cancel()
	conn, err := proto.h.NewStream(subCtx, pid, PublishProtocol+ProtocolVersion)
	if err != nil {
		log.Debugf("Fail open stream to %v: %v", pid, err.Error())
		return err
	}
	defer conn.Close()
	// Create request.
	req, err := comms.NewRequestOut(ctx, proto.opTimeout, proto.ioTimeout, conn, proto.signer, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to create request: %v", err.Error())
		return err
	}
	// Send request
	routes := make([][]string, 0)
	subCtx, cancel = context.WithCancel(ctx)
	defer cancel()
	toAddrOut, errOut1 := proto.pservMgr.ListRecipients(subCtx, currencyID)
	for from := range toAddrOut {
		routeOut, errOut2 := proto.rs.ListRoutesFrom(subCtx, currencyID, from)
		for route := range routeOut {
			routes = append(routes, route)
			if len(routes) > 250 {
				// TODO:
				// 1. Optimizing routes, so that common routes will be removed, only longest will survive.
				// 2. To make sure each request does not go beyond max request size, use multiple requests (Need to update handler too).
				// At the moment, limit to 250 routes.
				cancel()
			}
		}
		err = <-errOut2
		if err != nil {
			log.Warnf("Fail to list routes from %v-%v: %v", currencyID, from, err.Error())
			return err
		}
	}
	err = <-errOut1
	if err != nil {
		log.Warnf("Fail to list recipients for %v: %v", currencyID, err.Error())
		return err
	}
	type reqJson struct {
		Routes [][]string `json:"routes"`
	}
	data, err := json.Marshal(reqJson{
		Routes: routes,
	})
	if err != nil {
		log.Errorf("Fail to encode request: %v", err.Error())
		return err
	}
	err = req.Send(ctx, proto.opTimeout, proto.ioTimeout, data)
	if err != nil {
		log.Debugf("Fail to send request: %v", err.Error())
		return err
	}
	// Get response
	succeed, data, err := req.GetResponse(ctx, proto.ioTimeout)
	if err != nil {
		log.Debugf("Fail to receive response: %v", err.Error())
		return err
	}
	if !succeed {
		return fmt.Errorf("Peer fail to process request: %v", string(data))
	}
	return nil
}
