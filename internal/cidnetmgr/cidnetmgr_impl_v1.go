package cidnetmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	badgerds "github.com/ipfs/go-ds-badger"
	gs "github.com/ipfs/go-graphsync"
	graphsync "github.com/ipfs/go-graphsync/impl"
	gsnet "github.com/ipfs/go-graphsync/network"
	"github.com/ipfs/go-graphsync/storeutil"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	files "github.com/ipfs/go-ipfs-files"
	logging "github.com/ipfs/go-log"
	"github.com/ipfs/go-merkledag"
	unixfile "github.com/ipfs/go-unixfs/file"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/ipld/go-ipld-prime/traversal/selector"
	"github.com/ipld/go-ipld-prime/traversal/selector/builder"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	dhtpvds "github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/wcgcyx/fcr/internal/cidoffer"
	"github.com/wcgcyx/fcr/internal/cidservstore"
	"github.com/wcgcyx/fcr/internal/config"
	"github.com/wcgcyx/fcr/internal/crypto"
	"github.com/wcgcyx/fcr/internal/paynetmgr"
	"github.com/wcgcyx/fcr/internal/payoffer"
	"github.com/wcgcyx/fcr/internal/peermgr"
	"github.com/wcgcyx/fcr/internal/piecemgr"
)

const (
	// protocols
	queryOfferProtocol = "/fcr/cidnet/queryoffer"
	// time
	repubInterval     = 1 * time.Hour
	pubTTL            = 2 * time.Hour
	ioTimeout         = 1 * time.Minute
	cidOfferTTL       = 10 * time.Minute
	cidOfferTTLRoom   = 1 * time.Minute
	payOfferTTLRoom   = 1 * time.Minute
	payOfferInactRoom = 10 * time.Second
	// Others
	cidOfferExtension  = "cidoffer"
	payOfferExtension  = "payoffer"
	fromExtension      = "from"
	signatureExtension = "signature"
	paymentExtension   = "payment"
	cacheStoreSubPath  = "cache"
)

var log = logging.Logger("cidnetmgr")

// CIDNetworkManagerImplV1 implements CIDNetworkManager.
type CIDNetworkManagerImplV1 struct {
	// Parent ctx
	ctx context.Context
	// Host addr info
	hAddr peer.AddrInfo
	// Host
	h host.Host
	// DHT Network
	dht *dual.DHT
	// Payment network manager
	paynetMgr paynetmgr.PaymentNetworkManager
	// Keystore
	ks datastore.Datastore
	// Piece manager
	ps piecemgr.PieceManager
	// Serving store
	ss cidservstore.CIDServStore
	// Peer manager
	pem peermgr.PeerManager
	// Current in-use served cid.
	active     map[string]int
	activeLock sync.RWMutex
	// Storage for retrieval cache.
	cds    *badgerds.Datastore
	cs     blockstore.Blockstore
	csLock sync.RWMutex
	// Graphsync for answering retrieval queries
	exchangeIn gs.GraphExchange
	// Channels storing incoming retrieval queries
	retrievalInChs     map[string]*channelIn
	retrievalInChsLock sync.RWMutex
	// Graphsync for sending retrieval queries
	h2          host.Host
	exchangeOut gs.GraphExchange
	// Channels stroing outgoing retrieval queries
	retrievalOutID      map[string]string
	retrievalOutChs     map[string]*channelOut
	retrievalOutChsLock sync.RWMutex
	// Offer protocol
	offerProto *OfferProtocol
	// Force publish channel
	fpc chan bool
}

