package paynetmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-datastore"
	badgerds "github.com/ipfs/go-ds-badger"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/wcgcyx/fcr/internal/config"
	"github.com/wcgcyx/fcr/internal/paychoffer"
	"github.com/wcgcyx/fcr/internal/paymgr"
	"github.com/wcgcyx/fcr/internal/paymgr/chainmgr"
	"github.com/wcgcyx/fcr/internal/payoffer"
	"github.com/wcgcyx/fcr/internal/payofferstore"
	"github.com/wcgcyx/fcr/internal/payservstore"
	"github.com/wcgcyx/fcr/internal/peermgr"
	"github.com/wcgcyx/fcr/internal/pubsubstore"
	"github.com/wcgcyx/fcr/internal/routestore"
)

const (
	// protocols
	subProtocol        = "/fcr/paynet/subscribe"
	pubProtocol        = "/fcr/paynet/publish"
	queryOfferProtocol = "/fcr/paynet/queryoffer"
	queryPaychProtocol = "fcr/paynet/querypaych"
	addPaychProtocol   = "fcr/paynet/addpaych"
	payProtocol        = "/fcr/paynet/pay"
	// time
	resubInterval     = 1 * time.Hour
	repubInterval     = 1 * time.Hour
	subTTL            = 2 * time.Hour
	pubTTL            = 2 * time.Hour
	payTTL            = 2 * time.Minute
	payOfferInact     = 2 * time.Minute
	payOfferInactRoom = 10 * time.Second
	payOfferTTL       = 10 * time.Minute
	payOfferTTLRoom   = 1 * time.Minute
	paychOfferTTL     = 10 * time.Minute
	paychOfferTTLRoom = 1 * time.Minute
	ioTimeout         = 1 * time.Minute
	// Others
	keySubPath      = "keystore"
	sep             = ";"
	minHop          = 3
	settlementDelay = 168 * time.Hour
	retryAttempt    = 10
	retryDelay      = 2000 * time.Millisecond
)

var log = logging.Logger("paynetmgr")

// PaymentNetworkManagerImplV1 implements PaymentNetworkManager.
type PaymentNetworkManagerImplV1 struct {
	// Parent ctx
	ctx context.Context
	// Host addr info
	hAddr peer.AddrInfo
	// Host
	h host.Host
	// Currency IDs
	currencyIDs []uint64
	// Keystore
	ks datastore.Datastore
	// Route store
	rs routestore.RouteStore
	// Offer store
	os payofferstore.PayOfferStore
	// Serving store
	ss payservstore.PayServStore
	// Peer manager
	pem peermgr.PeerManager
	// Payment manager
	pam paymgr.PaymentManager
	// PubSub store
	pss pubsubstore.PubSubStore
	// Offer protocol
	offerProto *OfferProtocol
	// Record protocol
	recordProto *RecordProtocol
	// Pay protocol
	payProto *PayProtocol
	// Paych protocol
	paychProto *PaychProtocol
}

