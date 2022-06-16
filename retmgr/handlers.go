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
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	gs "github.com/ipfs/go-graphsync"
	"github.com/libp2p/go-libp2p-core/peer"
	golock "github.com/viney-shih/go-lock"
	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/peermgr"
)

// onIncomingRequest is the hook called on incoming request.
func (mgr *RetrievalManager) onIncomingRequest(p peer.ID, request gs.RequestData, hookActions gs.IncomingRequestHookActions) {
	log.Debugf("On incoming request from peer %v: %v", p, request.ID())
	// Verify offer
	offerData, ok := request.Extension(offerExtension)
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("expect piece offer data"))
		log.Debugf("Fail to get offer extention in incoming request")
		return
	}
	offer := fcroffer.PieceOffer{}
	err := offer.Decode(offerData)
	if err != nil {
		hookActions.TerminateWithError(fmt.Errorf("fail to decode piece offer"))
		log.Debugf("Fail to decode offer: %v", err.Error())
		return
	}
	toSign, err := offer.GetToBeSigned()
	if err != nil {
		hookActions.TerminateWithError(fmt.Errorf("internal error"))
		log.Errorf("Fail to get signed data for offer: %v", err.Error())
		return
	}
	if !offer.ID.Equals(request.Root()) {
		hookActions.TerminateWithError(fmt.Errorf("piece offer root id mismatch, expect %v, got %v", request.Root(), offer.ID))
		log.Debugf("Piece offer root id mismatch, expect %v, got %v", request.Root(), offer.ID)
		return
	}
	err = crypto.Verify(offer.CurrencyID, toSign, offer.SignatureType, offer.Signature, offer.RecipientAddr)
	if err != nil {
		hookActions.TerminateWithError(fmt.Errorf("offer fails to verify"))
		log.Debugf("Fail to verify offer signature: %v", err.Error())
		return
	}
	// Verify sender.
	keyType, ok := request.Extension(keyTypeExtension)
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("expect key type"))
		log.Debugf("Fail to get key type extension")
		return
	}
	if len(keyType) != 1 {
		hookActions.TerminateWithError(fmt.Errorf("expect key type of length 1 got %v", len(keyType)))
		log.Debugf("Key type should be length 1 got %v", len(keyType))
		return
	}
	fromAddr, ok := request.Extension(fromExtension)
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("expect from addr"))
		log.Debugf("Fail to get peer address")
		return
	}
	sig, ok := request.Extension(signatureExtension)
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("expect signature"))
		log.Debugf("Fail to get signature")
		return
	}
	err = crypto.Verify(offer.CurrencyID, offerData, keyType[0], sig, string(fromAddr))
	if err != nil {
		hookActions.TerminateWithError(fmt.Errorf("sender signature fails to verify"))
		log.Debugf("Fail to verify sender signature: %v", err.Error())
		return
	}
	// Check if peer is blocked.
	subCtx, cancel := context.WithTimeout(mgr.routineCtx, mgr.opTimeout)
	defer cancel()
	exists, err := mgr.peerMgr.HasPeer(subCtx, offer.CurrencyID, string(fromAddr))
	if err != nil {
		hookActions.TerminateWithError(fmt.Errorf("internal error"))
		log.Warnf("Fail to check if contains peer %v-%v: %v", offer.CurrencyID, string(fromAddr), err.Error())
		return
	}
	if exists {
		// Check if peer is blocked.
		blocked, err := mgr.peerMgr.IsBlocked(subCtx, offer.CurrencyID, string(fromAddr))
		if err != nil {
			hookActions.TerminateWithError(fmt.Errorf("internal error"))
			log.Warnf("Fail to check if peer %v-%v is blocked: %v", offer.CurrencyID, string(fromAddr), err.Error())
			return
		}
		if blocked {
			hookActions.TerminateWithError(fmt.Errorf("Peer %v-%v is blocked", offer.CurrencyID, string(fromAddr)))
			log.Debugf("Peer %v-%v has been blocked, stop processing request", offer.CurrencyID, string(fromAddr))
			return
		}
	}
	offerID, err := getOfferIDRaw(offerData)
	if err != nil {
		hookActions.TerminateWithError(fmt.Errorf("internal error"))
		log.Errorf("Fail to get offer ID, should never happen: %v", err.Error())
		return
	}
	// Check if there is an existing retrieval process linked with this offer id.
	mgr.activeInLock.RLock()
	release := mgr.activeInLock.RUnlock

	state, ok := mgr.activeInOfferID[offerID]
	if ok {
		// There is an existing offer.
		defer release()
		// Check if this offer is currently being processed.
		state.lock.RLock()
		if state.processed {
			// Offer is in process, fail now.
			defer state.lock.RUnlock()
			hookActions.TerminateWithError(fmt.Errorf("there is one active retrieval process using this offer"))
			log.Debugf("There is one active retrieval process using this offer")
			return
		}
		// Offer is not in process, resume the process.
		state.lock.RUnlock()

		state.lock.Lock()
		defer state.lock.Unlock()

		// Check again
		if state.processed {
			// Offer is in process, fail now.
			hookActions.TerminateWithError(fmt.Errorf("there is one active retrieval process using this offer during locking switch"))
			log.Debugf("There is one active retrieval process using this offer")
			return
		}
		// Offer is not in the process. Check if state has expired.
		if state.expiredAt.Before(time.Now()) {
			hookActions.TerminateWithError(fmt.Errorf("offer has expired"))
			log.Debugf("Offer has expired")
			return
		}
		// Check if state is from the same sender.
		if state.fromAddr != string(fromAddr) {
			hookActions.TerminateWithError(fmt.Errorf("offer not used by original sender, expect %v, got %v", state.fromAddr, string(fromAddr)))
			log.Debugf("Offer not used by original sender, expect %v, got %v", state.fromAddr, string(fromAddr))
			return
		}
		state.processed = true
		state.index = 0
	} else {
		// There is no existing offer, switch to lock.
		release()

		mgr.activeInLock.Lock()
		defer mgr.activeInLock.Unlock()

		mgr.cleanActiveIn()
		// Check again
		_, ok = mgr.activeInOfferID[offerID]
		if ok {
			hookActions.TerminateWithError(fmt.Errorf("there is one active retrieval process using this offer during locking switch"))
			log.Debugf("There is one active retrieval process using this offer")
			return
		}
		// Insert a new state.
		mgr.activeInOfferID[offerID] = &incomingRetrievalState{
			currencyID:      offer.CurrencyID,
			fromAddr:        string(fromAddr),
			inactivity:      offer.Inactivity,
			ppb:             offer.PPB,
			index:           0,
			paymentRequired: big.NewInt(0),
			expiredAt:       time.Now().Add(offer.Inactivity),
			lock:            golock.NewCASMutex(),
			processed:       true,
		}
	}
	go func() {
		rec := peermgr.Record{
			Description: fmt.Sprintf("peer starts to request retriaval of %v", offer.ID),
			CreatedAt:   time.Now(),
		}
		err := mgr.peerMgr.AddToHistory(mgr.routineCtx, offer.CurrencyID, string(fromAddr), rec)
		if err != nil {
			log.Warnf("Fail add %v to history of %v-%v: %v", rec, offer.CurrencyID, string(fromAddr), err.Error())
		}
	}()
	hookActions.ValidateRequest()
}