// NewCIDNetworkManagerImplV1 creates a cid network manager.
// It takes a context, a host, a dual dht, a host addr info, a config and a payment network manager as arguments.
// It returns a cid network manager and error.
func NewCIDNetworkManagerImplV1(ctx context.Context, h host.Host, dht *dual.DHT, hAddr peer.AddrInfo, conf config.Config, paynetMgr paynetmgr.PaymentNetworkManager) (CIDNetworkManager, error) {
	// First change the provide record validity
	dhtpvds.ProvideValidity = pubTTL
	// Initialise piece manager
	ps, err := piecemgr.NewPieceManagerImplV1(ctx, conf.RootPath)
	if err != nil {
		return nil, err
	}
	// Initialise serving store
	ss, err := cidservstore.NewCIDServStoreImplV1(ctx, conf.RootPath, paynetMgr.CurrencyIDs())
	if err != nil {
		return nil, err
	}
	// Initialise retrieval caching store
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	cds, err := badgerds.NewDatastore(filepath.Join(conf.RootPath, cacheStoreSubPath), &dsopts)
	if err != nil {
		return nil, err
	}
	// Initialise peer manager
	pem, err := peermgr.NewPeerManagerImplV1(ctx, conf.RootPath, "cid", paynetMgr.CurrencyIDs())
	if err != nil {
		return nil, err
	}
	cs := blockstore.NewBlockstore(cds)
	// Initialise graphsync exchange
	exchangeIn := graphsync.New(ctx, gsnet.NewFromLibp2pHost(h), ps.LinkSystem())
	h2, err := libp2p.New(ctx, libp2p.NoListenAddrs)
	if err != nil {
		return nil, err
	}
	exchangeOut := graphsync.New(ctx, gsnet.NewFromLibp2pHost(h2), storeutil.LinkSystemForBlockstore(cs))
	// Load miner key
	var minerKey []byte
	linkedMiner := ""
	if conf.MinerKey != "" {
		minerKey, err = base64.StdEncoding.DecodeString(conf.MinerKey)
		if err != nil {
			return nil, err
		}
		linkedMiner, err = crypto.GetAddress(minerKey)
		if err != nil {
			return nil, err
		}
	}
	linkedMinerSigs := make(map[uint64][]byte)
	for _, currencyID := range paynetMgr.CurrencyIDs() {
		linkedMinerSigs[currencyID] = nil
		if minerKey != nil {
			addr, err := paynetMgr.GetRootAddress(currencyID)
			if err != nil {
				return nil, err
			}
			sig, err := crypto.Sign(minerKey, []byte(addr))
			if err != nil {
				return nil, err
			}
			linkedMinerSigs[currencyID] = sig
		}
	}
	// Initialise offer protocol
	offerProto := NewOfferProtocol(hAddr, h, ss, ps, pem, paynetMgr.KeyStore(), linkedMiner, linkedMinerSigs)
	mgr := &CIDNetworkManagerImplV1{
		ctx:                 ctx,
		hAddr:               hAddr,
		h:                   h,
		dht:                 dht,
		paynetMgr:           paynetMgr,
		ks:                  paynetMgr.KeyStore(),
		ps:                  ps,
		ss:                  ss,
		pem:                 pem,
		active:              make(map[string]int),
		activeLock:          sync.RWMutex{},
		cds:                 cds,
		cs:                  cs,
		csLock:              sync.RWMutex{},
		exchangeIn:          exchangeIn,
		retrievalInChs:      make(map[string]*channelIn),
		retrievalInChsLock:  sync.RWMutex{},
		h2:                  h2,
		exchangeOut:         exchangeOut,
		retrievalOutID:      make(map[string]string),
		retrievalOutChs:     make(map[string]*channelOut),
		retrievalOutChsLock: sync.RWMutex{},
		offerProto:          offerProto,
		fpc:                 make(chan bool),
	}
	// Initialise gs handlers
	mgr.exchangeIn.RegisterIncomingRequestHook(mgr.onIncomingRequest)
	mgr.exchangeIn.RegisterRequestUpdatedHook(mgr.onUpdatedRequest)
	mgr.exchangeIn.RegisterOutgoingBlockHook(mgr.onOutgoingBlock)
	mgr.exchangeIn.RegisterCompletedResponseListener(mgr.onComplete)
	mgr.exchangeIn.RegisterOutgoingRequestHook(mgr.onOutgoingRequest)
	mgr.exchangeIn.RegisterIncomingBlockHook(mgr.onIncomingBlock)
	mgr.exchangeOut.RegisterOutgoingRequestHook(mgr.onOutgoingRequest)
	mgr.exchangeOut.RegisterIncomingBlockHook(mgr.onIncomingBlock)
	// Start publish routine
	go mgr.publishRoutine()
	go mgr.shutdownRoutine()
	return mgr, nil
}

