package retmgr

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
	golock "github.com/viney-shih/go-lock"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/paynet/protos/payproto"
	"github.com/wcgcyx/fcr/peermgr"
	"github.com/wcgcyx/fcr/peernet/protos/addrproto"
	"github.com/wcgcyx/fcr/piecemgr"
)

// Logger
var log = logging.Logger("retrievalmgr")

const (
	offerExtension        = "offer"
	keyTypeExtension      = "keyType"
	fromExtension         = "from"
	signatureExtension    = "signature"
	paymentExtension      = "payment"
	paymentIndexExtension = "index"
)

// RetrievalManager manages data retrieval.
type RetrievalManager struct {
	// Host (main host)
	h host.Host

	// Graphsync for answering retrieval queries
	exchangeIn gs.GraphExchange

	// Host2 (for sending retrieval queries)
	h2 host.Host

	// Graphsync for sending retrieval queries
	exchangeOut gs.GraphExchange

	// Addr proto
	addrProto *addrproto.AddrProtocol

	// Pay proto
	payProto *payproto.PayProtocol

	// Signer
	signer crypto.Signer

	// Peer manager
	peerMgr peermgr.PeerManager

	// Piece manager
	pieceMgr piecemgr.PieceManager

	// Storage for retrieval cache.
	cds    datastore.Datastore
	cs     blockstore.Blockstore
	csLock golock.RWMutex

	// Currently active incoming retrievals.
	activeInOfferID map[string]*incomingRetrievalState
	activeInLock    golock.RWMutex

	// Currently active outgoing retrievals.
	activeOutOfferID map[string]*outgoingRetrievalState
	activeOutReqID   map[string]*outgoingRetrievalState
	activeOutLock    golock.RWMutex

	// Process related
	routineCtx context.Context

	// Shutdown function.
	shutdown func()

	// Options
	tempPath  string
	ioTimeout time.Duration
	opTimeout time.Duration
}

// NewRetrievalManager creates a new retrieval manager.
//
// @input - context, host, addr protocol, pay protocol, signer, peer manager, piece manager, options.
//
// @output - retrieval manager, error.
func NewRetrievalManager(
	ctx context.Context,
	h host.Host,
	addrProto *addrproto.AddrProtocol,
	payProto *payproto.PayProtocol,
	signer crypto.Signer,
	peerMgr peermgr.PeerManager,
	pieceMgr piecemgr.PieceManager,
	opts Opts,
) (*RetrievalManager, error) {
	// Parse options
	if opts.TempPath == "" || opts.CachePath == "" {
		return nil, fmt.Errorf("empty path provided")
	}
	ioTimeout := opts.IOTimeout
	if ioTimeout == 0 {
		ioTimeout = defaultIOTimeout
	}
	opTimeout := opts.OpTimeout
	if opTimeout == 0 {
		opTimeout = defaultOpTimeout
	}
	log.Infof("Start retrieval manager...")
	// Open retrieval cache store
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	cds, err := badgerds.NewDatastore(opts.CachePath, &dsopts)
	if err != nil {
		return nil, err
	}
	cs := blockstore.NewBlockstore(cds)
	defer func() {
		if err != nil {
			log.Infof("Fail to start retrieval manager, close ds...")
			err0 := cds.Close()
			if err0 != nil {
				log.Errorf("Fail to close ds after failing to start retrieval manager: %v", err0.Error())
			}
		}
	}()
	// Create manager before initialising exchanges.
	mgr := &RetrievalManager{
		h:                h,
		addrProto:        addrProto,
		payProto:         payProto,
		signer:           signer,
		peerMgr:          peerMgr,
		pieceMgr:         pieceMgr,
		cds:              cds,
		cs:               cs,
		csLock:           golock.NewCASMutex(),
		activeInOfferID:  make(map[string]*incomingRetrievalState),
		activeInLock:     golock.NewCASMutex(),
		activeOutOfferID: make(map[string]*outgoingRetrievalState),
		activeOutReqID:   make(map[string]*outgoingRetrievalState),
		activeOutLock:    golock.NewCASMutex(),
		tempPath:         opts.TempPath,
		ioTimeout:        ioTimeout,
		opTimeout:        opTimeout,
	}
	// Initialise exchange out.
	h2, err := libp2p.New(libp2p.NoListenAddrs)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			log.Infof("Fail to start retrieval manager, close exchange out...")
			err0 := h2.Close()
			if err0 != nil {
				log.Errorf("Fail to close exchange out host after failing to start retrieval manager: %v", err0.Error())
			}
		}
	}()
	mgr.h2 = h2
	// Initialise exchange In.
	err = mgr.loadStates(ctx)
	if err != nil {
		log.Errorf("Fail to load retrieval states: %v", err.Error())
		return nil, err
	}
	// Create routine
	routineCtx, cancel := context.WithCancel(context.Background())
	mgr.routineCtx = routineCtx
	exchangeOut := graphsync.New(routineCtx, gsnet.NewFromLibp2pHost(h2), storeutil.LinkSystemForBlockstore(cs))
	exchangeOut.RegisterOutgoingRequestHook(mgr.onOutgoingRequest)
	exchangeOut.RegisterIncomingBlockHook(mgr.onIncomingBlock)
	mgr.exchangeOut = exchangeOut
	exchangeIn := graphsync.New(routineCtx, gsnet.NewFromLibp2pHost(h), mgr.pieceMgr.LinkSystem())
	exchangeIn.RegisterIncomingRequestHook(mgr.onIncomingRequest)
	exchangeIn.RegisterOutgoingBlockHook(mgr.onOutgoingBlock)
	exchangeIn.RegisterRequestUpdatedHook(mgr.onUpdatedRequest)
	exchangeIn.RegisterCompletedResponseListener(mgr.onComplete)
	exchangeIn.RegisterRequestorCancelledListener(mgr.OnRequestorCancelledListener)
	exchangeIn.RegisterNetworkErrorListener(mgr.OnNetworkErrorListener)
	mgr.exchangeIn = exchangeIn
	mgr.shutdown = func() {
		log.Infof("Stop all routines...")
		cancel()
		log.Infof("Close exchange out...")
		err = h2.Close()
		if err != nil {
			log.Errorf("Fail to close exchange out host: %v", err.Error())
		}
		log.Infof("Clean and save retrieval states...")
		if !mgr.activeOutLock.TryLockWithContext(context.Background()) {
			log.Errorf("Fail to lock active out")
		}
		mgr.cleanActiveIn()
		err = mgr.saveStates(context.Background())
		if err != nil {
			log.Errorf("Fail to save retrieval states: %v", err.Error())
		}
		log.Infof("Stop datastore...")
		err = cds.Close()
		if err != nil {
			log.Errorf("Fail to stop datastore: %v", err.Error())
		}
	}
	return mgr, nil
}

