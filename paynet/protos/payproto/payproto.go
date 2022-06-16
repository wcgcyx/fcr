package payproto

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
	"math/big"
	"os"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/locking"
	"github.com/wcgcyx/fcr/mdlstore"
	"github.com/wcgcyx/fcr/paymgr"
	"github.com/wcgcyx/fcr/peermgr"
	"github.com/wcgcyx/fcr/peernet/protos/addrproto"
)

const (
	DirectPayProtocol = "/fcr-calibr/paynet/pay/direct"
	ProxyPayProtocol  = "/fcr-calibr/paynet/pay/proxy"
	ProtocolVersion   = "/0.0.1"
	TempKey           = "tempkey"
)

// Logger
var log = logging.Logger("payproto")

// PayProtocol manages payment protocol.
// It handles payment.
type PayProtocol struct {
	// Host
	h host.Host

	// Addr proto
	addrProto *addrproto.AddrProtocol

	// Signer
	signer crypto.Signer

	// Peer manager
	peerMgr peermgr.PeerManager

	// Payment manager
	payMgr paymgr.PaymentManager

	// all recorded received amount.
	// Map from currency id -> original addr -> amount received.
	// TODO: Maybe give every amount received a timeout, avoid some attacks.
	received     map[byte]map[string]*big.Int
	receivedLock *locking.LockNode

	// Process related.
	routineCtx context.Context

	// Shutdown function.
	shutdown func()

	// Options
	tempDir      string
	ioTimeout    time.Duration
	opTimeout    time.Duration
	cleanFreq    time.Duration
	cleanTimeout time.Duration
}