// Import imports a file.
// It takes a context and a path as arguments.
// It returns the cid imported and error.
func (mgr *CIDNetworkManagerImplV1) Import(ctx context.Context, path string) (cid.Cid, error) {
	return mgr.ps.Import(ctx, path)
}

// ImportCar imports a car file.
// It takes a context and a path as arguments.
// It returns the cid imported and error.
func (mgr *CIDNetworkManagerImplV1) ImportCar(ctx context.Context, path string) (cid.Cid, error) {
	return mgr.ps.ImportCar(ctx, path)
}

// ImportSector imports an filecoin lotus unsealed sector copy.
// It takes a context and a path, a boolean indicating whether to keep a copy as arguments.
// It returns the a list of cids imported and error.
func (mgr *CIDNetworkManagerImplV1) ImportSector(ctx context.Context, path string, copy bool) ([]cid.Cid, error) {
	return mgr.ps.ImportSector(ctx, path, copy)
}

// ListImported lists all imported pieces.
// It takes a context as the argument.
// It returns a list of cids imported and error.
func (mgr *CIDNetworkManagerImplV1) ListImported(ctx context.Context) ([]cid.Cid, error) {
	return mgr.ps.ListImported(ctx)
}

// Inspect inspects a piece.
// It takes a cid as the argument.
// It returns the path, the index, the size, a boolean indicating whether a copy is kept and error.
func (mgr *CIDNetworkManagerImplV1) Inspect(id cid.Cid) (string, int, int64, bool, error) {
	return mgr.ps.Inspect(id)
}

// Remove removes a piece.
// It takes a cid as the argument.
// It returns error.
func (mgr *CIDNetworkManagerImplV1) Remove(id cid.Cid) error {
	// Check if it is in active serving.
	mgr.activeLock.RLock()
	defer mgr.activeLock.RUnlock()
	_, ok := mgr.active[id.String()]
	if ok {
		return fmt.Errorf("cid %v is being retrieved at the moment", id.String())
	}
	return mgr.ps.Remove(id)
}

// QueryOffer queries a given peer for offer.
// It takes a context, a peer id, a currency id, the root id as arguments.
// It returns a cid offer and error.
func (mgr *CIDNetworkManagerImplV1) QueryOffer(ctx context.Context, pid peer.ID, currencyID uint64, id cid.Cid) (*cidoffer.CIDOffer, error) {
	return mgr.offerProto.QueryOffer(ctx, currencyID, id, pid)
}

// SearchOffers searches a list of offers from peers using DHT.
// It takes a context, a currency id, the root id and maximum offer as arguments.
// It returns a list of cid offers.
func (mgr *CIDNetworkManagerImplV1) SearchOffers(ctx context.Context, currencyID uint64, id cid.Cid, max int) []cidoffer.CIDOffer {
	peers := mgr.dht.FindProvidersAsync(ctx, id, max)
	output := make(chan cidoffer.CIDOffer)

	go func() {
		defer close(output)
		for {
			peer, ok := <-peers
			if !ok {
				return
			}
			ctx, cancel := context.WithTimeout(ctx, ioTimeout)
			defer cancel()
			err := mgr.h.Connect(ctx, peer)
			if err != nil {
				log.Warningf("error connecting to peer: %v", err.Error())
				continue
			}
			offer, err := mgr.QueryOffer(ctx, peer.ID, currencyID, id)
			if err != nil {
				log.Warningf("error querying offer: %v", err.Error())
				continue
			}
			_, blocked, _, _, err := mgr.pem.GetPeerInfo(offer.CurrencyID(), offer.ToAddr())
			if err == nil && blocked {
				// Filter out offers from blocked peers.
				continue
			}
			select {
			case <-ctx.Done():
				return
			case output <- *offer:
			}
		}
	}()
	res := make([]cidoffer.CIDOffer, 0)
	for offer := range output {
		res = append(res, offer)
	}
	return res
}