// Shutdown safely shuts down the component.
func (mgr *RetrievalManager) Shutdown() {
	log.Infof("Start shutdown...")
	mgr.shutdown()
}

// Retrieve is used to retrieve a piece based on a given piece offer and a given reserved channel to given path.
//
// @input - context, piece offer, optional pay offer, reserved channel id, reservation id, output path.
//
// @output - response progress chan out, error chan out.
func (mgr *RetrievalManager) Retrieve(ctx context.Context, pieceOffer fcroffer.PieceOffer, payOffer *fcroffer.PayOffer, resCh string, resID uint64, outPath string) (<-chan gs.ResponseProgress, <-chan error) {
	out := make(chan gs.ResponseProgress, 128)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Start retrieval of %v using %v with res ch to be %v and res id to be %v to path %v", pieceOffer, payOffer, resCh, resID, outPath)
		defer func() {
			close(out)
			close(errChan)
		}()

		// Check if peer is blocked.
		exists, err := mgr.peerMgr.HasPeer(ctx, pieceOffer.CurrencyID, pieceOffer.RecipientAddr)
		if err != nil {
			log.Warnf("Fail to check if contains peer %v-%v: %v", pieceOffer.CurrencyID, pieceOffer.RecipientAddr, err.Error())
			errChan <- err
			return
		}
		if exists {
			blocked, err := mgr.peerMgr.IsBlocked(ctx, pieceOffer.CurrencyID, pieceOffer.RecipientAddr)
			if err != nil {
				log.Warnf("Fail to check if peer %v-%v is blocked: %v", pieceOffer.CurrencyID, pieceOffer.RecipientAddr, err.Error())
				errChan <- err
				return
			}
			if blocked {
				errChan <- fmt.Errorf("peer %v-%v has been blocked", pieceOffer.CurrencyID, pieceOffer.RecipientAddr)
				return
			}
		}
		// Counter sign the offer
		offerData, err := pieceOffer.Encode()
		if err != nil {
			log.Errorf("Fail to encode offer: %v", err.Error())
			errChan <- err
			return
		}
		keyType, fromAddr, err := mgr.signer.GetAddr(ctx, pieceOffer.CurrencyID)
		if err != nil {
			log.Warnf("Fail to get addr for %v: %v", pieceOffer.CurrencyID, err.Error())
			errChan <- err
			return
		}
		_, sig, err := mgr.signer.Sign(ctx, pieceOffer.CurrencyID, offerData)
		if err != nil {
			log.Warnf("Fail to sign for %v: %v", pieceOffer.CurrencyID, err.Error())
			errChan <- err
			return
		}
		// Get Offer ID.
		offerID, err := getOfferID(pieceOffer)
		if err != nil {
			log.Errorf("Fail to get offer ID: %v", err.Error())
			errChan <- err
			return
		}
		// Check if there is an existing retrieval process for the same offer.
		if !mgr.activeOutLock.RTryLockWithContext(ctx) {
			log.Debugf("Fail to obtain read lock for active out states")
			errChan <- fmt.Errorf("fail to RLock active out")
			return
		}
		release := mgr.activeOutLock.RUnlock

		// Make sure the context will be cancelled after retrieval is finished.
		ctxc, cancel := context.WithCancel(ctx)
		defer cancel()
		state, ok := mgr.activeOutOfferID[offerID]
		if ok {
			// There is an existing retrieval process for the same offer.
			if err = func() error {
				defer release()
				// Check if existing retrieval process has finished.
				if !state.lock.RTryLockWithContext(ctx) {
					log.Debugf("Fail to obtain read lock for offer ID %v: %v", offerID, err.Error())
					return fmt.Errorf("fail to RLock state")
				}
				release = state.lock.RUnlock

				if state.ctx.Err() == nil {
					// State has not yet finished.
					defer release()
					return fmt.Errorf("offer is in the middle of an active retrieval process")
				}
				// State has finished. Switch to lock.
				release()

				if !state.lock.TryLockWithContext(ctx) {
					log.Debugf("Fail to obtain write lock for offer ID %v: %v", offerID, err.Error())
					return fmt.Errorf("fail to Lock state")
				}
				release = state.lock.Unlock
				defer release()

				// Check again.
				if state.ctx.Err() == nil {
					// State has just started.
					log.Debugf("Offer just being used during switching lock")
					return fmt.Errorf("offer just being used during switching lock")
				}
				// Check if state has expired
				if state.expiredAt.Before(time.Now()) {
					// State has expired.
					log.Debugf("State has expired, cannot resume")
					return fmt.Errorf("state has expired, cannot resume")
				}
				// Process can be resumed. Continue the process.
				state.ctx = ctxc
				state.payOffer = payOffer
				state.resCh = resCh
				state.resID = resID
				return nil
			}(); err != nil {
				errChan <- err
				return
			}
		} else {
			// There is no existing retrieval process for the same offer, switch to lock access.
			if err = func() error {
				release()
				if !mgr.activeOutLock.TryLockWithContext(ctx) {
					log.Debugf("Fail to obtain write lock for active out states")
					return fmt.Errorf("fail to Lock active out")
				}
				release = mgr.activeOutLock.Unlock
				defer release()

				mgr.cleanActiveOut()
				// Check again
				state, ok = mgr.activeOutOfferID[offerID]
				if ok {
					log.Debugf("Offer is in the middle of an active retrieval process during switching lock")
					return fmt.Errorf("offer is in the middle of an active retrieval process during switching lock")
				}
				// Start a new process.
				// Check if offer has expired.
				if pieceOffer.Expiration.Before(time.Now()) {
					log.Debugf("Offer has expired at %v, now %v", pieceOffer.Expiration, time.Now())
					return fmt.Errorf("given offer has expired at %v, now %v", pieceOffer.Expiration, time.Now())
				}
				mgr.activeOutOfferID[offerID] = &outgoingRetrievalState{
					currencyID: pieceOffer.CurrencyID,
					toAddr:     pieceOffer.RecipientAddr,
					inactivity: pieceOffer.Inactivity,
					ppb:        pieceOffer.PPB,
					expiredAt:  pieceOffer.Expiration,
					lock:       golock.NewCASMutex(),
					ctx:        ctxc,
					payOffer:   payOffer,
					resCh:      resCh,
					resID:      resID,
				}
				return nil
			}(); err != nil {
				errChan <- err
				return
			}
		}
		// Start the retrieval process.
		// Connect to peer.
		pid, err := mgr.addrProto.ConnectToPeer(ctx, pieceOffer.CurrencyID, pieceOffer.RecipientAddr)
		if err != nil {
			log.Debugf("Fail to connect to peer %v-%v: %v", pieceOffer.CurrencyID, pieceOffer.RecipientAddr, err.Error())
			errChan <- err
			return
		}
		// Get peer addr
		pi, err := mgr.peerMgr.PeerAddr(ctx, pieceOffer.CurrencyID, pieceOffer.RecipientAddr)
		if err != nil {
			log.Warnf("Fail to obtain peer network address for %v-%v: %v", pieceOffer.CurrencyID, pieceOffer.RecipientAddr, err.Error())
			errChan <- err
			return
		}
		if !mgr.csLock.RTryLockWithContext(ctx) {
			log.Debugf("Fail to obtain read lock for cache store")
			errChan <- fmt.Errorf("fail to RLock cache store")
			return
		}
		release = mgr.csLock.RUnlock
		defer release()
		subCtx, cancel := context.WithTimeout(ctx, mgr.ioTimeout)
		defer cancel()
		err = mgr.h2.Connect(subCtx, pi)
		if err != nil {
			log.Warnf("Fail to connect to %v: %v", pi, err.Error())
			errChan <- err
			return
		}
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer starts to handle retriaval of %v", pieceOffer.ID),
				CreatedAt:   time.Now(),
			}
			err := mgr.peerMgr.AddToHistory(mgr.routineCtx, pieceOffer.CurrencyID, pieceOffer.RecipientAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, pieceOffer.CurrencyID, pieceOffer.RecipientAddr, err.Error())
			}
		}()
		ssb := builder.NewSelectorSpecBuilder(basicnode.Prototype.Any)
		allSelector := ssb.ExploreRecursive(selector.RecursionLimitNone(),
			ssb.ExploreAll(ssb.ExploreRecursiveEdge())).Node()
		respCh, errCh := mgr.exchangeOut.Request(ctx, pid, cidlink.Link{Cid: pieceOffer.ID}, allSelector,
			gs.ExtensionData{Name: offerExtension, Data: offerData},
			gs.ExtensionData{Name: keyTypeExtension, Data: []byte{keyType}},
			gs.ExtensionData{Name: fromExtension, Data: []byte(fromAddr)},
			gs.ExtensionData{Name: signatureExtension, Data: sig},
		)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for resp := range respCh {
				out <- resp
			}
		}()
		select {
		case err = <-errCh:
		case <-ctx.Done():
			err = ctx.Err()
		}
		wg.Wait()
		if err != nil {
			go func() {
				rec := peermgr.Record{
					Description: fmt.Sprintf("peer fails to handle retriaval of %v: %v", pieceOffer.ID, err.Error()),
					CreatedAt:   time.Now(),
				}
				err := mgr.peerMgr.AddToHistory(mgr.routineCtx, pieceOffer.CurrencyID, pieceOffer.RecipientAddr, rec)
				if err != nil {
					log.Warnf("Fail add %v to history of %v-%v: %v", rec, pieceOffer.CurrencyID, pieceOffer.RecipientAddr, err.Error())
				}
			}()
			errChan <- err
			return
		}
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer succeed to handle retriaval of %v", pieceOffer.ID),
				CreatedAt:   time.Now(),
			}
			err := mgr.peerMgr.AddToHistory(mgr.routineCtx, pieceOffer.CurrencyID, pieceOffer.RecipientAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, pieceOffer.CurrencyID, pieceOffer.RecipientAddr, err.Error())
			}
		}()
		// Save to out path
		dag := merkledag.NewDAGService(blockservice.New(mgr.cs, nil))
		nd, err := dag.Get(ctx, pieceOffer.ID)
		if err != nil {
			log.Errorf("Fail to get node %v: %v", pieceOffer.ID, err.Error())
			errChan <- err
			return
		}
		file, err := unixfile.NewUnixfsFile(ctx, dag, nd)
		if err != nil {
			log.Errorf("Fail to create file: %v", err.Error())
			errChan <- err
			return
		}
		fi, err := os.Stat(outPath)
		if err != nil {
			if os.IsNotExist(err) {
				err = files.WriteTo(file, outPath)
			}
			errChan <- err
			return
		}
		if fi.Mode().IsDir() {
			outPath = filepath.Join(outPath, pieceOffer.ID.String())
		}
		errChan <- files.WriteTo(file, outPath)
		return
	}()
	return out, errChan
}

