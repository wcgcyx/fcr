package node

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	noise "github.com/libp2p/go-libp2p-noise"
	routing "github.com/libp2p/go-libp2p-routing"
	libp2ptls "github.com/libp2p/go-libp2p-tls"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/wcgcyx/fcr/internal/cidnetmgr"
	"github.com/wcgcyx/fcr/internal/config"
	"github.com/wcgcyx/fcr/internal/paynetmgr"
	"github.com/wcgcyx/fcr/internal/version"
)

const (
	p2pKeyFilename = "p2pkey"
	protocolPrefix = "/fc-retrieval"
)

// Node is the retrieval node.
type Node struct {
	// Parent ctx
	ctx context.Context
	// Host
	h host.Host
	// CID network manager
	cidnetMgr cidnetmgr.CIDNetworkManager
	// Payment network manager
	paynetMgr paynetmgr.PaymentNetworkManager
}

// NewNode creates a new retrieval node.
// It takes a context, a configuration as arguments.
// It returns a retrieval node and error.
func NewNode(ctx context.Context, conf config.Config) (*Node, error) {
	// First check if root path has been initialised.
	// Try to load the p2p key
	keyBytes, err := ioutil.ReadFile(filepath.Join(conf.RootPath, p2pKeyFilename))
	if err != nil {
		return nil, fmt.Errorf("error loading p2p key: %v", err.Error())
	}
	// Decode key
	key, err := base64.StdEncoding.DecodeString(string(keyBytes))
	if err != nil {
		return nil, fmt.Errorf("error decoding p2p key: %v", err.Error())
	}
	prv, err := crypto.UnmarshalEd25519PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling p2p key: %v", err.Error())
	}
	var dualDHT *dual.DHT
	h, err := libp2p.New(
		ctx,
		libp2p.Identity(prv),
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%v", conf.P2PPort),
		),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Security(noise.ID, noise.New),
		libp2p.DefaultTransports,
		libp2p.ConnectionManager(connmgr.NewConnManager(
			100,         // Lowwater
			400,         // HighWater,
			time.Minute, // GracePeriod
		)),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			dualDHT, err = dual.New(ctx, h, dual.DHTOption(dht.ProtocolPrefix(protocolPrefix)))
			return dualDHT, err
		}),
		libp2p.EnableAutoRelay(),
		libp2p.UserAgent("go-fcr-"+version.Version),
	)
	hAddr := peer.AddrInfo{ID: h.ID(), Addrs: h.Addrs()}
	// Initialise a payment network manager
	paynetMgr, err := paynetmgr.NewPaymentNetworkManagerImplV1(ctx, h, hAddr, conf)
	if err != nil {
		return nil, fmt.Errorf("error initialising payment network manager: %v", err.Error())
	}
	// Initialise a cid network manager
	cidnetMgr, err := cidnetmgr.NewCIDNetworkManagerImplV1(ctx, h, dualDHT, hAddr, conf, paynetMgr)
	if err != nil {
		return nil, fmt.Errorf("error initialising cid network manager: %v", err.Error())
	}
	node := &Node{
		ctx:       ctx,
		h:         h,
		cidnetMgr: cidnetMgr,
		paynetMgr: paynetMgr,
	}
	go node.shutdownRoutine()
	return node, nil
}

// AddrInfo is used to obtain the addr info of this node.
// It returns the addr info.
func (node *Node) AddrInfo() peer.AddrInfo {
	return peer.AddrInfo{ID: node.h.ID(), Addrs: node.h.Addrs()}
}

// Connect is used to connect to a node.
// It takes a context and peer addr info as arguments.
// It returns the error.
func (node *Node) Connect(ctx context.Context, pi peer.AddrInfo) error {
	return node.h.Connect(ctx, pi)
}

// CIDNetworkManager is used to obtain the cid network manager.
// It returns the cid network manager.
func (node *Node) CIDNetworkManager() cidnetmgr.CIDNetworkManager {
	return node.cidnetMgr
}

// PaymentNetworkManager is used to obtain the payment network manager.
// It returns the payment network manager.
func (node *Node) PaymentNetworkManager() paynetmgr.PaymentNetworkManager {
	return node.paynetMgr
}

// shutdownRoutine is used to safely close the routine.
func (node *Node) shutdownRoutine() {
	<-node.ctx.Done()
	node.h.Close()
}