// RetrieveFromCache retrieves a content based on given cid from cache.
// It takes a context, a cid and outpath as arguments.
// It returns boolean indicating if found in cache and error.
func (mgr *CIDNetworkManagerImplV1) RetrieveFromCache(ctx context.Context, id cid.Cid, outPath string) (bool, error) {
	// Check if we have this retrieved.
	mgr.csLock.RLock()
	defer mgr.csLock.RUnlock()
	exists, err := mgr.cs.Has(id)
	if err != nil {
		return false, fmt.Errorf("error checking retrieval cache: %v", err.Error())
	}
	if exists {
		// Exists
		log.Infof("cid %v exists in retrieval cache, save directly", id.String())
		dag := merkledag.NewDAGService(blockservice.New(mgr.cs, nil))
		nd, err := dag.Get(ctx, id)
		if err != nil {
			return true, err
		}
		file, err := unixfile.NewUnixfsFile(ctx, dag, nd)
		if err != nil {
			return true, err
		}
		fi, err := os.Stat(outPath)
		if err != nil {
			if os.IsNotExist(err) {
				return true, files.WriteTo(file, outPath)
			}
			return true, err
		}
		if fi.Mode().IsDir() {
			outPath = filepath.Join(outPath, id.String())
		}
		return true, files.WriteTo(file, outPath)
	}
	return false, nil
}

// Retrieve retrieves a content based on an offer to a given path.
// It takes a context, a cid offer, a pay offer (can be nil if retrieve directly from connected peers), an outpath as arguments.
// It returns error.
func (mgr *CIDNetworkManagerImplV1) Retrieve(ctx context.Context, cidOffer cidoffer.CIDOffer, payOffer *payoffer.PayOffer, outPath string) error {
	// Verify offer expiration time
	if time.Now().Unix() > cidOffer.Expiration()-int64(cidOfferTTLRoom.Seconds()) {
		return fmt.Errorf("invalid cid offer received: offer expired at %v, now is %v", cidOffer.Expiration(), time.Now())
	}
	if payOffer != nil {
		if time.Now().Unix() > payOffer.Expiration()-int64(payOfferTTLRoom.Seconds()) {
			return fmt.Errorf("invalid pay offer received: offer expired at %v, now is %v", payOffer.Expiration(), time.Now())
		}
		if time.Now().Unix() > payOffer.CreatedAt()+int64(payOffer.MaxInactivity().Seconds())-int64(payOfferInactRoom.Seconds()) {
			return fmt.Errorf("invalid pay offer received: offer created at %v, max inactivity of %v, now is %v", payOffer.CreatedAt(), payOffer.MaxInactivity(), time.Now())
		}
		if err := mgr.paynetMgr.ReserveWithPayOffer(payOffer); err != nil {
			return fmt.Errorf("error reserving funds with offer: %v", err.Error())
		}
	} else {
		if err := mgr.paynetMgr.Reserve(cidOffer.CurrencyID(), cidOffer.ToAddr(), big.NewInt(0).Mul(big.NewInt(cidOffer.Size()), cidOffer.PPB())); err != nil {
			return fmt.Errorf("error reserving funds: %v", err.Error())
		}
	}
	// Check any current retrieval process
	mgr.retrievalOutChsLock.RLock()
	_, ok := mgr.retrievalOutID[cidOffer.Root().String()]
	mgr.retrievalOutChsLock.RUnlock()
	if ok {
		return fmt.Errorf("error in retrieving offer: already in the middle of retrieval")
	}
	err := mgr.h2.Connect(ctx, cidOffer.PeerAddr())
	if err != nil {
		return err
	}
	// Counter sign the offer
	// Get key
	prv, err := mgr.ks.Get(datastore.NewKey(fmt.Sprintf("%v", cidOffer.CurrencyID())))
	if err != nil {
		return fmt.Errorf("error getting key for currency id %v: %v", cidOffer.CurrencyID(), err.Error())
	}
	fromAddr, err := crypto.GetAddress(prv)
	if err != nil {
		return fmt.Errorf("error getting address for currency id %v: %v", cidOffer.CurrencyID(), err.Error())
	}
	sig, err := crypto.Sign(prv, cidOffer.ToBytes())
	if err != nil {
		return fmt.Errorf("error counter signing the offer: %v", err.Error())
	}
	// Add the retrieval peer.
	err = mgr.pem.AddPeer(cidOffer.CurrencyID(), cidOffer.ToAddr(), cidOffer.PeerAddr().ID)
	if err != nil {
		log.Warnf("error in adding the retrieval peer: %v", err.Error())
	}
	ssb := builder.NewSelectorSpecBuilder(basicnode.Prototype.Any)
	allSelector := ssb.ExploreRecursive(selector.RecursionLimitNone(),
		ssb.ExploreAll(ssb.ExploreRecursiveEdge())).Node()
	var respCh <-chan gs.ResponseProgress
	var errCh <-chan error
	if payOffer != nil {
		respCh, errCh = mgr.exchangeOut.Request(ctx, cidOffer.PeerAddr().ID, cidlink.Link{Cid: cidOffer.Root()}, allSelector,
			gs.ExtensionData{Name: cidOfferExtension, Data: cidOffer.ToBytes()},
			gs.ExtensionData{Name: payOfferExtension, Data: payOffer.ToBytes()},
			gs.ExtensionData{Name: fromExtension, Data: []byte(fromAddr)},
			gs.ExtensionData{Name: signatureExtension, Data: sig})
	} else {
		respCh, errCh = mgr.exchangeOut.Request(ctx, cidOffer.PeerAddr().ID, cidlink.Link{Cid: cidOffer.Root()}, allSelector,
			gs.ExtensionData{Name: cidOfferExtension, Data: cidOffer.ToBytes()},
			gs.ExtensionData{Name: fromExtension, Data: []byte(fromAddr)},
			gs.ExtensionData{Name: signatureExtension, Data: sig})
	}
	defer func() {
		mgr.retrievalOutChsLock.Lock()
		defer mgr.retrievalOutChsLock.Unlock()
		id := mgr.retrievalOutID[cidOffer.Root().String()]
		delete(mgr.retrievalOutID, cidOffer.Root().String())
		delete(mgr.retrievalOutChs, id)
	}()
	go func() {
		for resp := range respCh {
			log.Infof("Pulled block: %v", resp.LastBlock)
		}
	}()
	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = ctx.Err()
	}
	if err != nil {
		mgr.pem.Record(cidOffer.CurrencyID(), cidOffer.ToAddr(), false)
		return err
	}
	mgr.pem.Record(cidOffer.CurrencyID(), cidOffer.ToAddr(), true)
	// Save to outPath
	mgr.csLock.RLock()
	defer mgr.csLock.RUnlock()
	dag := merkledag.NewDAGService(blockservice.New(mgr.cs, nil))
	nd, err := dag.Get(ctx, cidOffer.Root())
	if err != nil {
		return err
	}
	file, err := unixfile.NewUnixfsFile(ctx, dag, nd)
	if err != nil {
		return err
	}
	fi, err := os.Stat(outPath)
	if err != nil {
		if os.IsNotExist(err) {
			return files.WriteTo(file, outPath)
		}
		return err
	}
	if fi.Mode().IsDir() {
		outPath = filepath.Join(outPath, cidOffer.Root().String())
	}
	return files.WriteTo(file, outPath)
}