// NewPayProtocol creates a new payment protocol.
//
// @input - context, host, addr protocol, signer, peer manager, payment manager, options.
//
// @output - payment protocol, error.
func NewPayProtocol(
	ctx context.Context,
	h host.Host,
	addrProto *addrproto.AddrProtocol,
	signer crypto.Signer,
	peerMgr peermgr.PeerManager,
	payMgr paymgr.PaymentManager,
	opts Opts,
) (*PayProtocol, error) {
	// Parse options
	ioTimeout := opts.IOTimeout
	if ioTimeout == 0 {
		ioTimeout = defaultIOTimeout
	}
	opTimeout := opts.OpTimeout
	if opTimeout == 0 {
		opTimeout = defaultOpTimeout
	}
	cleanFreq := opts.CleanFreq
	if cleanFreq == 0 {
		cleanFreq = defaultCleanFreq
	}
	cleanTimeout := opts.CleanTimeout
	if cleanTimeout == 0 {
		cleanTimeout = defaultCleanTimeout
	}
	log.Infof("Start pay protocol...")
	// Open the temp ds
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, opts.Path)
	if err != nil {
		log.Errorf("Fail to open mdlstore: %v", err.Error())
		return nil, err
	}
	defer func() {
		err = ds.Shutdown(context.Background())
		if err != nil {
			log.Errorf("Fail to close temp ds: %v", err.Error())
		}
	}()
	// Load received
	var received map[byte]map[string]*big.Int
	receivedLock := locking.CreateLockNode()
	txn, err := ds.NewTransaction(ctx, true)
	if err != nil {
		log.Errorf("Fail to start new transaction: %v", err.Error())
		return nil, err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, TempKey)
	if err != nil {
		log.Errorf("Fail to check if contains %v: %v", TempKey, err.Error())
		return nil, err
	}
	if exists {
		dsVal, err := txn.Get(ctx, TempKey)
		if err != nil {
			log.Errorf("Fail to read ds value for %v: %v", TempKey, err.Error())
			return nil, err
		}
		err = json.Unmarshal(dsVal, &received)
		if err != nil {
			log.Errorf("Fail to decode ds value: %v", err.Error())
			return nil, err
		}
		for currencyID, addrs := range received {
			for addr := range addrs {
				receivedLock.Insert(currencyID, addr)
			}
		}
	} else {
		received = make(map[byte]map[string]*big.Int)
	}
	// Create routine
	routineCtx, cancel := context.WithCancel(context.Background())
	proto := &PayProtocol{
		h:            h,
		addrProto:    addrProto,
		signer:       signer,
		peerMgr:      peerMgr,
		payMgr:       payMgr,
		received:     received,
		receivedLock: receivedLock,
		routineCtx:   routineCtx,
		tempDir:      opts.Path,
		ioTimeout:    ioTimeout,
		opTimeout:    opTimeout,
		cleanFreq:    cleanFreq,
		cleanTimeout: cleanTimeout,
	}
	h.SetStreamHandler(DirectPayProtocol+ProtocolVersion, proto.handleDirectPay)
	h.SetStreamHandler(ProxyPayProtocol+ProtocolVersion, proto.handleProxyPay)
	proto.shutdown = func() {
		log.Infof("Remove stream handler...")
		// Remove stream handler
		h.RemoveStreamHandler(DirectPayProtocol + ProtocolVersion)
		h.RemoveStreamHandler(ProxyPayProtocol + ProtocolVersion)
		// Close routines
		log.Infof("Stop all routines...")
		cancel()
		// Save received
		_, err = proto.receivedLock.Lock(context.Background())
		if err != nil {
			log.Errorf("Fail to obtain lock for received amount: %v", err.Error())
		}
		// Clear the current temp ds.
		err = os.RemoveAll(proto.tempDir)
		if err != nil {
			log.Errorf("Fail to clear temp ds: %v", err.Error())
		}
		err = os.Mkdir(proto.tempDir, os.ModePerm)
		if err != nil {
			log.Errorf("Fail to create temp dir: %v", err.Error())
		}
		ds, err := mdlstore.NewMultiDimLockableStoreImpl(context.Background(), proto.tempDir)
		if err != nil {
			log.Errorf("Error creating temp ds to store pay protocol received data: %v", err.Error())
		}
		var txn mdlstore.Transaction
		if ds != nil {
			txn, err = ds.NewTransaction(context.Background(), false)
			if err != nil {
				log.Errorf("Fail to start new transaction: %v", err.Error())
			} else {
				defer txn.Discard(context.Background())
			}
		}
		defer func() {
			if ds != nil {
				err = ds.Shutdown(context.Background())
				if err != nil {
					log.Errorf("Fail to close temp ds: %v", err.Error())
				}
			}
		}()
		if txn != nil {
			dsVal, err := json.Marshal(received)
			if err != nil {
				log.Errorf("Fail to encode received amount %v: %v", received, err.Error())
			} else {
				if err = txn.Put(context.Background(), dsVal, TempKey); err != nil {
					log.Errorf("Fail to put %v in %v: %v", dsVal, TempKey, err.Error())
				}
			}
			err = txn.Commit(context.Background())
			if err != nil {
				log.Errorf("Fail to commit transaction: %v", err.Error())
			}
		}
	}
	go proto.cleanRoutine()
	return proto, nil
}

// Shutdown safely shuts down the component.
func (p *PayProtocol) Shutdown() {
	log.Infof("Start shutdown...")
	p.shutdown()
}

// Receive receives amount for given peer.
//
// @input - context, currency id, original address.
//
// @output - amount received, error.
func (p *PayProtocol) Receive(ctx context.Context, currencyID byte, originalAddr string) (*big.Int, error) {
	log.Debugf("Receive amount for %v-%v", currencyID, originalAddr)
	// Do a check first
	release, err := p.receivedLock.RLock(ctx, currencyID, originalAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, originalAddr, err.Error())
		return nil, err
	}

	_, ok := p.received[currencyID]
	if !ok {
		release()
		return big.NewInt(0), nil
	}
	_, ok = p.received[currencyID][originalAddr]
	if !ok {
		release()
		return big.NewInt(0), nil
	}
	if p.received[currencyID][originalAddr].Cmp(big.NewInt(0)) == 0 {
		release()
		return big.NewInt(0), nil
	}
	// Switch from read to write
	release()

	release, err = p.receivedLock.Lock(ctx, currencyID, originalAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, originalAddr, err.Error())
		return nil, err
	}
	defer release()

	_, ok = p.received[currencyID]
	if !ok {
		return big.NewInt(0), nil
	}
	_, ok = p.received[currencyID][originalAddr]
	if !ok {
		return big.NewInt(0), nil
	}
	res := big.NewInt(0).Set(p.received[currencyID][originalAddr])
	p.received[currencyID][originalAddr] = big.NewInt(0)
	return res, nil
}
