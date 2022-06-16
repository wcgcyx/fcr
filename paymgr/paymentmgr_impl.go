package paymgr

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
	"math/big"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/locking"
	"github.com/wcgcyx/fcr/mdlstore"
	"github.com/wcgcyx/fcr/paychstate"
	"github.com/wcgcyx/fcr/paychstore"
	"github.com/wcgcyx/fcr/pservmgr"
	"github.com/wcgcyx/fcr/reservmgr"
	"github.com/wcgcyx/fcr/routestore"
	"github.com/wcgcyx/fcr/substore"
	"github.com/wcgcyx/fcr/trans"
)

// PaymentManagerImpl is the implementation of the PaymentManager interface.
type PaymentManagerImpl struct {
	// Signer
	signer crypto.Signer

	// Transactor
	transactor trans.Transactor

	// Payment channel serving Manager
	pservMgr pservmgr.PaychServingManager

	// Reservation manager
	reservMgr reservmgr.ReservationManager

	// Route store
	rs routestore.RouteStore

	// Subscriber store
	subs substore.SubStore

	// Process related.
	routineCtx context.Context

	// High speed channel state cache.
	// Map from direction (true for outbound) -> currency id -> (Map from toAddr -> (Map from chAddr -> channel state))
	active     map[bool]map[byte]map[string]map[string]*cachedPaychState
	activeLock *locking.LockNode

	// Low speed channel state store
	activeOutStore   paychstore.PaychStore
	activeInStore    paychstore.PaychStore
	inactiveOutStore paychstore.PaychStore
	inactiveInStore  paychstore.PaychStore

	// Shutdown function.
	shutdown func()

	// Options
	tempDir          string
	cacheSyncFreq    time.Duration
	resCleanFreq     time.Duration
	resCleanTimeout  time.Duration
	peerCleanFreq    time.Duration
	peerCleanTimeout time.Duration
}