// GetRetrievalCacheSize gets the current cache DB size in bytes.
// It returns the size and error.
func (mgr *CIDNetworkManagerImplV1) GetRetrievalCacheSize() (uint64, error) {
	mgr.csLock.RLock()
	defer mgr.csLock.RUnlock()
	ids, err := mgr.cs.AllKeysChan(mgr.ctx)
	if err != nil {
		return 0, err
	}
	size := uint64(0)
	for id := range ids {
		bs, err := mgr.cs.GetSize(id)
		if err != nil {
			return 0, err
		}
		size += uint64(bs)
	}
	return size, nil
}

// CleanRetrievalCache will clean the cache DB.
// It returns the error.
func (mgr *CIDNetworkManagerImplV1) CleanRetrievalCache() error {
	mgr.csLock.Lock()
	defer mgr.csLock.Unlock()
	ids, err := mgr.cs.AllKeysChan(context.Background())
	if err != nil {
		return err
	}
	for id := range ids {
		err = mgr.cs.DeleteBlock(id)
		if err != nil {
			return err
		}
	}
	return nil
}

// StartServing starts serving a cid for others to use.
// It takes a context, a currency id, the cid, ppn (price per byte) as arguments.
// It returns error.
func (mgr *CIDNetworkManagerImplV1) StartServing(ctx context.Context, currencyID uint64, id cid.Cid, ppb *big.Int) error {
	_, _, _, _, err := mgr.ps.Inspect(id)
	if err != nil {
		return err
	}
	err = mgr.dht.Provide(ctx, id, true)
	if err != nil {
		return err
	}
	return mgr.ss.Serve(currencyID, id, ppb)
}