// NewPaymentNetworkManagerImplV1 creates a payment network manager.
// It takes a context, a host, a host addr info, a config, a keystore as arguments.
// It returns a payment network manager and error.
func NewPaymentNetworkManagerImplV1(ctx context.Context, h host.Host, hAddr peer.AddrInfo, conf config.Config) (PaymentNetworkManager, error) {
	// New keystore
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	ks, err := badgerds.NewDatastore(filepath.Join(conf.RootPath, keySubPath), &dsopts)
	if err != nil {
		return nil, err
	}
	// Initialise settings
	currencyIDs := make([]uint64, 0)
	cms := make(map[uint64]chainmgr.ChainManager)
	maxHops := make(map[uint64]int)
	// For currency 0
	if conf.EnableCur0 {
		pid, err := peer.Decode(conf.ServIDCur0)
		if err != nil {
			return nil, err
		}
		addrs := make([]multiaddr.Multiaddr, 0)
		addrStrs := strings.Split(conf.ServAddrCur0, sep)
		for _, addrStr := range addrStrs {
			addr, err := multiaddr.NewMultiaddr(addrStr)
			if err != nil {
				return nil, err
			}
			addrs = append(addrs, addr)
		}
		cm, err := chainmgr.NewMockChainManagerImplV1(peer.AddrInfo{ID: pid, Addrs: addrs})
		if err != nil {
			return nil, err
		}
		currencyIDs = append(currencyIDs, chainmgr.MockedChainCurrencyID)
		cms[chainmgr.MockedChainCurrencyID] = cm
		if conf.MaxHopCur0 < minHop {
			maxHops[chainmgr.MockedChainCurrencyID] = minHop
		} else {
			maxHops[chainmgr.MockedChainCurrencyID] = conf.MaxHopCur0
		}
		// Check key
		exists, err := ks.Has(datastore.NewKey(fmt.Sprintf("%v", chainmgr.MockedChainCurrencyID)))
		if err != nil {
			return nil, err
		}
		if !exists {
			// Not exists, try to parse the key
			data, err := base64.StdEncoding.DecodeString(conf.KeyCur0)
			if err != nil {
				return nil, err
			}
			if len(data) != 32 {
				return nil, fmt.Errorf("expect a secp256k1 base64 key string but got %v bytes", len(data))
			}
			err = ks.Put(datastore.NewKey(fmt.Sprintf("%v", chainmgr.MockedChainCurrencyID)), data)
			if err != nil {
				return nil, err
			}
		}
	}
	// For currency 1
	if conf.EnableCur1 {
		cm, err := chainmgr.NewFILChainManagerImplV1(conf.ServAddrCur1, conf.ServIDCur1)
		if err != nil {
			return nil, err
		}
		currencyIDs = append(currencyIDs, chainmgr.FILMainnetCurrencyID)
		cms[chainmgr.FILMainnetCurrencyID] = cm
		if conf.MaxHopCur1 < minHop {
			maxHops[chainmgr.FILMainnetCurrencyID] = minHop
		} else {
			maxHops[chainmgr.FILMainnetCurrencyID] = conf.MaxHopCur1
		}
		// Check key
		exists, err := ks.Has(datastore.NewKey(fmt.Sprintf("%v", chainmgr.FILMainnetCurrencyID)))
		if err != nil {
			return nil, err
		}
		if !exists {
			// Not exists, try to parse the key
			data, err := base64.StdEncoding.DecodeString(conf.KeyCur1)
			if err != nil {
				return nil, err
			}
			if len(data) != 32 {
				return nil, fmt.Errorf("expect a secp256k1 base64 key string but got %v bytes", len(data))
			}
			err = ks.Put(datastore.NewKey(fmt.Sprintf("%v", chainmgr.FILMainnetCurrencyID)), data)
			if err != nil {
				return nil, err
			}
		}
	}
	// For currency 2
	if conf.EnableCur2 {
		cm, err := chainmgr.NewFILChainManagerImplV1(conf.ServAddrCur2, conf.ServIDCur2)
		if err != nil {
			return nil, err
		}
		currencyIDs = append(currencyIDs, chainmgr.FILCalibrationCurrencyID)
		cms[chainmgr.FILCalibrationCurrencyID] = cm
		if conf.MaxHopCur2 < minHop {
			maxHops[chainmgr.FILCalibrationCurrencyID] = minHop
		} else {
			maxHops[chainmgr.FILCalibrationCurrencyID] = conf.MaxHopCur2
		}
		// Check key
		exists, err := ks.Has(datastore.NewKey(fmt.Sprintf("%v", chainmgr.FILCalibrationCurrencyID)))
		if err != nil {
			return nil, err
		}
		if !exists {
			// Not exists, try to parse the key
			data, err := base64.StdEncoding.DecodeString(conf.KeyCur2)
			if err != nil {
				return nil, err
			}
			if len(data) != 32 {
				return nil, fmt.Errorf("expect a secp256k1 base64 key string but got %v bytes", len(data))
			}
			err = ks.Put(datastore.NewKey(fmt.Sprintf("%v", chainmgr.FILCalibrationCurrencyID)), data)
			if err != nil {
				return nil, err
			}
		}
	}
	// TODO: Process more currency IDs.
	// If nothing is configured, return error
	if len(currencyIDs) == 0 {
		return nil, fmt.Errorf("at least 1 currency ID should be configured.")
	}
	// Initialise offer store
	os, err := payofferstore.NewPayOfferStoreImplV1(ctx, conf.RootPath)
	if err != nil {
		return nil, err
	}
	// Initialise payment manager
	pam, err := paymgr.NewPaymentManagerImplV1(ctx, conf.RootPath, currencyIDs, cms, ks, os)
	if err != nil {
		return nil, err
	}
	// Initialise route store
	roots := make(map[uint64]string)
	for _, currencyID := range currencyIDs {
		addr, err := pam.GetRootAddress(currencyID)
		if err != nil {
			return nil, err
		}
		roots[currencyID] = addr
	}
	rs, err := routestore.NewRouteStoreImplV1(ctx, conf.RootPath, currencyIDs, maxHops, roots)
	if err != nil {
		return nil, err
	}
	// Initialise serving store
	ss, err := payservstore.NewPayServStoreImplV1(ctx, conf.RootPath, currencyIDs)
	if err != nil {
		return nil, err
	}
	// Initialise peer manager
	pem, err := peermgr.NewPeerManagerImplV1(ctx, conf.RootPath, "pay", currencyIDs)
	if err != nil {
		return nil, err
	}
	// Initialise pubsub store
	pss, err := pubsubstore.NewPubSubStoreImplV1(ctx, conf.RootPath, currencyIDs)
	if err != nil {
		return nil, err
	}
	// Initialise offer protocol
	offerProto := NewOfferProtocol(ctx, h, rs, ss, os, pem, pam)
	// Initialise record protocol
	recordProto := NewRecordProtocol(ctx, hAddr, h, rs, ss, pss)
	// Initialise pay protocol
	payProto := NewPayProtocol(ctx, h, rs, os, pem, pam)
	// Initialise paych protocol
	paychProto := NewPaychProtocol(h, settlementDelay, pam)
	mgr := &PaymentNetworkManagerImplV1{
		ctx:         ctx,
		hAddr:       hAddr,
		h:           h,
		currencyIDs: currencyIDs,
		ks:          ks,
		rs:          rs,
		os:          os,
		ss:          ss,
		pem:         pem,
		pam:         pam,
		pss:         pss,
		offerProto:  offerProto,
		recordProto: recordProto,
		payProto:    payProto,
		paychProto:  paychProto,
	}
	go mgr.shutdownRoutine()
	return mgr, nil
}

