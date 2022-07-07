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
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	gs "github.com/ipfs/go-graphsync"
	"github.com/libp2p/go-libp2p-core/peer"
)

// onOutgoingRequest is the hook called on outgoing request.
func (mgr *RetrievalManager) onOutgoingRequest(p peer.ID, request gs.RequestData, hookActions gs.OutgoingRequestHookActions) {
	log.Debugf("On outgoing request to peer %v: %v", p, request.ID())
	offerData, ok := request.Extension(offerExtension)
	if !ok {
		// Should never happen, will throw error in the next stage.
		log.Errorf("Fail to get offer extension in outgoing request, should never happe")
		return
	}
	offerID, err := getOfferIDRaw(offerData)
	if err != nil {
		// Should never happen, it will throw error in the next stage.
		log.Errorf("Fail to get offer ID, should never happen: %v", err.Error())
		return
	}
	// Create a new map for the req id.
	reqID := fmt.Sprintf("%v-%v", p.String(), request.ID())
	mgr.activeOutLock.Lock()
	defer mgr.activeOutLock.Unlock()

	mgr.cleanActiveOut()
	state, ok := mgr.activeOutOfferID[offerID]
	if !ok {
		// Should never happen, it will throw in the next stage.
		log.Errorf("Fail to get offer state for %v, should never happen", offerID)
		return
	}
	mgr.activeOutReqID[reqID] = state
}

// onIncomingBlock is the hook called on incoming block.
func (mgr *RetrievalManager) onIncomingBlock(p peer.ID, responseData gs.ResponseData, blockData gs.BlockData, hookActions gs.IncomingBlockHookActions) {
	log.Debugf("Onincoming block: index %v, size %v, size on wire %v", blockData.Index(), blockData.BlockSize(), blockData.BlockSizeOnWire())
	reqID := fmt.Sprintf("%v-%v", p.String(), responseData.RequestID())
	mgr.activeOutLock.RLock()
	defer mgr.activeOutLock.RUnlock()

	state, ok := mgr.activeOutReqID[reqID]
	if !ok {
		hookActions.TerminateWithError(fmt.Errorf("req id %v does not exist", reqID))
		log.Debugf("Req id %v does not exist", reqID)
		return
	}

	state.lock.Lock()
	defer state.lock.Unlock()

	if state.ppb.Cmp(big.NewInt(0)) > 0 {
		// Need to send payment info.
		log.Debugf("Need to send payment info")
		paymentRequired := big.NewInt(0).Mul(big.NewInt(int64(blockData.BlockSizeOnWire())), state.ppb)
		if paymentRequired.Cmp(big.NewInt(0)) > 0 {
			// Need to make a payment.
			if state.payOffer == nil {
				err := mgr.payProto.PayForSelf(state.ctx, state.currencyID, state.toAddr, state.resCh, state.resID, paymentRequired)
				if err != nil {
					hookActions.TerminateWithError(fmt.Errorf("req id %v fail to do payment of %v: %v", reqID, paymentRequired, err.Error()))
					log.Warnf("Fail to pay for self: %v", err.Error())
					return
				}
			} else {
				err := mgr.payProto.PayForSelfWithOffer(state.ctx, *state.payOffer, state.resCh, state.resID, paymentRequired)
				if err != nil {
					hookActions.TerminateWithError(fmt.Errorf("req id %v fail to do proxy payment of %v: %v", reqID, paymentRequired, err.Error()))
					log.Warnf("Fail to pay for self with offer: %v", err.Error())
					return
				}
			}
		}
		// Send extension.
		index := make([]byte, 8)
		binary.LittleEndian.PutUint64(index, uint64(blockData.Index()))
		hookActions.UpdateRequestWithExtensions(gs.ExtensionData{Name: paymentExtension, Data: paymentRequired.Bytes()}, gs.ExtensionData{Name: paymentIndexExtension, Data: index})
	}
	// In case peer restarted, double the extension.
	state.expiredAt = time.Now().Add(2 * state.inactivity)
}
