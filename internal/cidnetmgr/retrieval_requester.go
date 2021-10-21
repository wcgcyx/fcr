package cidnetmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"fmt"
	"math/big"

	gs "github.com/ipfs/go-graphsync"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wcgcyx/fcr/internal/cidoffer"
	"github.com/wcgcyx/fcr/internal/payoffer"
)

// channelOut is the information of an outgoing retrieval channel.
type channelOut struct {
	cidOffer *cidoffer.CIDOffer
	payOffer *payoffer.PayOffer
}

// onOutgoingRequest is the hook called on outgoing request.
func (mgr *CIDNetworkManagerImplV1) onOutgoingRequest(p peer.ID, request gs.RequestData, hookAction gs.OutgoingRequestHookActions) {
	data, _ := request.Extension(cidOfferExtension)
	cidOffer, _ := cidoffer.FromBytes(data)
	data, _ = request.Extension(payOfferExtension)
	var payOffer *payoffer.PayOffer
	if len(data) > 0 {
		payOffer, _ = payoffer.FromBytes(data)
	}
	// Add to the map
	mgr.retrievalOutChsLock.Lock()
	defer mgr.retrievalOutChsLock.Unlock()
	mgr.retrievalOutID[request.Root().String()] = fmt.Sprintf("%v-%v", p.String(), request.ID())
	mgr.retrievalOutChs[fmt.Sprintf("%v-%v", p.String(), request.ID())] = &channelOut{
		cidOffer: cidOffer,
		payOffer: payOffer,
	}
}

// onIncomingBlock is the hook called on incoming block.
func (mgr *CIDNetworkManagerImplV1) onIncomingBlock(p peer.ID, responseData gs.ResponseData, blockData gs.BlockData, hookActions gs.IncomingBlockHookActions) {
	if blockData.BlockSizeOnWire() == 0 {
		// This block is existed locally, no payment required.
		return
	}
	data, ok := responseData.Extension(paymentExtension)
	if ok {
		requested := big.NewInt(0).SetBytes(data)
		mgr.retrievalOutChsLock.RLock()
		defer mgr.retrievalOutChsLock.RUnlock()
		// Get offer price from it
		ch := mgr.retrievalOutChs[fmt.Sprintf("%v-%v", p.String(), responseData.RequestID())]
		ppb := ch.cidOffer.PPB()
		// Get required amount
		required := big.NewInt(0).Mul(big.NewInt(int64(blockData.BlockSizeOnWire())), ppb)
		if required.Cmp(requested) < 0 {
			// Invalid amount.
			hookActions.TerminateWithError(fmt.Errorf("received invalid payment requested"))
			return
		}
		// Pay required amount.
		if ch.payOffer == nil {
			err := mgr.paynetMgr.Pay(mgr.ctx, ch.cidOffer.CurrencyID(), ch.cidOffer.ToAddr(), requested)
			if err != nil {
				hookActions.TerminateWithError(fmt.Errorf("error in payment %v", err.Error()))
				return
			}
		} else {
			err := mgr.paynetMgr.PayWithOffer(mgr.ctx, ch.payOffer, requested)
			if err != nil {
				hookActions.TerminateWithError(fmt.Errorf("error in payment with offer %v", err.Error()))
				return
			}
		}
		pay := required
		hookActions.UpdateRequestWithExtensions(gs.ExtensionData{Name: paymentExtension, Data: pay.Bytes()})
	}
}