// QueryOutboundChOffer is used to query a peer for a outbound channel offer.
// It takes a context, a currency ID, peer addr info as arguments.
// It returns a paych offer and error.
func (mgr *PaymentNetworkManagerImplV1) QueryOutboundChOffer(ctx context.Context, currencyID uint64, toAddr string, pi peer.AddrInfo) (*paychoffer.PaychOffer, error) {
	return mgr.paychProto.QueryOffer(ctx, currencyID, toAddr, pi)
}

// CreateOutboundCh is used to create an outbound channel.
// It takes a context, a paych offer, amt, peer addr info as arguments.
// It returns error.
func (mgr *PaymentNetworkManagerImplV1) CreateOutboundCh(ctx context.Context, offer *paychoffer.PaychOffer, amt *big.Int, pi peer.AddrInfo) error {
	if time.Now().Unix() > offer.Expiration()-int64(paychOfferTTLRoom.Seconds()) {
		return fmt.Errorf("invalid offer received: offer expired at %v, now is %v", offer.Expiration(), time.Now())
	}
	chAddr, err := mgr.pam.CreateOutboundCh(ctx, offer.CurrencyID(), offer.Addr(), amt, offer.Settlement())
	if err != nil {
		return fmt.Errorf("error creating outbound channel: %v", err.Error())
	}
	// Channel created, need to register it to the recipient
	err = mgr.paychProto.AddPaych(ctx, offer.CurrencyID(), chAddr, offer, pi.ID)
	if err != nil {
		return fmt.Errorf("error adding paych %v to recipient: %v", chAddr, err.Error())
	}
	// Channel is registered. Need to add to peer manager
	err = mgr.pem.AddPeer(offer.CurrencyID(), offer.Addr(), pi.ID)
	if err != nil {
		return fmt.Errorf("error adding peer to peer manager: %v", err.Error())
	}
	// Need to subscribe for any routes.
	err = mgr.recordProto.Subscribe(offer.CurrencyID(), pi.ID)
	if err != nil {
		return fmt.Errorf("error subscribe peer: %v", err.Error())
	}
	// Add to route store
	err = mgr.rs.AddDirectLink(offer.CurrencyID(), offer.Addr())
	if err != nil {
		return fmt.Errorf("error adding direct link: %v", err.Error())
	}
	return nil
}