// NewPaymentManagerImpl creates a new PaymentManagerImpl.
//
// @input - context, currencies, active out channel store, active in channel store, inactive out channel store, inactive in channel store, transactor, signer, options.
//
// @output - payment manager, error.
func NewPaymentManagerImpl(
	ctx context.Context,
	activeOutStore paychstore.PaychStore,
	activeInStore paychstore.PaychStore,
	inactiveOutStore paychstore.PaychStore,
	inactiveInStore paychstore.PaychStore,
	pservMgr pservmgr.PaychServingManager,
	reservMgr reservmgr.ReservationManager,
	rs routestore.RouteStore,
	subs substore.SubStore,
	transactor trans.Transactor,
	signer crypto.Signer,
	opts Opts,
) (*PaymentManagerImpl, error) {
	// Parse options
	cacheSyncFreq := opts.CacheSyncFreq
	if cacheSyncFreq == 0 {
		cacheSyncFreq = defaultCacheSyncFreq
	}
	resCleanFreq := opts.ResCleanFreq
	if resCleanFreq == 0 {
		resCleanFreq = defaultResCleanFreq
	}
	resCleanTimeout := opts.ResCleanTimeout
	if resCleanTimeout == 0 {
		resCleanTimeout = defaultResCleanTimeout
	}
	peerCleanFreq := opts.PeerCleanFreq
	if peerCleanFreq == 0 {
		peerCleanFreq = defaultPeerCleanFreq
	}
	peerCleanTimeout := opts.PeerCleanTimeout
	if peerCleanTimeout == 0 {
		peerCleanTimeout = defaultPeerCleanTimeout
	}
	log.Infof("Start payment manager...")
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
	txn, err := ds.NewTransaction(ctx, true)
	if err != nil {
		log.Errorf("Fail to start new transaction: %v", err.Error())
		return nil, err
	}
	defer txn.Discard(context.Background())
	// Load cache
	active := make(map[bool]map[byte]map[string]map[string]*cachedPaychState)
	activeLock := locking.CreateLockNode()
	active[true] = make(map[byte]map[string]map[string]*cachedPaychState)
	active[false] = make(map[byte]map[string]map[string]*cachedPaychState)
	// Load active in
	curChan, errChan1 := activeInStore.ListCurrencyIDs(ctx)
	for currencyID := range curChan {
		peerChan, errChan2 := activeInStore.ListPeers(ctx, currencyID)
		for peer := range peerChan {
			paychChan, errChan3 := activeInStore.ListPaychsByPeer(ctx, currencyID, peer)
			for paych := range paychChan {
				cs, err := activeInStore.Read(ctx, currencyID, peer, paych)
				if err != nil {
					log.Errorf("Fail to read %v-%v-%v from active in: %v", currencyID, peer, paych, err.Error())
					return nil, err
				}
				_, ok := active[false][currencyID]
				if !ok {
					active[false][currencyID] = make(map[string]map[string]*cachedPaychState)
				}
				_, ok = active[false][currencyID][peer]
				if !ok {
					active[false][currencyID][peer] = make(map[string]*cachedPaychState)
				}
				active[false][currencyID][peer][paych] = &cachedPaychState{
					state: &cs,
				}
				activeLock.Insert(false, currencyID, peer, paych)
			}
			err = <-errChan3
			if err != nil {
				log.Errorf("Fail to list paychs for %v-%v: %v", currencyID, peer, err.Error())
				return nil, err
			}
		}
		err = <-errChan2
		if err != nil {
			log.Errorf("Fail to list peers for %v: %v", currencyID, err.Error())
			return nil, err
		}
	}
	err = <-errChan1
	if err != nil {
		log.Errorf("Fail to list currencies: %v", err.Error())
		return nil, err
	}
	// Load active out
	curChan, errChan1 = activeOutStore.ListCurrencyIDs(ctx)
	for currencyID := range curChan {
		peerChan, errChan2 := activeOutStore.ListPeers(ctx, currencyID)
		for peer := range peerChan {
			paychChan, errChan3 := activeOutStore.ListPaychsByPeer(ctx, currencyID, peer)
			for paych := range paychChan {
				cs, err := activeOutStore.Read(ctx, currencyID, peer, paych)
				if err != nil {
					log.Errorf("Fail to read %v-%v-%v from active out: %v", currencyID, peer, paych, err.Error())
					return nil, err
				}
				_, ok := active[true][currencyID]
				if !ok {
					active[true][currencyID] = make(map[string]map[string]*cachedPaychState)
				}
				_, ok = active[true][currencyID][peer]
				if !ok {
					active[true][currencyID][peer] = make(map[string]*cachedPaychState)
				}
				// Check if has reservation.
				exists, err := txn.Has(ctx, currencyID, peer, paych)
				if err != nil {
					log.Errorf("Fail to check if contains %v-%v-%v in reservation ds: %v", currencyID, peer, paych, err.Error())
					return nil, err
				}
				if !exists {
					active[true][currencyID][peer][paych] = &cachedPaychState{
						state:         &cs,
						reservedAmt:   big.NewInt(0),
						reservedNonce: 0,
						reservations:  make(map[uint64]*reservation),
					}
				} else {
					val, err := txn.Get(ctx, currencyID, peer, paych)
					if err != nil {
						log.Errorf("Fail to get reservation for %v-%v-%v: %v", currencyID, peer, paych, err.Error())
						return nil, err
					}
					stored, err := decDSVal(val)
					if err != nil {
						log.Errorf("Fail to decode ds value %v: %v", val, err.Error())
						return nil, err
					}
					active[true][currencyID][peer][paych] = &cachedPaychState{
						state:         &cs,
						reservedAmt:   stored.reservedAmt,
						reservedNonce: stored.reservedNonce,
						reservations:  stored.reservations,
					}
				}
				activeLock.Insert(true, currencyID, peer, paych)
			}
			err = <-errChan3
			if err != nil {
				log.Errorf("Fail to list paychs for %v-%v: %v", currencyID, peer, err.Error())
				return nil, err
			}
		}
		err = <-errChan2
		if err != nil {
			log.Errorf("Fail to list peers for %v: %v", currencyID, err.Error())
			return nil, err
		}
	}
	err = <-errChan1
	if err != nil {
		log.Errorf("Fail to list currencies: %v", err.Error())
		return nil, err
	}
	// Create routine context
	routineCtx, cancel := context.WithCancel(context.Background())
	mgr := &PaymentManagerImpl{
		signer:           signer,
		transactor:       transactor,
		pservMgr:         pservMgr,
		reservMgr:        reservMgr,
		rs:               rs,
		subs:             subs,
		active:           active,
		activeLock:       activeLock,
		activeOutStore:   activeOutStore,
		activeInStore:    activeInStore,
		inactiveOutStore: inactiveOutStore,
		inactiveInStore:  inactiveInStore,
		tempDir:          opts.Path,
		cacheSyncFreq:    cacheSyncFreq,
		resCleanFreq:     resCleanFreq,
		resCleanTimeout:  resCleanTimeout,
		peerCleanFreq:    peerCleanFreq,
		peerCleanTimeout: peerCleanTimeout,
		routineCtx:       routineCtx,
	}
	mgr.shutdown = func() {
		// Shutdown routines
		log.Infof("Shutdown all routines...")
		cancel()
		// Save reservations
		log.Infof("Save reservations and push state...")
		_, err = mgr.activeLock.Lock(context.Background())
		if err != nil {
			log.Errorf("Fail to obtain lock for active lock: %v", err.Error())
		}
		// Clear the current temp ds.
		err = os.RemoveAll(mgr.tempDir)
		if err != nil {
			log.Errorf("Fail to clear temp ds: %v", err.Error())
		}
		err = os.Mkdir(mgr.tempDir, os.ModePerm)
		if err != nil {
			log.Errorf("Fail to create temp dir: %v", err.Error())
		}
		ds, err := mdlstore.NewMultiDimLockableStoreImpl(context.Background(), mgr.tempDir)
		if err != nil {
			log.Errorf("Error creating temp ds to store payment manager reservation data: %v", err.Error())
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
		for currencyID := range mgr.active[false] {
			for fromAddr := range mgr.active[false][currencyID] {
				for chAddr := range mgr.active[false][currencyID][fromAddr] {
					cs := mgr.active[false][currencyID][fromAddr][chAddr].state
					mgr.activeInStore.Upsert(paychstate.State{
						CurrencyID:         cs.CurrencyID,
						FromAddr:           cs.FromAddr,
						ToAddr:             cs.ToAddr,
						ChAddr:             cs.ChAddr,
						Balance:            big.NewInt(0).Set(cs.Balance),
						Redeemed:           big.NewInt(0).Set(cs.Redeemed),
						Nonce:              cs.Nonce,
						Voucher:            cs.Voucher,
						NetworkLossVoucher: cs.NetworkLossVoucher,
						StateNonce:         cs.StateNonce,
					})
				}
			}
		}
		for currencyID := range mgr.active[true] {
			for toAddr := range mgr.active[true][currencyID] {
				for chAddr := range mgr.active[true][currencyID][toAddr] {
					cs := mgr.active[true][currencyID][toAddr][chAddr].state
					mgr.activeOutStore.Upsert(paychstate.State{
						CurrencyID:         cs.CurrencyID,
						FromAddr:           cs.FromAddr,
						ToAddr:             cs.ToAddr,
						ChAddr:             cs.ChAddr,
						Balance:            big.NewInt(0).Set(cs.Balance),
						Redeemed:           big.NewInt(0).Set(cs.Redeemed),
						Nonce:              cs.Nonce,
						Voucher:            cs.Voucher,
						NetworkLossVoucher: cs.NetworkLossVoucher,
						StateNonce:         cs.StateNonce,
					})
					if txn != nil {
						dsVal, err := encDSVal(mgr.active[true][currencyID][toAddr][chAddr])
						if err != nil {
							log.Errorf("Fail to encode %v-%v-%v: %v", currencyID, toAddr, chAddr, err.Error())
						} else {
							if err = txn.Put(context.Background(), dsVal, currencyID, toAddr, chAddr); err != nil {
								log.Errorf("Fail to put %v in %v-%v-%v: %v", dsVal, currencyID, toAddr, chAddr, err.Error())
							}
						}
					}
				}
			}
		}
		if txn != nil {
			err = txn.Commit(context.Background())
			if err != nil {
				log.Errorf("Fail to commit transaction: %v", err.Error())
			}
		}
	}
	go mgr.syncRoutine()
	go mgr.reservationCleanRoutine()
	go mgr.peerCleanRoutine()
	return mgr, nil
}

// Shutdown safely shuts down the component.
func (mgr *PaymentManagerImpl) Shutdown() {
	log.Infof("Start shutdown...")
	mgr.shutdown()
}

// Reserve is used to reserve a certain amount.
//
// @input - context, currency id, recipient address, petty amount required, whether reservation requires served channel, optional received offer, first expiration time, subsequent allowed inactivity time, peer addr who requests the reservation.
//
// @output - reserved inbound price-per-byte, reserved inbound period, reserved channel address, reservation id, error.
func (mgr *PaymentManagerImpl) Reserve(ctx context.Context, currencyID byte, toAddr string, pettyAmtRequired *big.Int, servedRequired bool, receivedOffer *fcroffer.PayOffer, expiration time.Time, inactivity time.Duration, peerAddr string) (*big.Int, *big.Int, string, uint64, error) {
	log.Debugf("Start reserving at %v-%v with %v, serving required %v, received offer %v", currencyID, toAddr, pettyAmtRequired, servedRequired, receivedOffer)
	if time.Now().After(expiration) {
		return nil, nil, "", 0, fmt.Errorf("expiration in the past, got %v, now %v", expiration, time.Now())
	}
	// Make copies of the parameters.
	pettyAmtRequired = big.NewInt(0).Set(pettyAmtRequired)
	outboundPPP := big.NewInt(0)
	outboundPeriod := big.NewInt(0)
	if receivedOffer != nil {
		outboundPPP.Set(receivedOffer.PPP)
		outboundPeriod.Set(receivedOffer.Period)
	}
	// Check if parameters are legal.
	if pettyAmtRequired.Cmp(big.NewInt(0)) <= 0 {
		return nil, nil, "", 0, fmt.Errorf("petty amount required must be positive, got %v", pettyAmtRequired)
	}
	if !((outboundPPP.Cmp(big.NewInt(0)) == 0 && outboundPeriod.Cmp(big.NewInt(0)) == 0) || (outboundPPP.Cmp(big.NewInt(0)) > 0 && outboundPeriod.Cmp(big.NewInt(0)) > 0)) {
		return nil, nil, "", 0, fmt.Errorf("outbound ppp and period must be either both 0 or both positive, got ppp %v, period %v", outboundPPP, outboundPeriod)
	}

	// Calculate gross amount required.
	var grossAmtRequired *big.Int
	if outboundPPP.Cmp(big.NewInt(0)) > 0 {
		// Has outbound surcharge.
		// gross amt required = petty amt + ((petty amt - 1) / period + 1) * ppp
		temp := big.NewInt(0).Sub(pettyAmtRequired, big.NewInt(1))
		temp = big.NewInt(0).Div(temp, outboundPeriod)
		temp = big.NewInt(0).Add(temp, big.NewInt(1))
		temp = big.NewInt(0).Mul(temp, outboundPPP)
		grossAmtRequired = big.NewInt(0).Add(temp, pettyAmtRequired)
	} else {
		// No outbound surcharge.
		grossAmtRequired = pettyAmtRequired
	}

	// RLock peer state.
	release, err := mgr.activeLock.RLock(ctx, true, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", true, currencyID, toAddr, err.Error())
		return nil, nil, "", 0, err
	}
	defer release()

	// Check if currencyID and toAddr exists.
	_, ok := mgr.active[true][currencyID]
	if !ok {
		return nil, nil, "", 0, fmt.Errorf("no channels existed for currency id %v", currencyID)
	}
	ps, ok := mgr.active[true][currencyID][toAddr]
	if !ok {
		return nil, nil, "", 0, fmt.Errorf("no channels existed for recipient %v", toAddr)
	}

	// Use a sub ctx to communicate between routines.
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Reservation result.
	var resInboundPPP *big.Int
	var resInboundPeriod *big.Int
	var resChAddr string
	var resID uint64
	resLock := sync.RWMutex{}

	// Do a quick shuffle
	chs := make([]string, 0)
	for chAddr := range ps {
		chs = append(chs, chAddr)
	}
	for i := range chs {
		j := rand.Intn(i + 1)
		chs[i], chs[j] = chs[j], chs[i]
	}
	// Start go routines to attempt reservation at every channel at the same time.
	wg := sync.WaitGroup{}
	for _, chAddr := range chs {
		wg.Add(1)
		log.Debugf("Attempt to do reservation on %v-%v-%v-%v", true, currencyID, toAddr, chAddr)
		go func(chAddr string) {
			defer wg.Done()
			// Do a provisional check to see if reservation can be attempted.
			// Check serving status.
			served, inboundPPP, inboundPeriod, err := mgr.pservMgr.Inspect(ctx, currencyID, toAddr, chAddr)
			if err != nil {
				return
			}
			if servedRequired && !served {
				return
			}
			if peerAddr != "" {
				// Check reservation policy.
				unlimited, max, err := mgr.reservMgr.GetPeerPolicy(ctx, currencyID, chAddr, peerAddr)
				if err != nil {
					return
				}
				if !unlimited && grossAmtRequired.Cmp(max) > 0 {
					// Does not meet policy requirement
					return
				}
			}
			if servedRequired {
				if !((inboundPPP.Cmp(big.NewInt(0)) == 0 && inboundPeriod.Cmp(big.NewInt(0)) == 0) || (inboundPPP.Cmp(big.NewInt(0)) > 0 && inboundPeriod.Cmp(big.NewInt(0)) > 0)) {
					// inbound ppp and period must be either both 0 or both positive
					return
				}
				// If outbound ppp is not 0, calibrate inbound ppp.
				inboundPPP.Add(inboundPPP, outboundPPP)
				if outboundPeriod.Cmp(big.NewInt(0)) > 0 && outboundPeriod.Cmp(inboundPeriod) < 0 {
					inboundPeriod = outboundPeriod
				}
			} else {
				inboundPPP = big.NewInt(0)
				inboundPeriod = big.NewInt(0)
			}
			subRelease, err := mgr.activeLock.RLock(subCtx, true, currencyID, toAddr, chAddr)
			if err != nil {
				log.Debugf("Fail to obtain read lock for %v-%v-%v-%v: %v", true, currencyID, toAddr, chAddr, err.Error())
				return
			}

			cs := ps[chAddr]
			if cs.state.NetworkLossVoucher != "" {
				// There is network loss.
				subRelease()
				return
			}
			// Total current expired.
			expired := getExpiredAmount(cs)
			available := big.NewInt(0).Sub(cs.state.Balance, big.NewInt(0).Add(cs.reservedAmt, cs.state.Redeemed))
			available.Add(available, expired)
			if available.Cmp(grossAmtRequired) < 0 {
				// Available is less than required.
				subRelease()
				return
			}
			// Provisional check passed. Do the actual reservation.
			subRelease()

			subRelease, err = mgr.activeLock.Lock(subCtx, true, currencyID, toAddr, chAddr)
			if err != nil {
				log.Debugf("Fail to obtain write lock for %v-%v-%v-%v: %v", true, currencyID, toAddr, chAddr, err.Error())
				return
			}
			defer subRelease()

			cs = ps[chAddr]
			// First, check network loss.
			if cs.state.NetworkLossVoucher != "" {
				// There is network loss.
				return
			}
			// Second, do a cleaning & check if balance is still sufficient.
			releaseExpiredAmount(cs)
			available = big.NewInt(0).Sub(cs.state.Balance, big.NewInt(0).Add(cs.reservedAmt, cs.state.Redeemed))
			if available.Cmp(grossAmtRequired) < 0 {
				// Available is less than required.
				return
			}
			// Third, shutdown other routines.
			cancel()
			resLock.Lock()
			defer resLock.Unlock()
			if resChAddr != "" {
				// Other routine has succeed.
				return
			}
			// Succeed, reserve.
			log.Debugf("Succeed reserve in channel %v-%v-%v-%v", true, currencyID, toAddr, chAddr)
			resInboundPPP = big.NewInt(0).Set(inboundPPP)
			resInboundPeriod = big.NewInt(0).Set(inboundPeriod)
			resChAddr = chAddr
			resID = cs.reservedNonce + 1
			cs.reservations[resID] = &reservation{
				inboundPPP:        inboundPPP,
				inboundPeriod:     inboundPeriod,
				pettyAmtReceived:  big.NewInt(0),
				surchargeReceived: big.NewInt(0),
				outboundPPP:       outboundPPP,
				outboundPeriod:    outboundPeriod,
				pettyAmtPaid:      big.NewInt(0),
				surchargePaid:     big.NewInt(0),
				remain:            grossAmtRequired,
				expiration:        expiration,
				inactivity:        inactivity,
				offer:             receivedOffer,
			}
			cs.reservedAmt.Add(cs.reservedAmt, grossAmtRequired)
			cs.reservedNonce = resID
		}(chAddr)
	}
	wg.Wait()

	// Check if reservation succeed.
	if resChAddr == "" {
		// Failed.
		return nil, nil, "", 0, fmt.Errorf("fail to reserve for %v %v with petty amount %v", currencyID, toAddr, pettyAmtRequired)
	}
	// Succeed.
	return resInboundPPP, resInboundPeriod, resChAddr, resID, nil
}

// Pay is used to make a payment.
//
// @input - context, currency id, recipient address, reserved channel address, reservation id, received gross payment to drive this payment, petty amount required.
//
// @output - voucher, potential linked offer, function to commit & record access time, function to revert & record network loss, error.
func (mgr *PaymentManagerImpl) Pay(ctx context.Context, currencyID byte, toAddr string, chAddr string, resID uint64, grossAmtReceived *big.Int, pettyAmtRequired *big.Int) (string, *fcroffer.PayOffer, func(time.Time), func(string), error) {
	log.Debugf("Pay using %v-%v-%v with res id %v gross amount received %v, petty amount required %v", currencyID, toAddr, chAddr, resID, grossAmtReceived, pettyAmtRequired)
	// Make copies of the parameters.
	if grossAmtReceived != nil {
		grossAmtReceived = big.NewInt(0).Set(grossAmtReceived)
	}
	pettyAmtRequired = big.NewInt(0).Set(pettyAmtRequired)

	// Check if parameters are legal.
	if pettyAmtRequired.Cmp(big.NewInt(0)) <= 0 {
		return "", nil, nil, nil, fmt.Errorf("petty amount required must be positive, got %v", pettyAmtRequired)
	}
	if grossAmtReceived != nil && grossAmtReceived.Cmp(big.NewInt(0)) <= 0 {
		return "", nil, nil, nil, fmt.Errorf("gross amount received must be positive, got %v", grossAmtReceived)
	}

	// RLock peer state.
	release, err := mgr.activeLock.RLock(ctx, true, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", true, currencyID, toAddr, err.Error())
		return "", nil, nil, nil, err
	}
	defer release()

	// Check if currencyID and toAddr exists.
	_, ok := mgr.active[true][currencyID]
	if !ok {
		return "", nil, nil, nil, fmt.Errorf("no channels existed for currency id %v", currencyID)
	}
	ps, ok := mgr.active[true][currencyID][toAddr]
	if !ok {
		return "", nil, nil, nil, fmt.Errorf("no channels existed for recipient %v", toAddr)
	}

	// Do a provisional check to see if reservation is valid.
	subRelease, err := mgr.activeLock.RLock(ctx, true, currencyID, toAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v-%v: %v", true, currencyID, toAddr, chAddr, err.Error())
		return "", nil, nil, nil, err
	}

	cs, ok := ps[chAddr]
	if !ok {
		subRelease()
		return "", nil, nil, nil, fmt.Errorf("no channel found for channel address %v", chAddr)
	}
	if cs.state.NetworkLossVoucher != "" {
		// There is network loss.
		subRelease()
		return "", nil, nil, nil, fmt.Errorf("network loss presents for toAddr %v chAddr %v", toAddr, chAddr)
	}
	res, ok := cs.reservations[resID]
	if !ok {
		subRelease()
		return "", nil, nil, nil, fmt.Errorf("no reservation found for reservation id %v", resID)
	}
	// Check if expired.
	if (!res.expiration.IsZero() && time.Now().After(res.expiration)) || (res.expiration.IsZero() && time.Now().After(res.lastAccessed.Add(res.inactivity))) {
		// Expired.
		subRelease()
		return "", nil, nil, nil, fmt.Errorf("reservation for channel %v reservation id %v has expired", chAddr, resID)
	}
	// Check if petty received matches petty amount required.
	if grossAmtReceived != nil {
		// This payment is not for self, check.
		_, _, pettyAmtToReceive := calculatePettyAmountToReceive(res.inboundPPP, res.inboundPeriod, res.pettyAmtReceived, res.surchargeReceived, grossAmtReceived)
		if pettyAmtToReceive.Cmp(pettyAmtRequired) != 0 {
			subRelease()
			return "", nil, nil, nil, fmt.Errorf("petty amount to receive %v does not match petty amount required %v", pettyAmtToReceive, pettyAmtRequired)
		}
	}
	// Check if reservation remaining amount is sufficient for gross amount to pay.
	_, _, grossAmtToPay := calculateGrossAmountToPay(res.outboundPPP, res.outboundPeriod, res.pettyAmtPaid, res.surchargePaid, pettyAmtRequired)
	if grossAmtToPay.Cmp(res.remain) > 0 {
		subRelease()
		return "", nil, nil, nil, fmt.Errorf("reservation remain %v does not cover gross amount to pay %v", res.remain, grossAmtToPay)
	}
	// Provisional check passed. Attempt the actual payment.
	subRelease()

	subRelease, err = mgr.activeLock.Lock(ctx, true, currencyID, toAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v-%v: %v", true, currencyID, toAddr, chAddr, err.Error())
		return "", nil, nil, nil, err
	}

	cs = ps[chAddr]
	// First, check network loss
	if cs.state.NetworkLossVoucher != "" {
		// There is network loss.
		subRelease()
		return "", nil, nil, nil, fmt.Errorf("network loss presents for toAddr %v chAddr %v", toAddr, chAddr)
	}
	// Do a cleaning & check reservations.
	releaseExpiredAmount(cs)
	res, ok = cs.reservations[resID]
	if !ok {
		subRelease()
		return "", nil, nil, nil, fmt.Errorf("no reservation found for reservation id %v", resID)
	}
	// Check if expired.
	if (!res.expiration.IsZero() && time.Now().After(res.expiration)) || (res.expiration.IsZero() && time.Now().After(res.lastAccessed.Add(res.inactivity))) {
		// Expired.
		subRelease()
		return "", nil, nil, nil, fmt.Errorf("reservation has expired")
	}
	// Attempt payment.
	// Make some provisional updates.
	// Only the following elements will be subject to update.
	// Channel state, current reserved amount, reservation.
	stateProvisional := &paychstate.State{
		CurrencyID:         cs.state.CurrencyID,
		FromAddr:           cs.state.FromAddr,
		ToAddr:             cs.state.ToAddr,
		ChAddr:             cs.state.ChAddr,
		Balance:            big.NewInt(0).Set(cs.state.Balance),
		Redeemed:           big.NewInt(0).Set(cs.state.Redeemed),
		Nonce:              cs.state.Nonce,
		Voucher:            cs.state.Voucher,
		NetworkLossVoucher: cs.state.NetworkLossVoucher,
		StateNonce:         cs.state.StateNonce,
	}
	reservedAmtProvisional := big.NewInt(0).Set(cs.reservedAmt)
	resProvisional := &reservation{
		inboundPPP:        big.NewInt(0).Set(res.inboundPPP),
		inboundPeriod:     big.NewInt(0).Set(res.inboundPeriod),
		pettyAmtReceived:  big.NewInt(0).Set(res.pettyAmtReceived),
		surchargeReceived: big.NewInt(0).Set(res.surchargeReceived),
		outboundPPP:       big.NewInt(0).Set(res.outboundPPP),
		outboundPeriod:    big.NewInt(0).Set(res.outboundPeriod),
		pettyAmtPaid:      big.NewInt(0).Set(res.pettyAmtPaid),
		surchargePaid:     big.NewInt(0).Set(res.surchargePaid),
		remain:            big.NewInt(0).Set(res.remain),
		lastAccessed:      res.lastAccessed,
		expiration:        res.expiration,
		inactivity:        res.inactivity,
		offer:             res.offer,
	}
	if grossAmtReceived != nil {
		newPettyAmtReceived, newSurchargeReceived, pettyAmtToReceive := calculatePettyAmountToReceive(res.inboundPPP, res.inboundPeriod, res.pettyAmtReceived, res.surchargeReceived, grossAmtReceived)
		if pettyAmtToReceive.Cmp(pettyAmtRequired) != 0 {
			subRelease()
			return "", nil, nil, nil, fmt.Errorf("petty amount to receive %v does not match petty amount required %v", pettyAmtToReceive, pettyAmtRequired)
		}
		resProvisional.pettyAmtReceived = newPettyAmtReceived
		resProvisional.surchargeReceived = newSurchargeReceived
	}
	newPettyAmtPaid, newSurchargePaid, grossAmtToPay := calculateGrossAmountToPay(res.outboundPPP, res.outboundPeriod, res.pettyAmtPaid, res.surchargePaid, pettyAmtRequired)
	if grossAmtToPay.Cmp(res.remain) > 0 {
		subRelease()
		return "", nil, nil, nil, fmt.Errorf("reservation remain %v does not cover gross amount to pay %v", res.remain, grossAmtToPay)
	}
	resProvisional.pettyAmtPaid = newPettyAmtPaid
	resProvisional.surchargePaid = newSurchargePaid
	resProvisional.remain.Sub(resProvisional.remain, grossAmtToPay)
	reservedAmtProvisional.Sub(reservedAmtProvisional, grossAmtToPay)
	if resProvisional.remain.Cmp(big.NewInt(0)) == 0 {
		// Reservation completed.
		resProvisional = nil
	}
	voucher, err := mgr.transactor.GenerateVoucher(ctx, cs.state.CurrencyID, cs.state.ChAddr, 0, cs.state.Nonce+1, big.NewInt(0).Add(cs.state.Redeemed, grossAmtToPay))
	if err != nil {
		subRelease()
		return "", nil, nil, nil, err
	}
	stateProvisional.Redeemed.Add(stateProvisional.Redeemed, grossAmtToPay)
	stateProvisional.Nonce += 1
	stateProvisional.Voucher = voucher
	stateProvisional.StateNonce += 1
	commit := func(newLastAccessed time.Time) {
		defer subRelease()
		if resProvisional == nil {
			delete(cs.reservations, resID)
		} else {
			resProvisional.lastAccessed = newLastAccessed
			resProvisional.expiration = time.Time{}
			cs.reservations[resID] = resProvisional
		}
		cs.reservedAmt = reservedAmtProvisional
		cs.state = stateProvisional
	}
	revert := func(networkLossVoucher string) {
		defer subRelease()
		if networkLossVoucher != "" {
			// Check network loss voucher.
			sender, channel, lane, nonce, redeemed, err := mgr.transactor.VerifyVoucher(currencyID, networkLossVoucher)
			if err != nil || sender != cs.state.FromAddr || channel != cs.state.ChAddr || lane != 0 || nonce <= cs.state.Nonce || redeemed.Cmp(cs.state.Redeemed) < 0 {
				// Invalid network loss voucher.
				return
			}
			// Record network loss
			cs.state.NetworkLossVoucher = networkLossVoucher
			cs.state.StateNonce += 1
			// Delete all reservations.
			cs.reservedAmt = big.NewInt(0)
			cs.reservations = make(map[uint64]*reservation)
		}
	}
	return voucher, res.offer, commit, revert, nil
}

// Receive is used to receive a payment in a voucher.
//
// @input - context, currency id, voucher.
//
// @output - received gross amount, function to commit, function to revert, the lastest voucher recorded previously if network loss presents, error.
func (mgr *PaymentManagerImpl) Receive(ctx context.Context, currencyID byte, voucher string) (*big.Int, func(), func(), string, error) {
	log.Debugf("Received payment for %v with %v", currencyID, voucher)
	fromAddr, chAddr, lane, nonce, redeemed, err := mgr.transactor.VerifyVoucher(currencyID, voucher)
	if err != nil {
		return nil, nil, nil, "", err
	}
	if lane != 0 {
		return nil, nil, nil, "", fmt.Errorf("lane is not 0")
	}

	// RLock peer state.
	release, err := mgr.activeLock.RLock(ctx, false, currencyID, fromAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", false, currencyID, fromAddr, err.Error())
		return nil, nil, nil, "", err
	}
	defer release()

	// Check if currencyID and fromAddr exists.
	_, ok := mgr.active[false][currencyID]
	if !ok {
		return nil, nil, nil, "", fmt.Errorf("no channels existed for currency id %v", currencyID)
	}
	ps, ok := mgr.active[false][currencyID][fromAddr]
	if !ok {
		return nil, nil, nil, "", fmt.Errorf("no channels existed for sender %v", fromAddr)
	}

	// Do a provisional check to see if voucher is valid
	subRelease, err := mgr.activeLock.RLock(ctx, false, currencyID, fromAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v-%v: %v", false, currencyID, fromAddr, chAddr, err.Error())
		return nil, nil, nil, "", err
	}

	cs, ok := ps[chAddr]
	if !ok {
		subRelease()
		return nil, nil, nil, "", fmt.Errorf("no channel found for channel address %v", chAddr)
	}
	if nonce <= cs.state.Nonce {
		// Nonce is not increasing, there is network loss.
		subRelease()
		return nil, nil, nil, cs.state.Voucher, fmt.Errorf("network loss for %v presents", chAddr)
	}
	// Check if redeemed is smaller than recorded.
	if redeemed.Cmp(cs.state.Redeemed) < 0 {
		// There is a network loss.
		subRelease()
		return nil, nil, nil, cs.state.Voucher, fmt.Errorf("invalid voucher received, redeemed %v is smaller than current %v", redeemed, cs.state.Redeemed)
	}
	// Check if redeemed is smaller than balance.
	if redeemed.Cmp(cs.state.Balance) > 0 {
		// redeemed is larger than balance, check balance.
		_, _, _, _, balance, _, _, err := mgr.transactor.Check(ctx, currencyID, chAddr)
		if err != nil {
			subRelease()
			return nil, nil, nil, "", err
		}
		if redeemed.Cmp(balance) > 0 {
			// still larger than balance.
			subRelease()
			return nil, nil, nil, "", fmt.Errorf("invalid voucher received, redeemed %v is larger than balance %v", redeemed, balance)
		}
	}
	// Provisional check passed. Attempt the actual receive.
	subRelease()

	subRelease, err = mgr.activeLock.Lock(ctx, false, currencyID, fromAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v-%v: %v", false, currencyID, fromAddr, chAddr, err.Error())
		return nil, nil, nil, "", err
	}

	cs = ps[chAddr]
	if nonce <= cs.state.Nonce {
		// Nonce is not increasing, there is network loss.
		subRelease()
		return nil, nil, nil, cs.state.Voucher, fmt.Errorf("network loss for %v presents", chAddr)
	}
	if redeemed.Cmp(cs.state.Redeemed) < 0 {
		// There is a network loss.
		subRelease()
		return nil, nil, nil, "", fmt.Errorf("invalid voucher received, redeemed %v is smaller than current %v", redeemed, cs.state.Redeemed)
	}
	newBalance := big.NewInt(0).Set(cs.state.Balance)
	if redeemed.Cmp(cs.state.Balance) > 0 {
		// redeemed is larger than balance, check balance.
		_, _, _, _, balance, _, _, err := mgr.transactor.Check(ctx, currencyID, chAddr)
		if err != nil {
			subRelease()
			return nil, nil, nil, "", err
		}
		if redeemed.Cmp(balance) > 0 {
			// still larger than balance.
			subRelease()
			return nil, nil, nil, "", fmt.Errorf("invalid voucher received, redeemed %v is larger than balance %v", redeemed, balance)
		}
		newBalance = balance
	}
	// Make some provisional updates.
	stateProvisional := &paychstate.State{
		CurrencyID:         cs.state.CurrencyID,
		FromAddr:           cs.state.FromAddr,
		ToAddr:             cs.state.ToAddr,
		ChAddr:             cs.state.ChAddr,
		Balance:            newBalance,
		Redeemed:           big.NewInt(0).Set(cs.state.Redeemed),
		Nonce:              cs.state.Nonce,
		Voucher:            cs.state.Voucher,
		NetworkLossVoucher: cs.state.NetworkLossVoucher,
		StateNonce:         cs.state.StateNonce,
	}
	grossAmtReceived := big.NewInt(0).Sub(redeemed, cs.state.Redeemed)
	stateProvisional.Redeemed = redeemed
	stateProvisional.Nonce = nonce
	stateProvisional.Voucher = voucher
	stateProvisional.StateNonce += 1
	commit := func() {
		defer subRelease()
		cs.state = stateProvisional
	}
	revert := func() {
		defer subRelease()
	}
	return grossAmtReceived, commit, revert, "", nil
}

// ReserveForSelf is used to reserve a certain amount for self to spend for configured time.
//
// @input - context, currency id, recipient address, petty amount, first expiration time, subsequent allowed inactivity time.
//
// @output - reserved channel address, reservation id, error.
func (mgr *PaymentManagerImpl) ReserveForSelf(ctx context.Context, currencyID byte, toAddr string, pettyAmt *big.Int, expiration time.Time, inactivity time.Duration) (string, uint64, error) {
	_, _, resChAddr, resID, err := mgr.Reserve(ctx, currencyID, toAddr, pettyAmt, false, nil, expiration, inactivity, "")
	return resChAddr, resID, err
}

// ReserveForSelfWithOffer is used to reserve a certain amount for self to spend based on a received offer.
//
// @input - context, received offer.
//
// @output - reserved channel address, reservation id, error.
func (mgr *PaymentManagerImpl) ReserveForSelfWithOffer(ctx context.Context, receivedOffer fcroffer.PayOffer) (string, uint64, error) {
	_, _, resChAddr, resID, err := mgr.Reserve(ctx, receivedOffer.CurrencyID, receivedOffer.SrcAddr, receivedOffer.Amt, false, &receivedOffer, receivedOffer.Expiration, receivedOffer.Inactivity, "")
	return resChAddr, resID, err
}

// ReserveForOthersIntermediate is used to reserve some amount for others as an intermediate node based on required inbound surcharge and a received offer, peer addr who requests the reservation.
//
// @input - context, received offer.
//
// @output - reserved inbound price-per-byte, reserved inbound period, reserved channel address, reservation id, error.
func (mgr *PaymentManagerImpl) ReserveForOthersIntermediate(ctx context.Context, receivedOffer fcroffer.PayOffer, peerAddr string) (*big.Int, *big.Int, string, uint64, error) {
	return mgr.Reserve(ctx, receivedOffer.CurrencyID, receivedOffer.SrcAddr, receivedOffer.Amt, true, &receivedOffer, receivedOffer.Expiration, receivedOffer.Inactivity, peerAddr)
}

// ReserveForOthersFinal is used to reserve some amount for others as a final node based on required inbound surcharge, peer addr who requests the reservation.
//
// @input - context, currency id, recipient address, petty amount, first expiration time, subsequent allowed inactivity time.
//
// @output - reserved inbound price-per-byte, reserved inbound period, reserved channel address, reservation id, error.
func (mgr *PaymentManagerImpl) ReserveForOthersFinal(ctx context.Context, currencyID byte, toAddr string, pettyAmt *big.Int, expiration time.Time, inactivity time.Duration, peerAddr string) (*big.Int, *big.Int, string, uint64, error) {
	return mgr.Reserve(ctx, currencyID, toAddr, pettyAmt, true, nil, expiration, inactivity, peerAddr)
}

// PayForSelf is used to make a payment for self.
//
// @input - context, currency id, recipient address, reserved channel address, reservation id, petty amount required.
//
// @output - voucher, potential linked offer, function to commit & record access time, function to revert & record network loss, error.
func (mgr *PaymentManagerImpl) PayForSelf(ctx context.Context, currencyID byte, toAddr string, chAddr string, resID uint64, pettyAmtRequired *big.Int) (string, *fcroffer.PayOffer, func(time.Time), func(string), error) {
	return mgr.Pay(ctx, currencyID, toAddr, chAddr, resID, nil, pettyAmtRequired)
}

// PayForOthers is used to make a payment for others.
//
// @input - context, currency id, recipient address, reserved channel address, reservation id, gross amount received, petty amount required.
//
// @output - voucher, potential linked offer, function to commit & record access time, function to revert & record network loss, error.
func (mgr *PaymentManagerImpl) PayForOthers(ctx context.Context, currencyID byte, toAddr string, chAddr string, resID uint64, grossAmtReceived *big.Int, pettyAmtRequired *big.Int) (string, *fcroffer.PayOffer, func(time.Time), func(string), error) {
	if grossAmtReceived == nil {
		return "", nil, nil, nil, fmt.Errorf("gross amount received can not be nil")
	}
	return mgr.Pay(ctx, currencyID, toAddr, chAddr, resID, grossAmtReceived, pettyAmtRequired)
}

// AddInboundChannel is used to add an inbound payment channel.
//
// @input - context, currency id, from address, channel address.
//
// @output - error.
func (mgr *PaymentManagerImpl) AddInboundChannel(ctx context.Context, currencyID byte, fromAddr string, chAddr string) error {
	log.Debugf("Add inbound channel for %v-%v-%v", currencyID, fromAddr, chAddr)
	// Get inbound channel state.
	settlingAt, minSettlingAt, _, redeemed, balance, sender, recipient, err := mgr.transactor.Check(ctx, currencyID, chAddr)
	if err != nil {
		return err
	}
	// Check settlingAt
	if settlingAt != 0 {
		return fmt.Errorf("channel %v is settling/settled", chAddr)
	}
	// Check minSettlingAt
	if minSettlingAt != 0 {
		return fmt.Errorf("channel %v has min settling at", chAddr)
	}
	// Check sender and recipient.
	if sender != fromAddr {
		return fmt.Errorf("channel sender does not match, should be %v but got %v", fromAddr, sender)
	}
	_, selfAddr, err := mgr.signer.GetAddr(ctx, currencyID)
	if err != nil {
		return err
	}
	if recipient != selfAddr {
		return fmt.Errorf("channel recipient does not match, should be %v but got %v", selfAddr, recipient)
	}
	// Check first
	release, err := mgr.activeLock.RLock(ctx, false, currencyID, fromAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v-%v: %v", false, currencyID, fromAddr, chAddr, err.Error())
		return err
	}

	_, ok := mgr.active[false][currencyID]
	if ok {
		_, ok = mgr.active[false][currencyID][fromAddr]
		if ok {
			_, ok = mgr.active[false][currencyID][fromAddr][chAddr]
			if ok {
				release()
				log.Debugf("channel %v already exists", chAddr)
				return nil
			}
		}
	}
	// Check passed, start insert.
	release()

	release, err = mgr.activeLock.Lock(ctx, false, currencyID, fromAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v-%v: %v", false, currencyID, fromAddr, chAddr, err.Error())
		return err
	}
	defer release()
	// Add channel.
	cs := &cachedPaychState{
		state: &paychstate.State{
			CurrencyID:         currencyID,
			FromAddr:           sender,
			ToAddr:             recipient,
			ChAddr:             chAddr,
			Balance:            balance,
			Redeemed:           redeemed,
			Nonce:              0,
			Voucher:            "",
			NetworkLossVoucher: "",
			StateNonce:         0,
		},
	}
	// Check again.
	_, ok = mgr.active[false][currencyID]
	if ok {
		_, ok = mgr.active[false][currencyID][fromAddr]
		if ok {
			_, ok = mgr.active[false][currencyID][fromAddr][chAddr]
			if ok {
				log.Debugf("channel %v already exists", chAddr)
				return nil
			}
		}
	} else {
		mgr.active[false][currencyID] = make(map[string]map[string]*cachedPaychState)
	}
	_, ok = mgr.active[false][currencyID][fromAddr]
	if !ok {
		mgr.active[false][currencyID][fromAddr] = make(map[string]*cachedPaychState)
	}
	mgr.active[false][currencyID][fromAddr][chAddr] = cs
	// Push to store
	mgr.activeInStore.Upsert(paychstate.State{
		CurrencyID:         currencyID,
		FromAddr:           sender,
		ToAddr:             recipient,
		ChAddr:             chAddr,
		Balance:            big.NewInt(0).Set(balance),
		Redeemed:           big.NewInt(0).Set(redeemed),
		Nonce:              0,
		Voucher:            "",
		NetworkLossVoucher: "",
		StateNonce:         0,
	})
	mgr.activeLock.Insert(false, currencyID, fromAddr, chAddr)
	// Add a new subscriber.
	err = mgr.subs.AddSubscriber(ctx, currencyID, fromAddr)
	if err != nil {
		log.Warnf("Fail to add subscriber for %v-%v: %v", currencyID, fromAddr, err.Error())
	}
	return nil
}

// RetireInboundChannel is used to retire an inbound payment channel.
//
// @input - context, currency id, from address, channel address.
//
// @output - error.
func (mgr *PaymentManagerImpl) RetireInboundChannel(ctx context.Context, currencyID byte, fromAddr string, chAddr string) error {
	log.Debugf("Retire inbound channel for %v-%v-%v", currencyID, fromAddr, chAddr)
	// Check first.
	release, err := mgr.activeLock.RLock(ctx, false, currencyID, fromAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", false, currencyID, fromAddr, err.Error())
		return err
	}

	_, ok := mgr.active[false][currencyID]
	if !ok {
		release()
		return fmt.Errorf("no channels existed for currency id %v", currencyID)
	}
	_, ok = mgr.active[false][currencyID][fromAddr]
	if !ok {
		release()
		return fmt.Errorf("no channels existed for sender %v", fromAddr)
	}
	_, ok = mgr.active[false][currencyID][fromAddr][chAddr]
	if !ok {
		release()
		return fmt.Errorf("no channels existed for channel address %v", chAddr)
	}
	release()

	// Check passed, start removing.
	release, err = mgr.activeLock.Lock(ctx, false, currencyID, fromAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v: %v", false, currencyID, fromAddr, err.Error())
		return err
	}
	defer release()

	// Check again.
	_, ok = mgr.active[false][currencyID]
	if !ok {
		return fmt.Errorf("no channels existed for currency id %v", currencyID)
	}
	_, ok = mgr.active[false][currencyID][fromAddr]
	if !ok {
		return fmt.Errorf("no channels existed for sender %v", fromAddr)
	}
	_, ok = mgr.active[false][currencyID][fromAddr][chAddr]
	if !ok {
		return fmt.Errorf("no channels existed for channel address %v", chAddr)
	}
	// Retire state.
	mgr.activeInStore.Remove(currencyID, fromAddr, chAddr)
	mgr.inactiveInStore.Upsert(*mgr.active[false][currencyID][fromAddr][chAddr].state)
	delete(mgr.active[false][currencyID][fromAddr], chAddr)
	mgr.activeLock.Remove(false, currencyID, fromAddr, chAddr)
	if len(mgr.active[false][currencyID][fromAddr]) == 0 {
		err = mgr.subs.RemoveSubscriber(ctx, currencyID, fromAddr)
		if err != nil {
			log.Warnf("Fail to remove subscriber for %v-%v: %v", currencyID, fromAddr, err.Error())
		}
	}
	return nil
}

// AddOutboundChannel is used to add an outbound payment channel.
//
// @input - context, currency id, recipient address, channel address.
//
// @output - error.
func (mgr *PaymentManagerImpl) AddOutboundChannel(ctx context.Context, currencyID byte, toAddr string, chAddr string) error {
	log.Debugf("Add outbound channel for %v-%v-%v", currencyID, toAddr, chAddr)
	// Get inbound channel state.
	settlingAt, minSettlingAt, _, redeemed, balance, sender, recipient, err := mgr.transactor.Check(ctx, currencyID, chAddr)
	if err != nil {
		return err
	}
	// Check settlingAt
	if settlingAt != 0 {
		return fmt.Errorf("channel %v is settling/settled", chAddr)
	}
	// Check minSettlingAt
	if minSettlingAt != 0 {
		return fmt.Errorf("channel %v has min settling at", chAddr)
	}
	// Check sender and recipient.
	if recipient != toAddr {
		return fmt.Errorf("channel recipient does not match, should be %v but got %v", toAddr, recipient)
	}
	_, selfAddr, err := mgr.signer.GetAddr(ctx, currencyID)
	if err != nil {
		return err
	}
	if sender != selfAddr {
		return fmt.Errorf("channel sender does not match, should be %v but got %v", selfAddr, sender)
	}
	// Check first
	release, err := mgr.activeLock.RLock(ctx, true, currencyID, toAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v-%v: %v", true, currencyID, toAddr, chAddr, err.Error())
		return err
	}

	_, ok := mgr.active[true][currencyID]
	if ok {
		_, ok = mgr.active[true][currencyID][toAddr]
		if ok {
			_, ok = mgr.active[true][currencyID][toAddr][chAddr]
			if ok {
				release()
				log.Debugf("channel %v already exists", chAddr)
				return nil
			}
		}
	}
	// Check passed, start insert.
	release()

	release, err = mgr.activeLock.Lock(ctx, true, currencyID, toAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v-%v: %v", true, currencyID, toAddr, chAddr, err.Error())
		return err
	}
	defer release()

	// Add channel.
	cs := &cachedPaychState{
		state: &paychstate.State{
			CurrencyID:         currencyID,
			FromAddr:           sender,
			ToAddr:             recipient,
			ChAddr:             chAddr,
			Balance:            balance,
			Redeemed:           redeemed,
			Nonce:              0,
			Voucher:            "",
			NetworkLossVoucher: "",
			StateNonce:         0,
		},
		reservedAmt:   big.NewInt(0),
		reservedNonce: 0,
		reservations:  make(map[uint64]*reservation),
	}
	// Check again
	_, ok = mgr.active[true][currencyID]
	if ok {
		_, ok = mgr.active[true][currencyID][toAddr]
		if ok {
			_, ok = mgr.active[true][currencyID][toAddr][chAddr]
			if ok {
				release()
				log.Debugf("channel %v already exists", chAddr)
				return nil
			}
		}
	} else {
		mgr.active[true][currencyID] = make(map[string]map[string]*cachedPaychState)
	}
	_, ok = mgr.active[true][currencyID][toAddr]
	if !ok {
		mgr.active[true][currencyID][toAddr] = make(map[string]*cachedPaychState)
	}
	mgr.active[true][currencyID][toAddr][chAddr] = cs
	// Push to store
	mgr.activeOutStore.Upsert(paychstate.State{
		CurrencyID:         currencyID,
		FromAddr:           sender,
		ToAddr:             recipient,
		ChAddr:             chAddr,
		Balance:            big.NewInt(0).Set(balance),
		Redeemed:           big.NewInt(0).Set(redeemed),
		Nonce:              0,
		Voucher:            "",
		NetworkLossVoucher: "",
		StateNonce:         0,
	})
	mgr.activeLock.Insert(true, currencyID, toAddr, chAddr)
	// Add a new direct link.
	err = mgr.rs.AddDirectLink(ctx, currencyID, toAddr)
	if err != nil {
		log.Errorf("error adding direct link for %v-%v: %v", currencyID, toAddr, err.Error())
	}
	return nil
}

// RetireOutboundChannel is used to retire an outbound payment channel.
//
// @input - context, currency id, recipient address, channel address.
//
// @output - error.
func (mgr *PaymentManagerImpl) RetireOutboundChannel(ctx context.Context, currencyID byte, toAddr string, chAddr string) error {
	log.Debugf("Retire outbound channel for %v-%v-%v", currencyID, toAddr, chAddr)
	// Check first.
	release, err := mgr.activeLock.RLock(ctx, true, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", true, currencyID, toAddr, err.Error())
		return err
	}

	_, ok := mgr.active[true][currencyID]
	if !ok {
		release()
		return fmt.Errorf("no channels existed for currency id %v", currencyID)
	}
	_, ok = mgr.active[true][currencyID][toAddr]
	if !ok {
		release()
		return fmt.Errorf("no channels existed for recipient %v", toAddr)
	}
	_, ok = mgr.active[true][currencyID][toAddr][chAddr]
	if !ok {
		release()
		return fmt.Errorf("no channels existed for channel address %v", chAddr)
	}
	release()

	// Check passed, start removing.
	release, err = mgr.activeLock.Lock(ctx, true, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v: %v", true, currencyID, toAddr, err.Error())
		return err
	}
	defer release()

	// Check again.
	_, ok = mgr.active[true][currencyID]
	if !ok {
		return fmt.Errorf("no channels existed for currency id %v", currencyID)
	}
	_, ok = mgr.active[true][currencyID][toAddr]
	if !ok {
		return fmt.Errorf("no channels existed for sender %v", toAddr)
	}
	_, ok = mgr.active[true][currencyID][toAddr][chAddr]
	if !ok {
		return fmt.Errorf("no channels existed for channel address %v", chAddr)
	}
	// Retire state.
	mgr.activeOutStore.Remove(currencyID, toAddr, chAddr)
	mgr.inactiveOutStore.Upsert(*mgr.active[true][currencyID][toAddr][chAddr].state)
	delete(mgr.active[true][currencyID][toAddr], chAddr)
	mgr.activeLock.Remove(true, currencyID, toAddr, chAddr)
	if len(mgr.active[true][currencyID][toAddr]) == 0 {
		err = mgr.rs.RemoveDirectLink(ctx, currencyID, toAddr)
		if err != nil {
			log.Error("error removing direct link for %v-%v: %v", currencyID, toAddr, err.Error())
		}
	}
	return nil
}

// UpdateOutboundChannelBalance is used to update the balance of an outbound payment channel.
//
// @input - context, currency id, recipient address, channel address.
//
// @output - error.
func (mgr *PaymentManagerImpl) UpdateOutboundChannelBalance(ctx context.Context, currencyID byte, toAddr string, chAddr string) error {
	log.Debugf("Update outbound channel for %v-%v-%v", currencyID, toAddr, chAddr)
	// Check first.
	release, err := mgr.activeLock.RLock(ctx, true, currencyID, toAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v-%v: %v", true, currencyID, toAddr, chAddr, err.Error())
		return err
	}

	_, ok := mgr.active[true][currencyID]
	if !ok {
		release()
		return fmt.Errorf("no channels existed for currency id %v", currencyID)
	}
	_, ok = mgr.active[true][currencyID][toAddr]
	if !ok {
		release()
		return fmt.Errorf("no channels existed for recipient %v", toAddr)
	}
	_, ok = mgr.active[true][currencyID][toAddr][chAddr]
	if !ok {
		release()
		return fmt.Errorf("no channels existed for channel address %v", chAddr)
	}
	_, _, _, _, balance, _, _, err := mgr.transactor.Check(ctx, currencyID, chAddr)
	if err != nil {
		release()
		return err
	}
	if mgr.active[true][currencyID][toAddr][chAddr].state.Balance.Cmp(balance) == 0 {
		release()
		return fmt.Errorf("channel %v balance does not need to be updated", chAddr)
	}
	release()

	// Check passed, start removing.
	release, err = mgr.activeLock.Lock(ctx, true, currencyID, toAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v-%v: %v", true, currencyID, toAddr, chAddr, err.Error())
		return err
	}
	defer release()

	// Check again.
	_, ok = mgr.active[true][currencyID]
	if !ok {
		return fmt.Errorf("no channels existed for currency id %v", currencyID)
	}
	_, ok = mgr.active[true][currencyID][toAddr]
	if !ok {
		return fmt.Errorf("no channels existed for sender %v", toAddr)
	}
	_, ok = mgr.active[true][currencyID][toAddr][chAddr]
	if !ok {
		return fmt.Errorf("no channels existed for channel address %v", chAddr)
	}
	_, _, _, _, balance, _, _, err = mgr.transactor.Check(ctx, currencyID, chAddr)
	if err != nil {
		return err
	}
	if mgr.active[true][currencyID][toAddr][chAddr].state.Balance.Cmp(balance) == 0 {
		return fmt.Errorf("channel %v balance does not need to be updated", chAddr)
	}
	mgr.active[true][currencyID][toAddr][chAddr].state.Balance = balance
	mgr.active[true][currencyID][toAddr][chAddr].state.StateNonce += 1
	return nil
}

// BearNetworkLoss is used to bear and ignore the network loss of a given channel.
//
// @input - context, currency id, recipient address, channel address.
//
// @output - error.
func (mgr *PaymentManagerImpl) BearNetworkLoss(ctx context.Context, currencyID byte, toAddr string, chAddr string) error {
	log.Debugf("Bear network loss for %v-%v-%v", currencyID, toAddr, chAddr)
	// Check first.
	release, err := mgr.activeLock.RLock(ctx, true, currencyID, toAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v-%v: %v", true, currencyID, toAddr, chAddr, err.Error())
		return err
	}

	_, ok := mgr.active[true][currencyID]
	if !ok {
		release()
		return fmt.Errorf("no channels existed for currency id %v", currencyID)
	}
	_, ok = mgr.active[true][currencyID][toAddr]
	if !ok {
		release()
		return fmt.Errorf("no channels existed for recipient %v", toAddr)
	}
	_, ok = mgr.active[true][currencyID][toAddr][chAddr]
	if !ok {
		release()
		return fmt.Errorf("no channels existed for channel address %v", chAddr)
	}
	if mgr.active[true][currencyID][toAddr][chAddr].state.NetworkLossVoucher == "" {
		release()
		return fmt.Errorf("channel %v is does not have network loss", chAddr)
	}
	release()

	// Check passed, start process.
	release, err = mgr.activeLock.Lock(ctx, true, currencyID, toAddr, chAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v-%v: %v", true, currencyID, toAddr, chAddr, err.Error())
		return err
	}
	defer release()

	// Check again.
	_, ok = mgr.active[true][currencyID]
	if !ok {
		return fmt.Errorf("no channels existed for currency id %v", currencyID)
	}
	_, ok = mgr.active[true][currencyID][toAddr]
	if !ok {
		return fmt.Errorf("no channels existed for recipient %v", toAddr)
	}
	_, ok = mgr.active[true][currencyID][toAddr][chAddr]
	if !ok {
		return fmt.Errorf("no channels existed for channel address %v", chAddr)
	}
	if mgr.active[true][currencyID][toAddr][chAddr].state.NetworkLossVoucher == "" {
		return fmt.Errorf("channel %v is does not have network loss", chAddr)
	}
	_, _, _, nonce, redeemed, err := mgr.transactor.VerifyVoucher(currencyID, mgr.active[true][currencyID][toAddr][chAddr].state.NetworkLossVoucher)
	if err != nil {
		return err
	}
	mgr.active[true][currencyID][toAddr][chAddr].state.Redeemed = redeemed
	mgr.active[true][currencyID][toAddr][chAddr].state.Nonce = nonce
	mgr.active[true][currencyID][toAddr][chAddr].state.Voucher = mgr.active[true][currencyID][toAddr][chAddr].state.NetworkLossVoucher
	mgr.active[true][currencyID][toAddr][chAddr].state.NetworkLossVoucher = ""
	mgr.active[true][currencyID][toAddr][chAddr].state.StateNonce += 1
	return nil
}