// RetrieveFromCache is used to retrieve a piece from given root id to given path.
//
// @input - context, root id, output path.
//
// @output - if found in cache, error.
func (mgr *RetrievalManager) RetrieveFromCache(ctx context.Context, root cid.Cid, outPath string) (bool, error) {
	log.Debugf("Retrieval %v from cache to %v", root, outPath)
	if !mgr.csLock.RTryLockWithContext(ctx) {
		log.Debugf("Fail to obtain read lock for cache store")
		return false, fmt.Errorf("fail to RLock cache store")
	}
	release := mgr.csLock.RUnlock
	defer release()

	exists, err := mgr.cs.Has(ctx, root)
	if err != nil {
		log.Warnf("Fail to check if contains %v: %v", root, err.Error())
		return false, fmt.Errorf("error checking retrieval cache: %v", err.Error())
	}
	if exists {
		dag := merkledag.NewDAGService(blockservice.New(mgr.cs, nil))
		nd, err := dag.Get(ctx, root)
		if err != nil {
			log.Errorf("Fail to get node %v: %v", root, err.Error())
			return true, err
		}
		file, err := unixfile.NewUnixfsFile(ctx, dag, nd)
		if err != nil {
			log.Errorf("Fail to create file: %v", err.Error())
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
			outPath = filepath.Join(outPath, root.String())
		}
		return true, files.WriteTo(file, outPath)
	}
	return false, nil
}