// TopupOutboundCh is used to topup an active outbound channel.
// It takes a context, a currency id, recipient address, amt as arguments.
// It returns error.
func (mgr *PaymentNetworkManagerImplV1) TopupOutboundCh(ctx context.Context, currencyID uint64, toAddr string, amt *big.Int) error {
	return mgr.pam.TopupOutboundCh(ctx, currencyID, toAddr, amt)
}

// ListActiveOutboundChs lists all active outbound channels.
// It takes a context as the argument.
// It returns a map from currency id to a list of channel addresses and error.
func (mgr *PaymentNetworkManagerImplV1) ListActiveOutboundChs(ctx context.Context) (map[uint64][]string, error) {
	return mgr.pam.ListActiveOutboundChs(ctx)
}

// ListOutboundChs lists all outbound channels.
// It takes a context as the argument.
// It returns a map from currency id to a list of channel addresses and error.
func (mgr *PaymentNetworkManagerImplV1) ListOutboundChs(ctx context.Context) (map[uint64][]string, error) {
	return mgr.pam.ListOutboundChs(ctx)
}

// InspectOutboundCh inspects an outbound payment channel.
// It takes a context, currency id and channel address as arguments.
// It returns recipient address, redeemed amount, total amount, settlement time of this channel (as specified when creating this channel),
// boolean indicates if channel is active, current chain height, settling height and error.
func (mgr *PaymentNetworkManagerImplV1) InspectOutboundCh(ctx context.Context, currencyID uint64, chAddr string) (string, *big.Int, *big.Int, int64, bool, int64, int64, error) {
	return mgr.pam.InspectOutboundCh(ctx, currencyID, chAddr)
}

// SettleOutboundCh settles an outbound payment channel.
// It takes a context, currency id and channel address as arguments.
// It returns the error.
func (mgr *PaymentNetworkManagerImplV1) SettleOutboundCh(ctx context.Context, currencyID uint64, chAddr string) error {
	return mgr.pam.SettleOutboundCh(ctx, currencyID, chAddr)
}

// CollectOutboundCh collects an outbound payment channel.
// It takes a context, currency id and channel address as arguments.
// It returns the error.
func (mgr *PaymentNetworkManagerImplV1) CollectOutboundCh(ctx context.Context, currencyID uint64, chAddr string) error {
	return mgr.pam.CollectOutboundCh(ctx, currencyID, chAddr)
}

// SetInboundChOffer sets the inbound channel settlement delay.
// It takes a settlement delay as the argument.
func (mgr *PaymentNetworkManagerImplV1) SetInboundChOffer(settlementDelay time.Duration) {
	mgr.paychProto.UpdateDelay(settlementDelay)
}

// ListInboundChs lists all inbound channels.
// It takes a context as the argument.
// It returns a map from currency id to list of paych addresses and error.
func (mgr *PaymentNetworkManagerImplV1) ListInboundChs(ctx context.Context) (map[uint64][]string, error) {
	return mgr.pam.ListInboundChs(ctx)
}

// InspectInboundCh inspects an inbound payment channel.
// It takes a context, currency id and channel address as arguments.
// It returns from address, redeemed amount, total amount, settlement time of this channel (as specified when creating this channel),
// boolean indicates if channel is active, current chain height, settling height and error.
func (mgr *PaymentNetworkManagerImplV1) InspectInboundCh(ctx context.Context, currencyID uint64, chAddr string) (string, *big.Int, *big.Int, int64, bool, int64, int64, error) {
	return mgr.pam.InspectInboundCh(ctx, currencyID, chAddr)
}

// SettleInboundCh settles an inbound payment channel.
// It takes a context, currency id and channel address as arguments.
// It returns the error.
func (mgr *PaymentNetworkManagerImplV1) SettleInboundCh(ctx context.Context, currencyID uint64, chAddr string) error {
	return mgr.pam.SettleInboundCh(ctx, currencyID, chAddr)
}