// onOutgoingBlock is the hook called on outgoing block.
func (mgr *RetrievalManager) onOutgoingBlock(p peer.ID, request gs.RequestData, block gs.BlockData, hookActions gs.OutgoingBlockHookActions) {
	log.Debugf("On outgoing block: index %v, size %v, size on wire %v", block.Index(), block.BlockSize(), block.BlockSizeOnWire())
	// Get offer ID
	offerData, ok := request.Extension(offerExtension)
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("expect piece offer data"))
		log.Debugf("Fail to get offer extention in incoming request")
		return
	}
	offerID, err := getOfferIDRaw(offerData)
	if err != nil {
		hookActions.TerminateWithError(fmt.Errorf("internal error"))
		log.Errorf("Fail to get offer id: %v", err.Error())
		return
	}
	// Get state
	mgr.activeInLock.RLock()
	release := mgr.activeInLock.RUnlock
	defer release()

	state, ok := mgr.activeInOfferID[offerID]
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("offer not found, possibly expired"))
		log.Debugf("Fail to find offer state, possibly expired")
		return
	}

	state.lock.Lock()
	defer state.lock.Unlock()

	if state.ppb.Cmp(big.NewInt(0)) > 0 {
		// Need to request payment info.
		log.Debugf("Need to request payment info")
		if block.Index() > state.index {
			// Haven't paid yet.
			state.paymentRequired.Mul(state.ppb, big.NewInt(int64(block.BlockSizeOnWire())))
			hookActions.SendExtensionData(gs.ExtensionData{Name: paymentExtension, Data: []byte{1}})
			log.Debugf("Request paused for %v-%v", p, request.ID())
			hookActions.PauseResponse()
		}
	} else {
		state.expiredAt = time.Now().Add(state.inactivity)
	}
}