// ListServings lists all servings.
// It takes a context as the argument.
// It returns a map from currency id to list of active servings and error.
func (mgr *CIDNetworkManagerImplV1) ListServings(ctx context.Context) (map[uint64][]cid.Cid, error) {
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
// It takes a currency id, the cid as arguments.
// It returns the ppb (price per byte), expiration and error.
func (mgr *CIDNetworkManagerImplV1) InspectServing(currencyID uint64, id cid.Cid) (*big.Int, int64, error) {
	return mgr.ss.Inspect(currencyID, id)
}

// StopServing stops a serving for cid, it will put a cid into retirement before it automatically expires.
// It takes a currency id, the cid as arguments.
// It returns the error.
func (mgr *CIDNetworkManagerImplV1) StopServing(currencyID uint64, id cid.Cid) error {
	return mgr.ss.Retire(currencyID, id, cidOfferTTL)
}

// ForcePublishServings will force the manager to do a publication of current servings immediately.
func (mgr *CIDNetworkManagerImplV1) ForcePublishServings() {
	go func() {
		mgr.fpc <- true
	}()
}

// ListInUseServings will list all the in use servings.
func (mgr *CIDNetworkManagerImplV1) ListInUseServings(ctx context.Context) ([]cid.Cid, error) {
	mgr.activeLock.RLock()
	defer mgr.activeLock.RUnlock()
	res := make([]cid.Cid, 0)
	for cidStr := range mgr.active {
		id, err := cid.Parse(cidStr)
		if err != nil {
			return nil, err
		}
		res = append(res, id)
	}
	return res, nil
}

// ListPeers is used to list all the peers.
// It takes a context as the argument.
// It returns a map from currency id to list of peers and error.
func (mgr *CIDNetworkManagerImplV1) ListPeers(ctx context.Context) (map[uint64][]string, error) {
	return mgr.pem.ListPeers(ctx)
}

// GetPeerInfo gets the information of a peer.
// It takes a currency id, peer address as arguments.
// It returns the peer id, a boolean indicate whether it is blocked, success count, failure count and error.
func (mgr *CIDNetworkManagerImplV1) GetPeerInfo(currencyID uint64, toAddr string) (peer.ID, bool, int, int, error) {
	return mgr.pem.GetPeerInfo(currencyID, toAddr)
}

// BlockPeer is used to block a peer.
// It takes a currency id, peer address to block as arguments.
// It returns error.
func (mgr *CIDNetworkManagerImplV1) BlockPeer(currencyID uint64, toAddr string) error {
	return mgr.pem.BlockPeer(currencyID, toAddr)
}

// UnblockPeer is used to unblock a peer.
// It takes a currency id, peer address to unblock as arguments.
// It returns error.
func (mgr *CIDNetworkManagerImplV1) UnblockPeer(currencyID uint64, toAddr string) error {
	return mgr.pem.UnblockPeer(currencyID, toAddr)
}

// publishRoutine is a routine that publish servings at configured frequency.
func (mgr *CIDNetworkManagerImplV1) publishRoutine() {
	for {
		ctx := context.Background()
		idsMap, err := mgr.ss.ListActive(ctx)
		if err != nil {
			log.Warnf("republish error in listing active servings: %s", err)
		} else {
			toPublish := make(map[string]cid.Cid)
			for _, ids := range idsMap {
				for _, id := range ids {
					toPublish[id.String()] = id
				}
			}
			for _, id := range toPublish {
				err = mgr.dht.Provide(ctx, id, true)
				if err != nil {
					log.Warnf("republish error in providing content to network: %s", err)
					continue
				}
			}
		}
		tc := time.After(repubInterval)
		select {
		case <-mgr.ctx.Done():
			return
		case <-tc:
		case <-mgr.fpc:
		}
	}
}

// shutdownRoutine is used to safely close the routine.
func (mgr *CIDNetworkManagerImplV1) shutdownRoutine() {
	<-mgr.ctx.Done()
	err := mgr.cds.Close()
	if err != nil {
		log.Errorf("error closing cache database: %s", err)
	}
}