// CollectInboundCh collects an inbound payment channel.
// It takes a context, currency id and channel address as arguments.
// It returns the error.
func (mgr *PaymentNetworkManagerImplV1) CollectInboundCh(ctx context.Context, currencyID uint64, chAddr string) error {
	return mgr.pam.CollectInboundCh(ctx, currencyID, chAddr)
}

// KeyStore gets the keystore of this manager.
// It returns the key store.
func (mgr *PaymentNetworkManagerImplV1) KeyStore() datastore.Datastore {
	return mgr.ks
}

// CurrencyIDs gets the currency id supported by this manager.
// It returns a list of currency IDs.
func (mgr *PaymentNetworkManagerImplV1) CurrencyIDs() []uint64 {
	return mgr.currencyIDs
}

// GetRootAddress gets the root address of given currency id.
// It takes a currency id as the argument.
// It returns the root address and error if not existed.
func (mgr *PaymentNetworkManagerImplV1) GetRootAddress(currencyID uint64) (string, error) {
	return mgr.pam.GetRootAddress(currencyID)
}

// Reserve will reserve given amount for a period of time.
// It takes a currency id, recipient address and amount as arguments.
// It returns error.
func (mgr *PaymentNetworkManagerImplV1) Reserve(currencyID uint64, toAddr string, amt *big.Int) error {
	for i := 0; i < retryAttempt; i++ {
		if err := mgr.pam.ReserveForSelf(currencyID, toAddr, amt); err == nil {
			return nil
		}
		time.Sleep(retryDelay)
	}
	return mgr.pam.ReserveForSelf(currencyID, toAddr, amt)
}

// Pay will pay given amount to a recipient.
// It takes a context, a currency id, recipient address and amount as arguments.
// It returns error.
func (mgr *PaymentNetworkManagerImplV1) Pay(ctx context.Context, currencyID uint64, toAddr string, amt *big.Int) error {
	for i := 0; i < retryAttempt; i++ {
		if err := mgr.payProto.Pay(ctx, currencyID, toAddr, amt); err == nil {
			return nil
		}
		time.Sleep(retryDelay)
	}
	return mgr.payProto.Pay(ctx, currencyID, toAddr, amt)
}

// SearchPayOffer will ask all active peers for pay offers to given recipient with given amount.
// It takes a context, a currency id, recipient address and total amount as arguments.
// It returns a list of pay offers and error.
func (mgr *PaymentNetworkManagerImplV1) SearchPayOffer(ctx context.Context, currencyID uint64, toAddr string, amt *big.Int) ([]payoffer.PayOffer, error) {
	routes, err := mgr.rs.GetRoutesTo(currencyID, toAddr)
	if err != nil {
		return nil, fmt.Errorf("error getting routes to currency id %v to addr %v: %v", currencyID, toAddr, err.Error())
	}
	lock := sync.RWMutex{}
	res := make([]payoffer.PayOffer, 0)
	var wg sync.WaitGroup
	for _, route := range routes {
		wg.Add(1)
		go func(route []string) {
			defer wg.Done()
			var offer *payoffer.PayOffer
			var err error
			for i := 0; i < retryAttempt; i++ {
				offer, err = mgr.offerProto.QueryOffer(ctx, currencyID, route, amt)
				if err == nil {
					break
				}
				time.Sleep(retryDelay)
			}
			if err != nil {
				offer, err = mgr.offerProto.QueryOffer(ctx, currencyID, route, amt)
				if err != nil {
					log.Errorf("error querying offer: %v", err.Error())
					return
				}
			}
			lock.Lock()
			defer lock.Unlock()
			res = append(res, *offer)
		}(route)
	}
	wg.Wait()
	return res, nil
}

// ReserveWithPayOffer will reserve given amount for an offer.
// It takes a pay offer as the argument.
// It returns error.
func (mgr *PaymentNetworkManagerImplV1) ReserveWithPayOffer(offer *payoffer.PayOffer) error {
	for i := 0; i < retryAttempt; i++ {
		if err := mgr.pam.ReserveForSelfWithOffer(offer); err == nil {
			return nil
		}
		time.Sleep(retryDelay)
	}
	return mgr.pam.ReserveForSelfWithOffer(offer)
}