// onUpdatedRequest is the hook called on updated request.
func (mgr *RetrievalManager) onUpdatedRequest(p peer.ID, request gs.RequestData, updateRequest gs.RequestData, hookActions gs.RequestUpdatedHookActions) {
	log.Debugf("On updated request for %v", updateRequest.ID())
	// Get offer ID
	offerData, ok := request.Extension(offerExtension)
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("expect piece offer data"))
		log.Debugf("Fail to get offer extention in incoming request")
		return
	}
	offerID, err := getOfferIDRaw(offerData)
	if err != nil {
		hookActions.TerminateWithError(fmt.Errorf("internal error"))
		log.Errorf("Fail to get offer id: %v", err.Error())
		return
	}
	b, ok := updateRequest.Extension(paymentIndexExtension)
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("payment index not found"))
		log.Debugf("Fail to get payment index extention in incoming request")
		return
	}
	index := int64(binary.LittleEndian.Uint64(b))
	b, ok = updateRequest.Extension(paymentExtension)
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("payment not found"))
		log.Debugf("Fail to get payment extention in incoming request")
		return
	}
	paid := big.NewInt(0).SetBytes(b)
	// Get state
	mgr.activeInLock.RLock()
	release := mgr.activeInLock.RUnlock
	defer release()

	state, ok := mgr.activeInOfferID[offerID]
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("offer not found, possibly expired"))
		log.Debugf("Fail to find offer state, possibly expired")
		return
	}

	state.lock.Lock()
	defer state.lock.Unlock()

	// Check if correct payment is received.
	subCtx, cancel := context.WithTimeout(mgr.routineCtx, mgr.opTimeout)
	defer cancel()

	if paid.Cmp(big.NewInt(0)) == 0 {
		// It has been paid.
		state.index = index
		state.paymentRequired = big.NewInt(0)
		state.expiredAt = time.Now().Add(state.inactivity)
		log.Debugf("Request resume for %v-%v", p, request.ID())
		hookActions.UnpauseResponse()
		return
	}
	payment, err := mgr.payProto.Receive(subCtx, state.currencyID, state.fromAddr)
	if err == nil {
		if payment.Cmp(state.paymentRequired) < 0 {
			hookActions.TerminateWithError(fmt.Errorf("payment not enough, expect %v got %v", state.paymentRequired, payment))
			log.Debugf("Payment not match, expect %v got %v", paid, payment)
			return
		}
	}
	// Either error in our end, continue or payment is successful.
	state.paymentRequired = big.NewInt(0)
	state.expiredAt = time.Now().Add(state.inactivity)
	state.index = index
	log.Debugf("Request resume for %v-%v", p, request.ID())
	hookActions.UnpauseResponse()
}

