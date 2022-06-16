package node

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
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	crypto2 "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	noise "github.com/libp2p/go-libp2p-noise"
	routing "github.com/libp2p/go-libp2p-routing"
	libp2ptls "github.com/libp2p/go-libp2p-tls"
	cofferproto "github.com/wcgcyx/fcr/cidnet/protos/offerproto"
	"github.com/wcgcyx/fcr/config"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/cservmgr"
	"github.com/wcgcyx/fcr/mpstore"
	"github.com/wcgcyx/fcr/offermgr"
	"github.com/wcgcyx/fcr/paychmon"
	"github.com/wcgcyx/fcr/paychstore"
	"github.com/wcgcyx/fcr/paymgr"
	pofferproto "github.com/wcgcyx/fcr/paynet/protos/offerproto"
	"github.com/wcgcyx/fcr/paynet/protos/paychproto"
	"github.com/wcgcyx/fcr/paynet/protos/payproto"
	"github.com/wcgcyx/fcr/paynet/protos/routeproto"
	"github.com/wcgcyx/fcr/peermgr"
	"github.com/wcgcyx/fcr/peernet/protos/addrproto"
	"github.com/wcgcyx/fcr/piecemgr"
	"github.com/wcgcyx/fcr/pservmgr"
	"github.com/wcgcyx/fcr/renewmgr"
	"github.com/wcgcyx/fcr/reservmgr"
	"github.com/wcgcyx/fcr/retmgr"
	"github.com/wcgcyx/fcr/routestore"
	"github.com/wcgcyx/fcr/settlemgr"
	"github.com/wcgcyx/fcr/substore"
	"github.com/wcgcyx/fcr/trans"
	"github.com/wcgcyx/fcr/version"
)

const keyFile = "p2p.key"

// Node contains all components of the system.
type Node struct {
	// Components
	Host             host.Host
	Signer           crypto.Signer
	PeerMgr          peermgr.PeerManager
	Transactor       trans.Transactor
	ActiveOutPaych   paychstore.PaychStore
	InactiveOutPaych paychstore.PaychStore
	ActiveInPaych    paychstore.PaychStore
	InactiveInPaych  paychstore.PaychStore
	PServMgr         pservmgr.PaychServingManager
	ReservMgr        reservmgr.ReservationManager
	RouteStore       routestore.RouteStore
	SubStore         substore.SubStore
	PayMgr           paymgr.PaymentManager
	SettleMgr        settlemgr.SettlementManager
	RenewMgr         renewmgr.RenewManager
	OfferMgr         offermgr.OfferManager
	PaychMonitor     paychmon.PaychMonitor
	PieceMgr         piecemgr.PieceManager
	CServMgr         cservmgr.PieceServingManager
	MinerProofStore  mpstore.MinerProofStore
	// Protocols
	AddrProto   *addrproto.AddrProtocol
	PaychProto  *paychproto.PaychProtocol
	PofferProto *pofferproto.OfferProtocol
	RouteProto  *routeproto.RouteProtocol
	CofferProto *cofferproto.OfferProtocol
	PayProto    *payproto.PayProtocol
	// Retrieval manager
	RetMgr *retmgr.RetrievalManager
	// Shutdown
	shutdown func()
}