// PayWithOffer will pay given amount using an offer.
// It takes a context, a pay offer and amount as arguments.
// It returns error.
func (mgr *PaymentNetworkManagerImplV1) PayWithOffer(ctx context.Context, offer *payoffer.PayOffer, amt *big.Int) error {
	for i := 0; i < retryAttempt; i++ {
		if err := mgr.payProto.PayWithOffer(ctx, amt, offer); err == nil {
			return nil
		}
		time.Sleep(retryDelay)
	}
	return mgr.payProto.PayWithOffer(ctx, amt, offer)
}

// Receive will receive from origin addr.
// It takes a currency id and origin address as arguments.
// It return amount received and error.
func (mgr *PaymentNetworkManagerImplV1) Receive(currencyID uint64, originAddr string) (*big.Int, error) {
	return mgr.pam.Receive(currencyID, originAddr)
}

// StartServing starts serving a payment channel for others to use.
// It takes a currency id, to address, ppp (price per period), period as arguments.
// It returns error.
func (mgr *PaymentNetworkManagerImplV1) StartServing(currencyID uint64, toAddr string, ppp *big.Int, period *big.Int) error {
	err := mgr.ss.Serve(currencyID, toAddr, ppp, period)
	if err != nil {
		return err
	}
	mgr.recordProto.Publish(currencyID, toAddr)
	return nil
}

// ListServings lists all servings.
// It takes a context as the argument.
// It returns a map from currency id to list of active servings and error.
func (mgr *PaymentNetworkManagerImplV1) ListServings(ctx context.Context) (map[uint64][]string, error) {
	res1, err := mgr.ss.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	res2, err := mgr.ss.ListRetiring(ctx)
	if err != nil {
		return nil, err
	}
	for currencyID, actives := range res1 {
		retires, ok := res2[currencyID]
		if !ok {
			res2[currencyID] = actives
		} else {
			res2[currencyID] = append(actives, retires...)
		}
	}
	return res2, nil
}

// InspectServing will check a given serving.
// It takes a currency id, the to address as arguments.
// It returns the ppp (price per period), period, expiration and error.
func (mgr *PaymentNetworkManagerImplV1) InspectServing(currencyID uint64, toAddr string) (*big.Int, *big.Int, int64, error) {
	return mgr.ss.Inspect(currencyID, toAddr)
}

// StopServing stops a serving for payment channel, it will put a channel into retirement before it automatically expires.
// It takes a currency id, the to address as arguments.
// It returns the error.
func (mgr *PaymentNetworkManagerImplV1) StopServing(currencyID uint64, toAddr string) error {
	return mgr.ss.Retire(currencyID, toAddr, payOfferTTL)
}

// ForcePublishServings will force the manager to do a publication of current servings immediately.
func (mgr *PaymentNetworkManagerImplV1) ForcePublishServings() {
	mgr.recordProto.ForcePublish()
}

// ListPeers is used to list all the peers.
// It takes a context as the argument.
// It returns a map from currency id to list of peers and error.
func (mgr *PaymentNetworkManagerImplV1) ListPeers(ctx context.Context) (map[uint64][]string, error) {
	return mgr.pem.ListPeers(ctx)
}

// GetPeerInfo gets the information of a peer.
// It takes a currency id, peer address as arguments.
// It returns the peer id, a boolean indicate whether it is blocked, success count, failure count and error.
func (mgr *PaymentNetworkManagerImplV1) GetPeerInfo(currencyID uint64, toAddr string) (peer.ID, bool, int, int, error) {
	return mgr.pem.GetPeerInfo(currencyID, toAddr)
}

// BlockPeer is used to block a peer.
// It takes a currency id, peer address to block as arguments.
// It returns error.
func (mgr *PaymentNetworkManagerImplV1) BlockPeer(currencyID uint64, toAddr string) error {
	return mgr.pem.BlockPeer(currencyID, toAddr)
}

// UnblockPeer is used to unblock a peer.
// It takes a currency id, peer address to unblock as arguments.
// It returns error.
func (mgr *PaymentNetworkManagerImplV1) UnblockPeer(currencyID uint64, toAddr string) error {
	return mgr.pem.UnblockPeer(currencyID, toAddr)
}

// shutdownRoutine is used to safely close the routine.
func (mgr *PaymentNetworkManagerImplV1) shutdownRoutine() {
	<-mgr.ctx.Done()
	if err := mgr.ks.Close(); err != nil {
		log.Errorf("error closing storage for keystore: %v", err.Error())
	}
}