// onComplete is the hook called when retrieval is completed.
func (mgr *RetrievalManager) onComplete(p peer.ID, request gs.RequestData, status gs.ResponseStatusCode) {
	// Get offer ID
	offerData, ok := request.Extension(offerExtension)
	if !ok {
		log.Warnf("Fail to get offer extention in incoming request")
		return
	}
	offerID, err := getOfferIDRaw(offerData)
	if err != nil {
		log.Errorf("fail to get offer id for %v: %v", offerData, err.Error())
		return
	}
	if status.IsFailure() {
		mgr.activeInLock.RLock()
		release := mgr.activeInLock.RUnlock
		defer release()

		state, ok := mgr.activeInOfferID[offerID]
		if !ok {
			log.Errorf("Expect to find state for %v", offerID)
			return
		}
		go func() {
			rec := peermgr.Record{
				Description: fmt.Sprintf("peer failed to complete retriaval of %v", request.Root()),
				CreatedAt:   time.Now(),
			}
			err := mgr.peerMgr.AddToHistory(mgr.routineCtx, state.currencyID, state.fromAddr, rec)
			if err != nil {
				log.Warnf("Fail add %v to history of %v-%v: %v", rec, state.currencyID, state.fromAddr, err.Error())
			}
		}()

		state.lock.Lock()
		defer state.lock.Unlock()

		state.processed = false
	} else {
		mgr.activeInLock.Lock()
		release := mgr.activeInLock.Unlock
		defer release()

		state, ok := mgr.activeInOfferID[offerID]
		if ok {
			go func() {
				rec := peermgr.Record{
					Description: fmt.Sprintf("peer succeed completing retriaval of %v", request.Root()),
					CreatedAt:   time.Now(),
				}
				err := mgr.peerMgr.AddToHistory(mgr.routineCtx, state.currencyID, state.fromAddr, rec)
				if err != nil {
					log.Warnf("Fail add %v to history of %v-%v: %v", rec, state.currencyID, state.fromAddr, err.Error())
				}
			}()
		}
		mgr.cleanActiveIn()
		delete(mgr.activeInOfferID, offerID)
	}
	log.Infof("Retrieval process for ID %v for piece %v completed with status: %s", request.ID(), request.Root(), status.String())
}

// OnRequestorCancelledListener is the hook called when retrieval is cancelled.
func (mgr *RetrievalManager) OnRequestorCancelledListener(p peer.ID, request gs.RequestData) {
	log.Debugf("Request cancelled for %v-%v", p, request.ID())
	// Get offer ID
	offerData, ok := request.Extension(offerExtension)
	if !ok {
		log.Warnf("Fail to get offer extention in incoming request")
		return
	}
	offerID, err := getOfferIDRaw(offerData)
	if err != nil {
		log.Errorf("fail to get offer id for %v: %v", offerData, err.Error())
		return
	}
	mgr.activeInLock.RLock()
	release := mgr.activeInLock.RUnlock
	defer release()

	state, ok := mgr.activeInOfferID[offerID]
	if !ok {
		log.Errorf("Expect to find state for %v", offerID)
		return
	}
	go func() {
		rec := peermgr.Record{
			Description: fmt.Sprintf("peer cancelled retriaval of %v", request.Root()),
			CreatedAt:   time.Now(),
		}
		err := mgr.peerMgr.AddToHistory(mgr.routineCtx, state.currencyID, state.fromAddr, rec)
		if err != nil {
			log.Warnf("Fail add %v to history of %v-%v: %v", rec, state.currencyID, state.fromAddr, err.Error())
		}
	}()

	state.lock.Lock()
	defer state.lock.Unlock()

	state.processed = false
}

// OnNetworkErrorListener is the hook called when retrieval has network error.
func (mgr *RetrievalManager) OnNetworkErrorListener(p peer.ID, request gs.RequestData, err error) {
	log.Debugf("Request has network error for %v-%v", p, request.ID())
	// Get offer ID
	offerData, ok := request.Extension(offerExtension)
	if !ok {
		log.Warnf("Fail to get offer extention in incoming request")
		return
	}
	offerID, err := getOfferIDRaw(offerData)
	if err != nil {
		log.Errorf("fail to get offer id for %v: %v", offerData, err.Error())
		return
	}
	mgr.activeInLock.RLock()
	release := mgr.activeInLock.RUnlock
	defer release()

	state, ok := mgr.activeInOfferID[offerID]
	if !ok {
		log.Errorf("Expect to find state for %v", offerID)
		return
	}
	go func() {
		rec := peermgr.Record{
			Description: fmt.Sprintf("peer has network error to complete retriaval of %v", request.Root()),
			CreatedAt:   time.Now(),
		}
		err := mgr.peerMgr.AddToHistory(mgr.routineCtx, state.currencyID, state.fromAddr, rec)
		if err != nil {
			log.Warnf("Fail add %v to history of %v-%v: %v", rec, state.currencyID, state.fromAddr, err.Error())
		}
	}()

	state.lock.Lock()
	defer state.lock.Unlock()

	state.processed = false
}