// NewNode creates a new node.
//
// @input - context, config.
//
// @output - node, error.
func NewNode(ctx context.Context, conf config.Config) (*Node, error) {
	// Configure loggings
	var err error
	if conf.APIServerLoggingLevel != "" {
		if err = logging.SetLogLevel("apiserver", conf.APIServerLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.SignerLoggingLevel != "" {
		if err = logging.SetLogLevel("signer", conf.SignerLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.PeerMgrLoggingLevel != "" {
		if err = logging.SetLogLevel("peermgr", conf.PeerMgrLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.TransactorLoggingLevel != "" {
		if err = logging.SetLogLevel("transactor", conf.TransactorLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.ActiveOutLoggingLevel != "" {
		if err = logging.SetLogLevel("active-out", conf.ActiveOutLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.InactiveOutLoggingLevel != "" {
		if err = logging.SetLogLevel("inactive-out", conf.InactiveOutLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.ActiveInLoggingLevel != "" {
		if err = logging.SetLogLevel("active-in", conf.ActiveInLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.InactiveInLoggingLevel != "" {
		if err = logging.SetLogLevel("inactive-in", conf.InactiveInLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.PServMgrLoggingLevel != "" {
		if err = logging.SetLogLevel("pservmgr", conf.PServMgrLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.RouteStoreLoggingLevel != "" {
		if err = logging.SetLogLevel("routestore", conf.RouteStoreLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.SubStoreLoggingLevel != "" {
		if err = logging.SetLogLevel("substore", conf.SubStoreLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.PayMgrLoggingLevel != "" {
		if err = logging.SetLogLevel("paymentmgr", conf.PayMgrLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.SettleMgrLoggingLevel != "" {
		if err = logging.SetLogLevel("settlemgr", conf.SettleMgrLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.RenewMgrLoggingLevel != "" {
		if err = logging.SetLogLevel("renewmgr", conf.RenewMgrLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.ReservMgrLoggingLevel != "" {
		if err = logging.SetLogLevel("reservmgr", conf.ReservMgrLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.OfferMgrLoggingLevel != "" {
		if err = logging.SetLogLevel("offermgr", conf.OfferMgrLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.PaychMonitorLoggingLevel != "" {
		if err = logging.SetLogLevel("paychmonitor", conf.PaychMonitorLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.PieceMgrLoggingLevel != "" {
		if err = logging.SetLogLevel("piecemgr", conf.PieceMgrLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.CServMgrLoggingLevel != "" {
		if err = logging.SetLogLevel("cservmgr", conf.CServMgrLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.MinerProofStoreLoggingLevel != "" {
		if err = logging.SetLogLevel("mpstore", conf.MinerProofStoreLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.AddrProtoLoggingLevel != "" {
		if err = logging.SetLogLevel("addrproto", conf.AddrProtoLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.PaychProtoLoggingLevel != "" {
		if err = logging.SetLogLevel("paychproto", conf.PaychProtoLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.RouteProtoLoggingLevel != "" {
		if err = logging.SetLogLevel("routeproto", conf.RouteProtoLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.POfferProtoLoggingLevel != "" {
		if err = logging.SetLogLevel("payofferproto", conf.POfferProtoLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.COfferProtoLoggingLevel != "" {
		if err = logging.SetLogLevel("cidofferproto", conf.COfferProtoLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.PayProtoLoggingLevel != "" {
		if err = logging.SetLogLevel("payproto", conf.PayProtoLoggingLevel); err != nil {
			return nil, err
		}
	}
	if conf.RetMgrLoggingLevel != "" {
		if err = logging.SetLogLevel("retrievalmgr", conf.RetMgrLoggingLevel); err != nil {
			return nil, err
		}
	}
	// Create path
	err = os.MkdirAll(conf.Path, os.ModePerm)
	if err != nil {
		return nil, err
	}
	// New signer
	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{
		Path: filepath.Join(conf.Path, "signer"),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			signer.Shutdown()
		}
	}()
	// New peer manager
	peerMgr, err := peermgr.NewPeerManagerImpl(ctx, peermgr.Opts{
		Path: filepath.Join(conf.Path, "peermgr"),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			peerMgr.Shutdown()
		}
	}()
	// New transactor
	transactor, err := trans.NewTransactorImpl(ctx, signer, trans.Opts{
		FilecoinEnabled:    conf.TransactorFilecoinEnabled,
		FilecoinAPI:        conf.TransactorFilecoinAPI,
		FilecoinAuthToken:  conf.TransactorFilecoinAuthToken,
		FilecoinConfidence: &conf.TransactorFilecoinConfidence,
	})
	if err != nil {
		return nil, err
	}
	// New active out
	activeOut, err := paychstore.NewPaychStoreImpl(ctx, true, true, paychstore.Opts{
		Path:      filepath.Join(conf.Path, "activeout"),
		DSTimeout: conf.ActiveOutDSTimeout,
		DSRetry:   conf.ActiveOutDSRetry,
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			activeOut.Shutdown()
		}
	}()
	// New inactive out
	inactiveOut, err := paychstore.NewPaychStoreImpl(ctx, false, true, paychstore.Opts{
		Path:      filepath.Join(conf.Path, "inactiveout"),
		DSTimeout: conf.InactiveOutDSTimeout,
		DSRetry:   conf.InactiveOutDSRetry,
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			inactiveOut.Shutdown()
		}
	}()
	// New active in
	activeIn, err := paychstore.NewPaychStoreImpl(ctx, true, false, paychstore.Opts{
		Path:      filepath.Join(conf.Path, "activein"),
		DSTimeout: conf.ActiveInDSTimeout,
		DSRetry:   conf.ActiveInDSRetry,
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			activeIn.Shutdown()
		}
	}()
	// New inactive in
	inactiveIn, err := paychstore.NewPaychStoreImpl(ctx, false, false, paychstore.Opts{
		Path:      filepath.Join(conf.Path, "inactivein"),
		DSTimeout: conf.InactiveInDSTimeout,
		DSRetry:   conf.InactiveInDSRetry,
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			inactiveIn.Shutdown()
		}
	}()
	// New paych serving manager
	pservMgr, err := pservmgr.NewPaychServingManagerImpl(ctx, pservmgr.Opts{
		Path: filepath.Join(conf.Path, "pservmgr"),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			pservMgr.Shutdown()
		}
	}()
	// New reservation manager
	reservMgr, err := reservmgr.NewReservationManagerImpl(ctx, reservmgr.Opts{
		Path: filepath.Join(conf.Path, "reservmgr"),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			reservMgr.Shutdown()
		}
	}()
	// New route store
	routeStore, err := routestore.NewRouteStoreImpl(ctx, signer, routestore.Opts{
		Path:         filepath.Join(conf.Path, "routestore"),
		CleanFreq:    conf.RouteStoreCleanFreq,
		CleanTimeout: conf.RouteStoreCleanTimeout,
		MaxHopFIL:    conf.RouteStoreMaxHopFIL,
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			routeStore.Shutdown()
		}
	}()
	// New subscriber store
	subStore, err := substore.NewSubStoreImpl(ctx, substore.Opts{
		Path: filepath.Join(conf.Path, "substore"),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			subStore.Shutdown()
		}
	}()
	// New payment manager
	payMgr, err := paymgr.NewPaymentManagerImpl(ctx, activeOut, activeIn, inactiveOut, inactiveIn, pservMgr, reservMgr, routeStore, subStore, transactor, signer, paymgr.Opts{
		Path:             filepath.Join(conf.Path, "paymgr"),
		CacheSyncFreq:    conf.PayMgrCacheSyncFreq,
		ResCleanFreq:     conf.PayMgrResCleanFreq,
		ResCleanTimeout:  conf.PayMgrResCleanTimeout,
		PeerCleanFreq:    conf.PayMgrPeerCleanFreq,
		PeerCleanTimeout: conf.PayMgrPeerCleanTimeout,
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			payMgr.Shutdown()
		}
	}()
	// New settlement manager
	settleMgr, err := settlemgr.NewSettlementManagerImpl(ctx, settlemgr.Opts{
		Path: filepath.Join(conf.Path, "settlemgr"),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			settleMgr.Shutdown()
		}
	}()
	// New renew manager
	renewMgr, err := renewmgr.NewRenewManagerImpl(ctx, renewmgr.Opts{
		Path: filepath.Join(conf.Path, "renewmgr"),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			renewMgr.Shutdown()
		}
	}()
	// New offer manager
	offerMgr, err := offermgr.NewOfferManagerImpl(ctx, offermgr.Opts{
		Path: filepath.Join(conf.Path, "offermgr"),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			offerMgr.Shutdown()
		}
	}()
	// New paych monitor
	paychMonitor, err := paychmon.NewPaychMonitorImpl(ctx, payMgr, transactor, paychmon.Opts{
		Path: filepath.Join(conf.Path, "paychmon"),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			paychMonitor.Shutdown()
		}
	}()
	// New piece manager
	pieceMgr, err := piecemgr.NewPieceManagerImpl(ctx, piecemgr.Opts{
		PsPath:    filepath.Join(conf.Path, "ps"),
		BsPath:    filepath.Join(conf.Path, "bs"),
		BrefsPath: filepath.Join(conf.Path, "brefs"),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			pieceMgr.Shutdown()
		}
	}()
	// New piece serving manager
	cservMgr, err := cservmgr.NewPieceServingManager(ctx, cservmgr.Opts{
		Path: filepath.Join(conf.Path, "cservmgr"),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			cservMgr.Shutdown()
		}
	}()
	// New miner proof store
	mps, err := mpstore.NewMinerProofStoreImpl(ctx, signer, mpstore.Opts{
		Path: filepath.Join(conf.Path, "mps"),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			mps.Shutdown()
		}
	}()
	// New host
	var dualDHT *dual.DHT
	conMgr, err := connmgr.NewConnManager(
		100,
		400,
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			conMgr.Close()
		}
	}()
	// Try to read key
	var p2pKey crypto2.PrivKey
	_, err = os.Stat(filepath.Join(conf.Path, keyFile))
	if err != nil {
		if os.IsNotExist(err) {
			// Try to generate and save key.
			// Create a p2p key
			p2pKey, _, err = crypto2.GenerateKeyPair(
				crypto2.Ed25519,
				-1,
			)
			if err != nil {
				return nil, fmt.Errorf("error generating p2p key: %v", err.Error())
			}
			prvBytes, err := p2pKey.Raw()
			if err != nil {
				return nil, fmt.Errorf("error getting private key bytes: %v", err.Error())
			}
			err = ioutil.WriteFile(filepath.Join(conf.Path, keyFile), []byte(base64.StdEncoding.EncodeToString(prvBytes)), os.ModePerm)
			if err != nil {
				return nil, fmt.Errorf("error saving p2p key: %v", err.Error())
			}
		} else {
			return nil, err
		}
	} else {
		// Try to load key.
		keyBytes, err := ioutil.ReadFile(filepath.Join(conf.Path, keyFile))
		if err != nil {
			return nil, fmt.Errorf("error loading p2p key: %v", err.Error())
		}
		// Decode key
		key, err := base64.StdEncoding.DecodeString(string(keyBytes))
		if err != nil {
			return nil, fmt.Errorf("error decoding p2p key: %v", err.Error())
		}
		p2pKey, err = crypto2.UnmarshalEd25519PrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling p2p key: %v", err.Error())
		}
	}
	h, err := libp2p.New(
		libp2p.Identity(p2pKey),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%v", conf.P2PPort)),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Security(noise.ID, noise.New),
		libp2p.DefaultTransports,
		libp2p.ConnectionManager(conMgr),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			dualDHT, err = dual.New(ctx, h, dual.DHTOption(dht.ProtocolPrefix("/fc-retrieval-calibr")))
			return dualDHT, err
		}),
		libp2p.EnableAutoRelay(),
		libp2p.UserAgent("go-fcr-"+version.Version),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			h.Close()
		}
	}()
	// New addr protocol
	addrProto := addrproto.NewAddrProtocol(h, dualDHT, signer, peerMgr, addrproto.Opts{
		IOTimeout:   conf.AddrProtoIOTimeout,
		OpTimeout:   conf.AddrProtoOPTimeout,
		PublishFreq: conf.AddrProtoPublishFreq,
	})
	defer func() {
		if err != nil {
			addrProto.Shutdown()
		}
	}()
	// New paych protocol
	paychProto := paychproto.NewPaychProtocol(h, addrProto, signer, peerMgr, offerMgr, settleMgr, renewMgr, payMgr, paychMonitor, paychproto.Opts{
		IOTimeout:   conf.PaychProtoIOTimeout,
		OpTimeout:   conf.PaychProtoOPTimeout,
		OfferExpiry: conf.PaychProtoOfferExpiry,
		RenewWindow: conf.PaychProtoRenewWindow,
	})
	defer func() {
		if err != nil {
			paychProto.Shutdown()
		}
	}()
	// New route protocol
	routeProto := routeproto.NewRouteProtocol(h, addrProto, paychProto, signer, peerMgr, pservMgr, routeStore, subStore, routeproto.Opts{
		IOTimeout:   conf.RouteProtoIOTimeout,
		OpTimeout:   conf.RouteProtoOPTimeout,
		PublishFreq: conf.RouteProtoPublishFreq,
		RouteExpiry: conf.RouteProtoRouteExpiry,
		PublishWait: conf.RouteProtoPublishWait,
	})
	defer func() {
		if err != nil {
			routeProto.Shutdown()
		}
	}()
	// New pay offer protocol
	pOfferProto := pofferproto.NewOfferProtocol(h, addrProto, signer, peerMgr, offerMgr, pservMgr, routeStore, payMgr, pofferproto.Opts{
		IOTimeout:       conf.POfferProtoIOTimeout,
		OpTimeout:       conf.POfferProtoOPTimeout,
		OfferExpiry:     conf.POfferProtoOfferExpiry,
		OfferInactivity: conf.POfferProtoOfferInactivity,
	})
	defer func() {
		if err != nil {
			pOfferProto.Shutdown()
		}
	}()
	// New piece offer protocol
	cOfferProto := cofferproto.NewOfferProtocol(h, dualDHT, addrProto, signer, peerMgr, offerMgr, pieceMgr, cservMgr, mps, cofferproto.Opts{
		IOTimeout:       conf.COfferProtoIOTimeout,
		OpTimeout:       conf.COfferProtoOPTimeout,
		OfferExpiry:     conf.COfferProtoOfferExpiry,
		OfferInactivity: conf.COfferProtoOfferInactivity,
		PublishFreq:     conf.COfferPublishFreq,
	})
	defer func() {
		if err != nil {
			cOfferProto.Shutdown()
		}
	}()
	// New pay protocol
	payProto, err := payproto.NewPayProtocol(ctx, h, addrProto, signer, peerMgr, payMgr, payproto.Opts{
		Path:         filepath.Join(conf.Path, "payproto"),
		IOTimeout:    conf.PayProtoIOTimeout,
		OpTimeout:    conf.PayProtoOPTimeout,
		CleanFreq:    conf.PayProtoCleanFreq,
		CleanTimeout: conf.PayProtoCleanTimeout,
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			payProto.Shutdown()
		}
	}()
	// New retrieval manager
	retMgr, err := retmgr.NewRetrievalManager(ctx, h, addrProto, payProto, signer, peerMgr, pieceMgr, retmgr.Opts{
		CachePath: filepath.Join(conf.Path, "retcache"),
		TempPath:  filepath.Join(conf.Path, "rettmp"),
		IOTimeout: conf.RetMgrIOTimeout,
		OpTimeout: conf.RetMgrOPTimeout,
	})
	if err != nil {
		return nil, err
	}
	shutdown := func() {
		retMgr.Shutdown()
		payProto.Shutdown()
		cOfferProto.Shutdown()
		routeProto.Shutdown()
		pOfferProto.Shutdown()
		paychProto.Shutdown()
		addrProto.Shutdown()
		h.Close()
		conMgr.Close()
		mps.Shutdown()
		cservMgr.Shutdown()
		pieceMgr.Shutdown()
		paychMonitor.Shutdown()
		offerMgr.Shutdown()
		renewMgr.Shutdown()
		settleMgr.Shutdown()
		payMgr.Shutdown()
		subStore.Shutdown()
		routeStore.Shutdown()
		reservMgr.Shutdown()
		pservMgr.Shutdown()
		inactiveIn.Shutdown()
		activeIn.Shutdown()
		inactiveOut.Shutdown()
		activeOut.Shutdown()
		peerMgr.Shutdown()
		signer.Shutdown()
	}
	return &Node{
		Host:             h,
		Signer:           signer,
		PeerMgr:          peerMgr,
		Transactor:       transactor,
		ActiveOutPaych:   activeOut,
		InactiveOutPaych: inactiveOut,
		ActiveInPaych:    activeIn,
		InactiveInPaych:  inactiveIn,
		PServMgr:         pservMgr,
		ReservMgr:        reservMgr,
		RouteStore:       routeStore,
		SubStore:         subStore,
		PayMgr:           payMgr,
		SettleMgr:        settleMgr,
		RenewMgr:         renewMgr,
		OfferMgr:         offerMgr,
		PaychMonitor:     paychMonitor,
		PieceMgr:         pieceMgr,
		CServMgr:         cservMgr,
		MinerProofStore:  mps,
		AddrProto:        addrProto,
		PaychProto:       paychProto,
		PofferProto:      pOfferProto,
		RouteProto:       routeProto,
		CofferProto:      cOfferProto,
		PayProto:         payProto,
		RetMgr:           retMgr,
		shutdown:         shutdown,
	}, nil
}

// Shutdown safely closes all components and services.
func (n *Node) Shutdown() {
	n.shutdown()
}
