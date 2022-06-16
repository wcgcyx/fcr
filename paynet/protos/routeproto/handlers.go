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
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/wcgcyx/fcr/comms"
)

// handlePublish handles the route publish request.
//
// @input - network stream.
func (proto *RouteProtocol) handlePublish(conn network.Stream) {
	defer conn.Close()
	// Read request.
	req, err := comms.NewRequestIn(proto.routineCtx, proto.opTimeout, proto.ioTimeout, conn, proto.signer)
	if err != nil {
		log.Debugf("Fail to establish request from %v: %v", conn.ID(), err.Error())
		return
	}
	// Prepare response.
	var respStatus bool
	var respData []byte
	var respErr string
	defer func() {
		var data []byte
		if respStatus {
			data = respData
		} else {
			data = []byte(respErr)
		}
		err = req.Respond(proto.routineCtx, proto.opTimeout, proto.ioTimeout, respStatus, data)
		if err != nil {
			log.Debugf("Error sending response: %v", err.Error())
		}
	}()
	// Get request
	reqData, err := req.Receive(proto.routineCtx, proto.ioTimeout)
	if err != nil {
		log.Debugf("Fail to receive request from stream %v: %v", conn.ID(), err.Error())
		return
	}
	// Start processing request.
	subCtx, cancel := context.WithTimeout(proto.routineCtx, proto.opTimeout)
	defer cancel()
	exists, err := proto.peerMgr.HasPeer(subCtx, req.CurrencyID, req.FromAddr)
	if err != nil {
		respErr = fmt.Sprintf("Internal error")
		log.Warnf("Fail to check if contains peer %v-%v: %v", req.CurrencyID, req.FromAddr, err.Error())
		return
	}
	if exists {
		// Check if peer is blocked.
		blocked, err := proto.peerMgr.IsBlocked(subCtx, req.CurrencyID, req.FromAddr)
		if err != nil {
			respErr = fmt.Sprintf("Internal error")
			log.Warnf("Fail to check if peer %v-%v is blocked: %v", req.CurrencyID, req.FromAddr, err.Error())
			return
		}
		if blocked {
			respErr = fmt.Sprintf("Peer %v-%v is blocked", req.CurrencyID, req.FromAddr)
			log.Debugf("Peer %v-%v has been blocked, stop processing request", req.CurrencyID, req.FromAddr)
			return
		}
	} else {
		// Insert peer.
		pi := peer.AddrInfo{
			ID:    conn.Conn().RemotePeer(),
			Addrs: []multiaddr.Multiaddr{conn.Conn().RemoteMultiaddr()},
		}
		err = proto.peerMgr.AddPeer(subCtx, req.CurrencyID, req.FromAddr, pi)
		if err != nil {
			respErr = fmt.Sprintf("Internal error")
			log.Warnf("Fail to add peer %v-%v with %v: %v", req.CurrencyID, req.FromAddr, pi, err.Error())
			return
		}
	}
	// Decode request
	type reqJson struct {
		Routes [][]string `json:"routes"`
	}
	decoded := reqJson{}
	err = json.Unmarshal(reqData, &decoded)
	if err != nil {
		respErr = fmt.Sprintf("Fail to decode request")
		log.Debugf("Fail to decode request: %v", err.Error())
		return
	}
	wg := sync.WaitGroup{}
	for _, route := range decoded.Routes {
		wg.Add(1)
		go func(route []string) {
			defer wg.Done()
			subCtx, cancel := context.WithTimeout(proto.routineCtx, proto.opTimeout)
			defer cancel()
			err = proto.rs.AddRoute(subCtx, req.CurrencyID, append([]string{req.FromAddr}, route...), proto.routeExpiry)
			if err != nil {
				if !strings.Contains(err.Error(), "route provided is cyclic") {
					log.Warnf("Error adding route %v-%v: %v", req.CurrencyID, route, err.Error())
				}
			}
		}(route)
	}
	wg.Wait()
	// Create response
	respStatus = true
}