// GetRetrievalCacheSize gets the current cache DB size in bytes.
//
// @output - size, error.
func (mgr *RetrievalManager) GetRetrievalCacheSize(ctx context.Context) (uint64, error) {
	log.Debugf("Get retrieval cache")
	if !mgr.csLock.RTryLockWithContext(ctx) {
		log.Debugf("Fail to obtain read lock for cache store")
		return 0, fmt.Errorf("fail to RLock cache store")
	}
	release := mgr.csLock.RUnlock
	defer release()

	ids, err := mgr.cs.AllKeysChan(ctx)
	if err != nil {
		log.Warnf("Fail to list all keys: %v", err.Error())
		return 0, err
	}
	size := uint64(0)
	for id := range ids {
		bs, err := mgr.cs.GetSize(ctx, id)
		if err != nil {
			log.Warnf("Fail to get size for %v: %v", id, err.Error())
			return 0, err
		}
		size += uint64(bs)
	}
	return size, nil
}

// CleanRetrievalCache will clean the cache DB.
//
// @output - error.
func (mgr *RetrievalManager) CleanRetrievalCache(ctx context.Context) error {
	log.Debugf("Clean retrieval cache")
	if !mgr.csLock.TryLockWithContext(ctx) {
		log.Warnf("Fail to obtain write lock for cache store")
		return fmt.Errorf("fail to Lock cache store")
	}
	release := mgr.csLock.Unlock
	defer release()

	ids, err := mgr.cs.AllKeysChan(ctx)
	if err != nil {
		log.Warnf("Fail to list all keys: %v", err.Error())
		return err
	}
	total := make([]cid.Cid, 0)
	for id := range ids {
		total = append(total, id)
	}
	for _, id := range total {
		err = mgr.cs.DeleteBlock(ctx, id)
		if err != nil {
			log.Warnf("Fail to delete block %v: %v", id, err.Error())
			return err
		}
	}
	return nil
}

// CleanIncomingProcesses will clean the incoming processes.
//
// @input - context.
//
// @output - error.
func (mgr *RetrievalManager) CleanIncomingProcesses(ctx context.Context) error {
	log.Debugf("Clean incoming processes")
	if !mgr.activeInLock.TryLockWithContext(ctx) {
		log.Warnf("Fail to obtain write lock for active in states")
		return fmt.Errorf("fail to lock active in")
	}
	release := mgr.activeInLock.Unlock
	defer release()

	mgr.cleanActiveIn()
	return nil
}

// CleanOutgoingProcesses will clean the outgoing processes.
//
// @input - context.
//
// @output - error.
func (mgr *RetrievalManager) CleanOutgoingProcesses(ctx context.Context) error {
	log.Debugf("Clean outgoing processes")
	if !mgr.activeOutLock.TryLockWithContext(ctx) {
		log.Warnf("Fail to obtain write lock for active out states")
		return fmt.Errorf("fail to lock active out")
	}
	release := mgr.activeOutLock.Unlock
	defer release()

	mgr.cleanActiveOut()
	return nil
}
