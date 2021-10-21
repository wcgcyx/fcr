package cidnetmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	gs "github.com/ipfs/go-graphsync"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wcgcyx/fcr/internal/cidoffer"
	"github.com/wcgcyx/fcr/internal/crypto"
)

// channelIn is the information of a incoming retrieval channel.
type channelIn struct {
	currencyID      uint64
	fromAddr        string
	ppb             *big.Int
	paymentRequired *big.Int
	lock            sync.RWMutex
}

// onIncomingRequest is the hook called on incoming request.
// It takes a peer id, a request and hook actions as arguments.
func (mgr *CIDNetworkManagerImplV1) onIncomingRequest(p peer.ID, request gs.RequestData, hookActions gs.IncomingRequestHookActions) {
	mgr.retrievalInChsLock.RLock()
	_, ok := mgr.retrievalInChs[fmt.Sprintf("%v-%v", p.String(), request.ID())]
	mgr.retrievalInChsLock.RUnlock()
	if ok {
		hookActions.TerminateWithError(fmt.Errorf("there is an existing retrieval process"))
		return
	}

	// Get offer
	data, ok := request.Extension(cidOfferExtension)
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("no offer attached"))
		return
	}
	offer, err := cidoffer.FromBytes(data)
	if err != nil {
		hookActions.TerminateWithError(fmt.Errorf("fail to parse offer"))
		return
	}

	// Verify offer
	root, err := mgr.paynetMgr.GetRootAddress(offer.CurrencyID())
	if err != nil {
		hookActions.TerminateWithError(fmt.Errorf("offer contains invalid currency id"))
		return
	}
	if offer.ToAddr() != root {
		hookActions.TerminateWithError(fmt.Errorf("offer contains invalid recipient"))
		return
	}
	// Check offer root CID
	if !offer.Root().Equals(request.Root()) {
		hookActions.TerminateWithError(fmt.Errorf("offer root does not match request root"))
		return
	}
	if offer.Verify() != nil {
		hookActions.TerminateWithError(fmt.Errorf("offer fails to verify"))
		return
	}
	// Check offer expiration time
	if time.Now().Unix() > offer.Expiration() {
		hookActions.TerminateWithError(fmt.Errorf("offer expired"))
		return
	}

	// Get from addr
	data, ok = request.Extension(fromExtension)
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("no from address attached"))
		return
	}
	fromAddr := string(data)

	// Get counter signature
	data, ok = request.Extension(signatureExtension)
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("no counter signature attached"))
		return
	}
	// Verify counter signature
	if crypto.Verify(fromAddr, data, offer.ToBytes()) != nil {
		hookActions.TerminateWithError(fmt.Errorf("counter signature fails to verify"))
		return
	}

	// Update active map
	mgr.activeLock.Lock()
	defer mgr.activeLock.Unlock()
	count, ok := mgr.active[offer.Root().String()]
	if !ok {
		count = 1
	} else {
		count += 1
	}
	mgr.active[offer.Root().String()] = count

	// Update retrieval channel map
	mgr.retrievalInChsLock.Lock()
	defer mgr.retrievalInChsLock.Unlock()
	mgr.retrievalInChs[fmt.Sprintf("%v-%v", p.String(), request.ID())] = &channelIn{
		currencyID:      offer.CurrencyID(),
		fromAddr:        fromAddr,
		ppb:             offer.PPB(),
		paymentRequired: big.NewInt(0),
		lock:            sync.RWMutex{},
	}
	hookActions.ValidateRequest()
}

// receiptJson is a receipt to verify the payment.
type receiptJson struct {
	Secret  string `json:"secret"`
	Receipt string `json:"receipt"`
}

// onUpdatedRequest is the hook called on updated request.
// It takes a peer id, a request, updated request and hook actions as arguments.
func (mgr *CIDNetworkManagerImplV1) onUpdatedRequest(p peer.ID, request gs.RequestData, updateRequest gs.RequestData, hookActions gs.RequestUpdatedHookActions) {
	mgr.retrievalInChsLock.RLock()
	ch, ok := mgr.retrievalInChs[fmt.Sprintf("%v-%v", p.String(), request.ID())]
	mgr.retrievalInChsLock.RUnlock()
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("there is not existing retrieval process"))
		return
	}
	ch.lock.RLock()
	defer ch.lock.RUnlock()
	_, ok = updateRequest.Extension(paymentExtension)
	if !ok {
		mgr.retrievalInChsLock.Lock()
		defer mgr.retrievalInChsLock.Unlock()
		delete(mgr.retrievalInChs, fmt.Sprintf("%v-%v", p.String(), request.ID()))
		hookActions.TerminateWithError(fmt.Errorf("expect payment"))
		return
	}
	// Payment required.
	payment, err := mgr.paynetMgr.Receive(ch.currencyID, ch.fromAddr)
	if err != nil {
		mgr.retrievalInChsLock.Lock()
		defer mgr.retrievalInChsLock.Unlock()
		delete(mgr.retrievalInChs, fmt.Sprintf("%v-%v", p.String(), request.ID()))
		hookActions.TerminateWithError(fmt.Errorf("error in receiving payment"))
		return
	}
	if ch.paymentRequired.Cmp(payment) > 0 {
		mgr.retrievalInChsLock.Lock()
		defer mgr.retrievalInChsLock.Unlock()
		delete(mgr.retrievalInChs, fmt.Sprintf("%v-%v", p.String(), request.ID()))
		hookActions.TerminateWithError(fmt.Errorf("received insufficient payment"))
		return
	}
	hookActions.UnpauseResponse()
}

// onOutgoingBlock is the hook called on outgoing block.
// It takes a peer id, a request, block data and hook actions as arguments.
func (mgr *CIDNetworkManagerImplV1) onOutgoingBlock(p peer.ID, request gs.RequestData, block gs.BlockData, hookActions gs.OutgoingBlockHookActions) {
	mgr.retrievalInChsLock.RLock()
	ch, ok := mgr.retrievalInChs[fmt.Sprintf("%v-%v", p.String(), request.ID())]
	mgr.retrievalInChsLock.RUnlock()
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("there is not existing retrieval process"))
		return
	}
	ch.lock.Lock()
	defer ch.lock.Unlock()
	ch.paymentRequired.Mul(big.NewInt(int64(block.BlockSizeOnWire())), ch.ppb)
	hookActions.SendExtensionData(gs.ExtensionData{Name: paymentExtension, Data: ch.paymentRequired.Bytes()})
	hookActions.PauseResponse()
}

// onComplete is the hook called when retrieval is completed.
// It takes a peer id, a request and status as arguments.
func (mgr *CIDNetworkManagerImplV1) onComplete(p peer.ID, request gs.RequestData, status gs.ResponseStatusCode) {
	// Update active map
	mgr.activeLock.Lock()
	count, ok := mgr.active[request.Root().String()]
	if !ok {
		log.Warnf("Completeing retrieval process on non-exsting active serving")
	} else {
		count--
		if count == 0 {
			delete(mgr.active, request.Root().String())
		} else {
			mgr.active[request.Root().String()] = count
		}
	}
	mgr.activeLock.Unlock()

	// Update retrieval channel map
	mgr.retrievalInChsLock.Lock()
	defer mgr.retrievalInChsLock.Unlock()
	delete(mgr.retrievalInChs, fmt.Sprintf("%v-%v", p.String(), request.ID()))

	log.Infof("Retrieval process for ID: %v completed with status: %s", request.ID(), status.String())
}
