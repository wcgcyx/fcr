package api

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
	"runtime"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/mpstore"
	"github.com/wcgcyx/fcr/node"
	"github.com/wcgcyx/fcr/paychstate"
	"github.com/wcgcyx/fcr/peermgr"
)

// userAPIHandler is used to handle user API.
type userAPIHandler struct {
	node *node.Node
}

// Network API
func (h *userAPIHandler) NetAddr(ctx context.Context) (peer.AddrInfo, error) {
	return peer.AddrInfo{ID: h.node.Host.ID(), Addrs: h.node.Host.Addrs()}, nil
}

func (h *userAPIHandler) NetConnect(ctx context.Context, pi peer.AddrInfo) error {
	return h.node.Host.Connect(ctx, pi)
}

// Signer API
func (h *userAPIHandler) SignerSetKey(ctx context.Context, currencyID byte, keyType byte, prv []byte) error {
	return h.node.Signer.SetKey(ctx, currencyID, keyType, prv)
}

func (h *userAPIHandler) SignerGetAddr(ctx context.Context, currencyID byte) (SignerGetAddrRes, error) {
	keyType, addr, err := h.node.Signer.GetAddr(ctx, currencyID)
	return SignerGetAddrRes{KeyType: keyType, Addr: addr}, err
}

func (h *userAPIHandler) SignerRetireKey(ctx context.Context, currencyID byte, timeout time.Duration) <-chan string {
	res := make(chan string, 1)
	go func() {
		defer close(res)
		err := <-h.node.Signer.RetireKey(ctx, currencyID, timeout)
		if err != nil {
			res <- err.Error()
		} else {
			res <- ""
		}
	}()
	return res
}

func (h *userAPIHandler) SignerStopRetire(ctx context.Context, currencyID byte) error {
	return h.node.Signer.StopRetire(ctx, currencyID)
}

func (h *userAPIHandler) SignerListCurrencyIDs(ctx context.Context) <-chan byte {
	curChan, errChan := h.node.Signer.ListCurrencyIDs(ctx)
	res := make(chan byte, 32)
	go func() {
		defer close(res)
		for currencyID := range curChan {
			res <- currencyID
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list currency ids: %v", err.Error())
		}
	}()
	return res
}

// Peer Manager API
func (h *userAPIHandler) PeerMgrAddPeer(ctx context.Context, currencyID byte, toAddr string, pi peer.AddrInfo) error {
	return h.node.PeerMgr.AddPeer(ctx, currencyID, toAddr, pi)
}

func (h *userAPIHandler) PeerMgrHasPeer(ctx context.Context, currencyID byte, toAddr string) (bool, error) {
	return h.node.PeerMgr.HasPeer(ctx, currencyID, toAddr)
}

func (h *userAPIHandler) PeerMgrRemovePeer(ctx context.Context, currencyID byte, toAddr string) error {
	return h.node.PeerMgr.RemovePeer(ctx, currencyID, toAddr)
}

func (h *userAPIHandler) PeerMgrPeerAddr(ctx context.Context, currencyID byte, toAddr string) (peer.AddrInfo, error) {
	return h.node.PeerMgr.PeerAddr(ctx, currencyID, toAddr)
}

func (h *userAPIHandler) PeerMgrIsBlocked(ctx context.Context, currencyID byte, toAddr string) (bool, error) {
	return h.node.PeerMgr.IsBlocked(ctx, currencyID, toAddr)
}

func (h *userAPIHandler) PeerMgrBlockPeer(ctx context.Context, currencyID byte, toAddr string) error {
	return h.node.PeerMgr.BlockPeer(ctx, currencyID, toAddr)
}

func (h *userAPIHandler) PeerMgrUnblockPeer(ctx context.Context, currencyID byte, toAddr string) error {
	return h.node.PeerMgr.UnblockPeer(ctx, currencyID, toAddr)
}

func (h *userAPIHandler) PeerMgrListCurrencyIDs(ctx context.Context) <-chan byte {
	curChan, errChan := h.node.PeerMgr.ListCurrencyIDs(ctx)
	res := make(chan byte, 32)
	go func() {
		defer close(res)
		for currencyID := range curChan {
			res <- currencyID
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list currency ids: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) PeerMgrListPeers(ctx context.Context, currencyID byte) <-chan string {
	peerChan, errChan := h.node.PeerMgr.ListPeers(ctx, currencyID)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for peer := range peerChan {
			res <- peer
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list peers: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) PeerMgrRemoveRecord(ctx context.Context, currencyID byte, toAddr string, recID *big.Int) error {
	return h.node.PeerMgr.RemoveRecord(ctx, currencyID, toAddr, recID)
}

func (h *userAPIHandler) PeerMgrSetRecID(ctx context.Context, currencyID byte, toAddr string, recID *big.Int) error {
	return h.node.PeerMgr.SetRecID(ctx, currencyID, toAddr, recID)
}

func (h *userAPIHandler) PeerMgrListHistory(ctx context.Context, currencyID byte, toAddr string) <-chan PeerMgrListHistoryRes {
	recIDChan, recordChain, errChan := h.node.PeerMgr.ListHistory(ctx, currencyID, toAddr)
	res := make(chan PeerMgrListHistoryRes, 32)
	go func() {
		defer close(res)
		for recID := range recIDChan {
			record := <-recordChain
			res <- PeerMgrListHistoryRes{
				RecID:  recID,
				Record: record,
			}
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list history: %v", err.Error())
		}
	}()
	return res
}

// Transactor API
func (h *userAPIHandler) TransactorCreate(ctx context.Context, currencyID byte, toAddr string, amt *big.Int) (string, error) {
	return h.node.Transactor.Create(ctx, currencyID, toAddr, amt)
}

func (h *userAPIHandler) TransactorTopup(ctx context.Context, currencyID byte, chAddr string, amt *big.Int) error {
	return h.node.Transactor.Topup(ctx, currencyID, chAddr, amt)
}

func (h *userAPIHandler) TransactorCheck(ctx context.Context, currencyID byte, chAddr string) (TransactorCheckRes, error) {
	settlingAt, minSettlingAt, currentHeight, redeemed, balance, sender, recipient, err := h.node.Transactor.Check(ctx, currencyID, chAddr)
	return TransactorCheckRes{SettlingAt: settlingAt, MinSettlingAt: minSettlingAt, CurrentHeight: currentHeight, Redeemed: redeemed, Balance: balance, Sender: sender, Recipient: recipient}, err
}

func (h *userAPIHandler) TransactorUpdate(ctx context.Context, currencyID byte, chAddr string, voucher string) error {
	return h.node.Transactor.Update(ctx, currencyID, chAddr, voucher)
}

func (h *userAPIHandler) TransactorSettle(ctx context.Context, currencyID byte, chAddr string) error {
	return h.node.Transactor.Settle(ctx, currencyID, chAddr)
}

func (h *userAPIHandler) TransactorCollect(ctx context.Context, currencyID byte, chAddr string) error {
	return h.node.Transactor.Collect(ctx, currencyID, chAddr)
}

func (h *userAPIHandler) TransactorVerifyVoucher(currencyID byte, voucher string) (TransactorVerifyVoucherRes, error) {
	sender, chAddr, lane, nonce, redeemed, err := h.node.Transactor.VerifyVoucher(currencyID, voucher)
	return TransactorVerifyVoucherRes{Sender: sender, ChAddr: chAddr, Lane: lane, Nonce: nonce, Redeemed: redeemed}, err
}

func (h *userAPIHandler) TransactorGetHeight(ctx context.Context, currencyID byte) (int64, error) {
	return h.node.Transactor.GetHeight(ctx, currencyID)
}

func (h *userAPIHandler) TransactorGetBalance(ctx context.Context, currencyID byte, addr string) (*big.Int, error) {
	return h.node.Transactor.GetBalance(ctx, currencyID, addr)
}

// Active out payment channel store API
func (h *userAPIHandler) ActiveOutRead(ctx context.Context, currencyID byte, peerAddr string, chAddr string) (paychstate.State, error) {
	return h.node.ActiveOutPaych.Read(ctx, currencyID, peerAddr, chAddr)
}

func (h *userAPIHandler) ActiveOutListCurrencyIDs(ctx context.Context) <-chan byte {
	curChan, errChan := h.node.ActiveOutPaych.ListCurrencyIDs(ctx)
	res := make(chan byte, 32)
	go func() {
		defer close(res)
		for currencyID := range curChan {
			res <- currencyID
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list currency ids: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) ActiveOutListPeers(ctx context.Context, currencyID byte) <-chan string {
	peerChan, errChan := h.node.ActiveOutPaych.ListPeers(ctx, currencyID)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for peer := range peerChan {
			res <- peer
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list peers: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) ActiveOutListPaychsByPeer(ctx context.Context, currencyID byte, peerAddr string) <-chan string {
	paychChan, errChan := h.node.ActiveOutPaych.ListPaychsByPeer(ctx, currencyID, peerAddr)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for paych := range paychChan {
			res <- paych
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list paychs: %v", err.Error())
		}
	}()
	return res
}

// Inactive out payment channel store API
func (h *userAPIHandler) InactiveOutRemove(currencyID byte, peerAddr string, chAddr string) {
	h.node.InactiveOutPaych.Remove(currencyID, peerAddr, chAddr)
}

func (h *userAPIHandler) InactiveOutRead(ctx context.Context, currencyID byte, peerAddr string, chAddr string) (paychstate.State, error) {
	return h.node.InactiveOutPaych.Read(ctx, currencyID, peerAddr, chAddr)
}

func (h *userAPIHandler) InactiveOutListCurrencyIDs(ctx context.Context) <-chan byte {
	curChan, errChan := h.node.InactiveOutPaych.ListCurrencyIDs(ctx)
	res := make(chan byte, 32)
	go func() {
		defer close(res)
		for currencyID := range curChan {
			res <- currencyID
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list currency ids: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) InactiveOutListPeers(ctx context.Context, currencyID byte) <-chan string {
	peerChan, errChan := h.node.InactiveOutPaych.ListPeers(ctx, currencyID)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for peer := range peerChan {
			res <- peer
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list peers: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) InactiveOutListPaychsByPeer(ctx context.Context, currencyID byte, peerAddr string) <-chan string {
	paychChan, errChan := h.node.InactiveOutPaych.ListPaychsByPeer(ctx, currencyID, peerAddr)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for paych := range paychChan {
			res <- paych
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list paychs: %v", err.Error())
		}
	}()
	return res
}

// Active in payment channel store API
func (h *userAPIHandler) ActiveInRead(ctx context.Context, currencyID byte, peerAddr string, chAddr string) (paychstate.State, error) {
	return h.node.ActiveInPaych.Read(ctx, currencyID, peerAddr, chAddr)
}

func (h *userAPIHandler) ActiveInListCurrencyIDs(ctx context.Context) <-chan byte {
	curChan, errChan := h.node.ActiveInPaych.ListCurrencyIDs(ctx)
	res := make(chan byte, 32)
	go func() {
		defer close(res)
		for currencyID := range curChan {
			res <- currencyID
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list currency ids: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) ActiveInListPeers(ctx context.Context, currencyID byte) <-chan string {
	peerChan, errChan := h.node.ActiveInPaych.ListPeers(ctx, currencyID)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for peer := range peerChan {
			res <- peer
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list peers: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) ActiveInListPaychsByPeer(ctx context.Context, currencyID byte, peerAddr string) <-chan string {
	paychChan, errChan := h.node.ActiveInPaych.ListPaychsByPeer(ctx, currencyID, peerAddr)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for paych := range paychChan {
			res <- paych
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list paychs: %v", err.Error())
		}
	}()
	return res
}

// Inactive in payment channel store API
func (h *userAPIHandler) InactiveInRemove(currencyID byte, peerAddr string, chAddr string) {
	h.node.InactiveInPaych.Remove(currencyID, peerAddr, chAddr)
}

func (h *userAPIHandler) InactiveInRead(ctx context.Context, currencyID byte, peerAddr string, chAddr string) (paychstate.State, error) {
	return h.node.InactiveInPaych.Read(ctx, currencyID, peerAddr, chAddr)
}

func (h *userAPIHandler) InactiveInListCurrencyIDs(ctx context.Context) <-chan byte {
	curChan, errChan := h.node.InactiveInPaych.ListCurrencyIDs(ctx)
	res := make(chan byte, 32)
	go func() {
		defer close(res)
		for currencyID := range curChan {
			res <- currencyID
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list currency ids: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) InactiveInListPeers(ctx context.Context, currencyID byte) <-chan string {
	peerChan, errChan := h.node.InactiveInPaych.ListPeers(ctx, currencyID)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for peer := range peerChan {
			res <- peer
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list peers: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) InactiveInListPaychsByPeer(ctx context.Context, currencyID byte, peerAddr string) <-chan string {
	paychChan, errChan := h.node.InactiveInPaych.ListPaychsByPeer(ctx, currencyID, peerAddr)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for paych := range paychChan {
			res <- paych
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list paychs: %v", err.Error())
		}
	}()
	return res
}

// Payment serving manager API
func (h *userAPIHandler) PservMgrServe(ctx context.Context, currencyID byte, toAddr string, chAddr string, ppp *big.Int, period *big.Int) error {
	return h.node.PServMgr.Serve(ctx, currencyID, toAddr, chAddr, ppp, period)
}

func (h *userAPIHandler) PservMgrStop(ctx context.Context, currencyID byte, toAddr string, chAddr string) error {
	return h.node.PServMgr.Stop(ctx, currencyID, toAddr, chAddr)
}

func (h *userAPIHandler) PservMgrInspect(ctx context.Context, currencyID byte, toAddr string, chAddr string) (PservMgrInspectRes, error) {
	served, ppp, period, err := h.node.PServMgr.Inspect(ctx, currencyID, toAddr, chAddr)
	return PservMgrInspectRes{Served: served, PPP: ppp, Period: period}, err
}

func (h *userAPIHandler) PservMgrListCurrencyIDs(ctx context.Context) <-chan byte {
	curChan, errChan := h.node.PServMgr.ListCurrencyIDs(ctx)
	res := make(chan byte, 32)
	go func() {
		defer close(res)
		for currencyID := range curChan {
			res <- currencyID
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list currency ids: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) PservMgrListRecipients(ctx context.Context, currencyID byte) <-chan string {
	peerChan, errChan := h.node.PServMgr.ListRecipients(ctx, currencyID)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for peer := range peerChan {
			res <- peer
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list peers: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) PservMgrListServings(ctx context.Context, currencyID byte, toAddr string) <-chan string {
	paychChan, errChan := h.node.PServMgr.ListServings(ctx, currencyID, toAddr)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for paych := range paychChan {
			res <- paych
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list paychs: %v", err.Error())
		}
	}()
	return res
}

// Route store API
func (h *userAPIHandler) RouteStoreMaxHop(ctx context.Context, currencyID byte) (uint64, error) {
	return h.node.RouteStore.MaxHop(ctx, currencyID)
}

func (h *userAPIHandler) RouteStoreListRoutesTo(ctx context.Context, currencyID byte, toAddr string) <-chan []string {
	routeChan, errChan := h.node.RouteStore.ListRoutesTo(ctx, currencyID, toAddr)
	res := make(chan []string, 32)
	go func() {
		defer close(res)
		for peer := range routeChan {
			res <- peer
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list routes: %v", err.Error())
		}
	}()
	return res
}

// Payment manager API
func (h *userAPIHandler) PayMgrReserveForSelf(ctx context.Context, currencyID byte, toAddr string, pettyAmt *big.Int, expiration time.Time, inactivity time.Duration) (PayMgrReserveForSelfRes, error) {
	resCh, resID, err := h.node.PayMgr.ReserveForSelf(ctx, currencyID, toAddr, pettyAmt, expiration, inactivity)
	return PayMgrReserveForSelfRes{ResCh: resCh, ResID: resID}, err
}

func (h *userAPIHandler) PayMgrReserveForSelfWithOffer(ctx context.Context, receivedOffer fcroffer.PayOffer) (PayMgrReserveForSelfWithOfferRes, error) {
	resCh, resID, err := h.node.PayMgr.ReserveForSelfWithOffer(ctx, receivedOffer)
	return PayMgrReserveForSelfWithOfferRes{ResCh: resCh, ResID: resID}, err
}

func (h *userAPIHandler) PayMgrUpdateOutboundChannelBalance(ctx context.Context, currencyID byte, toAddr string, chAddr string) error {
	return h.node.PayMgr.UpdateOutboundChannelBalance(ctx, currencyID, toAddr, chAddr)
}

func (h *userAPIHandler) PayMgrBearNetworkLoss(ctx context.Context, currencyID byte, toAddr string, chAddr string) error {
	return h.node.PayMgr.BearNetworkLoss(ctx, currencyID, toAddr, chAddr)
}

// Settlement manager API
func (h *userAPIHandler) SettleMgrSetDefaultPolicy(ctx context.Context, currencyID byte, duration time.Duration) error {
	return h.node.SettleMgr.SetDefaultPolicy(ctx, currencyID, duration)
}

func (h *userAPIHandler) SettleMgrGetDefaultPolicy(ctx context.Context, currencyID byte) (time.Duration, error) {
	return h.node.SettleMgr.GetDefaultPolicy(ctx, currencyID)
}

func (h *userAPIHandler) SettleMgrRemoveDefaultPolicy(ctx context.Context, currencyID byte) error {
	return h.node.SettleMgr.RemoveDefaultPolicy(ctx, currencyID)
}

func (h *userAPIHandler) SettleMgrSetSenderPolicy(ctx context.Context, currencyID byte, fromAddr string, duration time.Duration) error {
	return h.node.SettleMgr.SetSenderPolicy(ctx, currencyID, fromAddr, duration)
}

func (h *userAPIHandler) SettleMgrGetSenderPolicy(ctx context.Context, currencyID byte, fromAddr string) (time.Duration, error) {
	return h.node.SettleMgr.GetSenderPolicy(ctx, currencyID, fromAddr)
}

func (h *userAPIHandler) SettleMgrRemoveSenderPolicy(ctx context.Context, currencyID byte, fromAddr string) error {
	return h.node.SettleMgr.RemoveSenderPolicy(ctx, currencyID, fromAddr)
}

func (h *userAPIHandler) SettleMgrListCurrencyIDs(ctx context.Context) <-chan byte {
	curChan, errChan := h.node.SettleMgr.ListCurrencyIDs(ctx)
	res := make(chan byte, 32)
	go func() {
		defer close(res)
		for currencyID := range curChan {
			res <- currencyID
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list currency ids: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) SettleMgrListSenders(ctx context.Context, currencyID byte) <-chan string {
	peerChan, errChan := h.node.SettleMgr.ListSenders(ctx, currencyID)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for peer := range peerChan {
			res <- peer
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list peers: %v", err.Error())
		}
	}()
	return res
}

// Renew manager API
func (h *userAPIHandler) RenewMgrSetDefaultPolicy(ctx context.Context, currencyID byte, duration time.Duration) error {
	return h.node.RenewMgr.SetDefaultPolicy(ctx, currencyID, duration)
}

func (h *userAPIHandler) RenewMgrGetDefaultPolicy(ctx context.Context, currencyID byte) (time.Duration, error) {
	return h.node.RenewMgr.GetDefaultPolicy(ctx, currencyID)
}

func (h *userAPIHandler) RenewMgrRemoveDefaultPolicy(ctx context.Context, currencyID byte) error {
	return h.node.RenewMgr.RemoveDefaultPolicy(ctx, currencyID)
}

func (h *userAPIHandler) RenewMgrSetSenderPolicy(ctx context.Context, currencyID byte, fromAddr string, duration time.Duration) error {
	return h.node.RenewMgr.SetSenderPolicy(ctx, currencyID, fromAddr, duration)
}

func (h *userAPIHandler) RenewMgrGetSenderPolicy(ctx context.Context, currencyID byte, fromAddr string) (time.Duration, error) {
	return h.node.RenewMgr.GetSenderPolicy(ctx, currencyID, fromAddr)
}

func (h *userAPIHandler) RenewMgrRemoveSenderPolicy(ctx context.Context, currencyID byte, fromAddr string) error {
	return h.node.RenewMgr.RemoveSenderPolicy(ctx, currencyID, fromAddr)
}

func (h *userAPIHandler) RenewMgrSetPaychPolicy(ctx context.Context, currencyID byte, fromAddr string, chAddr string, duration time.Duration) error {
	return h.node.RenewMgr.SetPaychPolicy(ctx, currencyID, fromAddr, chAddr, duration)
}

func (h *userAPIHandler) RenewMgrGetPaychPolicy(ctx context.Context, currencyID byte, fromAddr string, chAddr string) (time.Duration, error) {
	return h.node.RenewMgr.GetPaychPolicy(ctx, currencyID, fromAddr, chAddr)
}

func (h *userAPIHandler) RenewMgrRemovePaychPolicy(ctx context.Context, currencyID byte, fromAddr string, chAddr string) error {
	return h.node.RenewMgr.RemovePaychPolicy(ctx, currencyID, fromAddr, chAddr)
}

func (h *userAPIHandler) RenewMgrListCurrencyIDs(ctx context.Context) <-chan byte {
	curChan, errChan := h.node.RenewMgr.ListCurrencyIDs(ctx)
	res := make(chan byte, 32)
	go func() {
		defer close(res)
		for currencyID := range curChan {
			res <- currencyID
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list currency ids: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) RenewMgrListSenders(ctx context.Context, currencyID byte) <-chan string {
	peerChan, errChan := h.node.RenewMgr.ListSenders(ctx, currencyID)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for peer := range peerChan {
			res <- peer
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list peers: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) RenewMgrListPaychs(ctx context.Context, currencyID byte, fromAddr string) <-chan string {
	paychChan, errChan := h.node.RenewMgr.ListPaychs(ctx, currencyID, fromAddr)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for paych := range paychChan {
			res <- paych
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list paychs: %v", err.Error())
		}
	}()
	return res
}

// Reservation manager API
func (h *userAPIHandler) ReservMgrSetDefaultPolicy(ctx context.Context, currencyID byte, unlimited bool, max *big.Int) error {
	return h.node.ReservMgr.SetDefaultPolicy(ctx, currencyID, unlimited, max)
}

func (h *userAPIHandler) ReservMgrGetDefaultPolicy(ctx context.Context, currencyID byte) (ReservMgrPolicyRes, error) {
	unlimited, max, err := h.node.ReservMgr.GetDefaultPolicy(ctx, currencyID)
	return ReservMgrPolicyRes{Unlimited: unlimited, Max: max}, err
}

func (h *userAPIHandler) ReservMgrRemoveDefaultPolicy(ctx context.Context, currencyID byte) error {
	return h.node.ReservMgr.RemoveDefaultPolicy(ctx, currencyID)
}

func (h *userAPIHandler) ReservMgrSetPaychPolicy(ctx context.Context, currencyID byte, chAddr string, unlimited bool, max *big.Int) error {
	return h.node.ReservMgr.SetPaychPolicy(ctx, currencyID, chAddr, unlimited, max)
}

func (h *userAPIHandler) ReservMgrGetPaychPolicy(ctx context.Context, currencyID byte, chAddr string) (ReservMgrPolicyRes, error) {
	unlimited, max, err := h.node.ReservMgr.GetPaychPolicy(ctx, currencyID, chAddr)
	return ReservMgrPolicyRes{Unlimited: unlimited, Max: max}, err
}

func (h *userAPIHandler) ReservMgrRemovePaychPolicy(ctx context.Context, currencyID byte, chAddr string) error {
	return h.node.ReservMgr.RemovePaychPolicy(ctx, currencyID, chAddr)
}

func (h *userAPIHandler) ReservMgrSetPeerPolicy(ctx context.Context, currencyID byte, chAddr string, peerAddr string, unlimited bool, max *big.Int) error {
	return h.node.ReservMgr.SetPeerPolicy(ctx, currencyID, chAddr, peerAddr, unlimited, max)
}

func (h *userAPIHandler) ReservMgrGetPeerPolicy(ctx context.Context, currencyID byte, chAddr string, peerAddr string) (ReservMgrPolicyRes, error) {
	unlimited, max, err := h.node.ReservMgr.GetPeerPolicy(ctx, currencyID, chAddr, peerAddr)
	return ReservMgrPolicyRes{Unlimited: unlimited, Max: max}, err
}

func (h *userAPIHandler) ReservMgrRemovePeerPolicy(ctx context.Context, currencyID byte, chAddr string, peerAddr string) error {
	return h.node.ReservMgr.RemovePeerPolicy(ctx, currencyID, chAddr, peerAddr)
}

func (h *userAPIHandler) ReservMgrListCurrencyIDs(ctx context.Context) <-chan byte {
	curChan, errChan := h.node.ReservMgr.ListCurrencyIDs(ctx)
	res := make(chan byte, 32)
	go func() {
		defer close(res)
		for currencyID := range curChan {
			res <- currencyID
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list currency ids: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) ReservMgrListPaychs(ctx context.Context, currencyID byte) <-chan string {
	paychChan, errChan := h.node.ReservMgr.ListPaychs(ctx, currencyID)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for peer := range paychChan {
			res <- peer
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list paychs: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) ReservMgrListPeers(ctx context.Context, currencyID byte, chAddr string) <-chan string {
	peerChan, errChan := h.node.ReservMgr.ListPeers(ctx, currencyID, chAddr)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for peer := range peerChan {
			res <- peer
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list peers: %v", err.Error())
		}
	}()
	return res
}

// Payment channel monitor API
func (h *userAPIHandler) PaychMonitorCheck(ctx context.Context, outbound bool, currencyID byte, chAddr string) (PaychMonitorCheckRes, error) {
	updated, settlement, err := h.node.PaychMonitor.Check(ctx, outbound, currencyID, chAddr)
	return PaychMonitorCheckRes{Updated: updated, Settlement: settlement}, err
}

// Piece manager API
func (h *userAPIHandler) PieceMgrImport(ctx context.Context, path string) (cid.Cid, error) {
	return h.node.PieceMgr.Import(ctx, path)
}

func (h *userAPIHandler) PieceMgrImportCar(ctx context.Context, path string) (cid.Cid, error) {
	return h.node.PieceMgr.ImportCar(ctx, path)
}

func (h *userAPIHandler) PieceMgrImportSector(ctx context.Context, path string, copy bool) ([]cid.Cid, error) {
	return h.node.PieceMgr.ImportSector(ctx, path, copy)
}

func (h *userAPIHandler) PieceMgrListImported(ctx context.Context) <-chan cid.Cid {
	rootChan, errChan := h.node.PieceMgr.ListImported(ctx)
	res := make(chan cid.Cid, 32)
	go func() {
		defer close(res)
		for id := range rootChan {
			res <- id
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list cids: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) PieceMgrInspect(ctx context.Context, id cid.Cid) (PieceMgrInspectRes, error) {
	exists, path, index, size, copy, err := h.node.PieceMgr.Inspect(ctx, id)
	return PieceMgrInspectRes{Exists: exists, Path: path, Index: index, Size: size, Copy: copy}, err
}

func (h *userAPIHandler) PieceMgrRemove(ctx context.Context, id cid.Cid) error {
	return h.node.PieceMgr.Remove(ctx, id)
}

// Piece serving manager API
func (h *userAPIHandler) CServMgrServe(ctx context.Context, id cid.Cid, currencyID byte, ppb *big.Int) error {
	return h.node.CServMgr.Serve(ctx, id, currencyID, ppb)
}

func (h *userAPIHandler) CServMgrStop(ctx context.Context, id cid.Cid, currencyID byte) error {
	return h.node.CServMgr.Stop(ctx, id, currencyID)
}

func (h *userAPIHandler) CServMgrInspect(ctx context.Context, id cid.Cid, currencyID byte) (CServMgrInspectRes, error) {
	served, ppb, err := h.node.CServMgr.Inspect(ctx, id, currencyID)
	return CServMgrInspectRes{Served: served, PPB: ppb}, err
}

func (h *userAPIHandler) CServMgrListPieceIDs(ctx context.Context) <-chan cid.Cid {
	rootChan, errChan := h.node.CServMgr.ListPieceIDs(ctx)
	res := make(chan cid.Cid, 32)
	go func() {
		defer close(res)
		for id := range rootChan {
			res <- id
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list cids: %v", err.Error())
		}
	}()
	return res
}

func (h *userAPIHandler) CServMgrListCurrencyIDs(ctx context.Context, id cid.Cid) <-chan byte {
	curChan, errChan := h.node.CServMgr.ListCurrencyIDs(ctx, id)
	res := make(chan byte, 32)
	go func() {
		defer close(res)
		for currencyID := range curChan {
			res <- currencyID
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list currency ids: %v", err.Error())
		}
	}()
	return res
}

// Miner proof store API
func (h *userAPIHandler) MPSUpsertMinerProof(ctx context.Context, currencyID byte, minerKeyType byte, minerAddr string, proof []byte) error {
	return h.node.MinerProofStore.UpsertMinerProof(ctx, currencyID, minerKeyType, minerAddr, proof)
}

func (h *userAPIHandler) MPSGetMinerProof(ctx context.Context, currencyID byte) (MPSGetMinerProofRes, error) {
	exists, minerKeyType, minerAddr, proof, err := h.node.MinerProofStore.GetMinerProof(ctx, currencyID)
	return MPSGetMinerProofRes{Exists: exists, MinerKeyType: minerKeyType, MinerAddr: minerAddr, Proof: proof}, err
}

// Addr protocol API
func (h *userAPIHandler) AddrProtoPublish(ctx context.Context) error {
	return h.node.AddrProto.Publish(ctx)
}

// Payment channel protocol API
func (h *userAPIHandler) PaychProtoQueryAdd(ctx context.Context, currencyID byte, toAddr string) (fcroffer.PaychOffer, error) {
	return h.node.PaychProto.QueryAdd(ctx, currencyID, toAddr)
}

func (h *userAPIHandler) PaychProtoAdd(ctx context.Context, chAddr string, offer fcroffer.PaychOffer) error {
	return h.node.PaychProto.Add(ctx, chAddr, offer)
}

func (h *userAPIHandler) PaychProtoRenew(ctx context.Context, currencyID byte, toAddr string, chAddr string) error {
	return h.node.PaychProto.Renew(ctx, currencyID, toAddr, chAddr)
}

// Payment offer protocol API
func (h *userAPIHandler) POfferProtoQueryOffer(ctx context.Context, currencyID byte, route []string, amt *big.Int) (fcroffer.PayOffer, error) {
	return h.node.PofferProto.QueryOffer(ctx, currencyID, route, amt)
}

// Route protocol API
func (h *userAPIHandler) RouteProtoPublish(ctx context.Context, currencyID byte, toAddr string) error {
	return h.node.RouteProto.Publish(ctx, currencyID, toAddr)
}

// Piece offer protocol API
func (h *userAPIHandler) COfferProtoFindOffersAsync(ctx context.Context, currencyID byte, id cid.Cid, max int) <-chan fcroffer.PieceOffer {
	return h.node.CofferProto.FindOffersAsync(ctx, currencyID, id, max)
}

func (h *userAPIHandler) COfferProtoQueryOffer(ctx context.Context, currencyID byte, toAddr string, root cid.Cid) (fcroffer.PieceOffer, error) {
	return h.node.CofferProto.QueryOffer(ctx, currencyID, toAddr, root)
}

// Retrieval manager API
func (h *userAPIHandler) RetMgrRetrieve(ctx context.Context, pieceOffer fcroffer.PieceOffer, payOffer *fcroffer.PayOffer, resCh string, resID uint64, outPath string) <-chan RetMgrRetrieveRes {
	resChan, errChan := h.node.RetMgr.Retrieve(ctx, pieceOffer, payOffer, resCh, resID, outPath)
	res := make(chan RetMgrRetrieveRes, 128)
	go func() {
		defer close(res)
		for {
			select {
			case progress := <-resChan:
				log.Infof("Retreival Progress: %v", progress)
				res <- RetMgrRetrieveRes{
					Progress: fmt.Sprintf("Pulled block at %v", time.Now().Format(time.RFC3339Nano)),
					Err:      "",
				}
			case err := <-errChan:
				if err != nil {
					res <- RetMgrRetrieveRes{
						Err: err.Error(),
					}
				} else {
					res <- RetMgrRetrieveRes{
						Err: "",
					}
				}
				return
			}
		}
	}()
	return res
}

func (h *userAPIHandler) RetMgrRetrieveFromCache(ctx context.Context, root cid.Cid, outPath string) (bool, error) {
	return h.node.RetMgr.RetrieveFromCache(ctx, root, outPath)
}

func (h *userAPIHandler) RetMgrGetRetrievalCacheSize(ctx context.Context) (uint64, error) {
	return h.node.RetMgr.GetRetrievalCacheSize(ctx)
}

func (h *userAPIHandler) RetMgrCleanRetrievalCache(ctx context.Context) error {
	return h.node.RetMgr.CleanRetrievalCache(ctx)
}

func (h *userAPIHandler) RetMgrCleanIncomingProcesses(ctx context.Context) error {
	return h.node.RetMgr.CleanIncomingProcesses(ctx)
}

func (h *userAPIHandler) RetMgrCleanOutgoingProcesses(ctx context.Context) error {
	return h.node.RetMgr.CleanOutgoingProcesses(ctx)
}

func (h *userAPIHandler) GC() error {
	runtime.GC()
	return nil
}

type devHandler struct {
	userAPIHandler
	node *node.Node
}

// Signer API
func (h *devHandler) SignerSign(ctx context.Context, currencyID byte, data []byte) (SignerSignRes, error) {
	sigType, sig, err := h.node.Signer.Sign(ctx, currencyID, data)
	return SignerSignRes{SigType: sigType, Sig: sig}, err
}

// Peer Manager API
func (h *devHandler) PeerMgrAddToHistory(ctx context.Context, currencyID byte, toAddr string, rec peermgr.Record) error {
	return h.node.PeerMgr.AddToHistory(ctx, currencyID, toAddr, rec)
}

// Transactor API
func (h *devHandler) TransactorGenerateVoucher(ctx context.Context, currencyID byte, chAddr string, lane uint64, nonce uint64, redeemed *big.Int) (string, error) {
	return h.node.Transactor.GenerateVoucher(ctx, currencyID, chAddr, lane, nonce, redeemed)
}

// Active out payment channel store API
func (h *devHandler) ActiveOutUpsert(state paychstate.State) {
	h.node.ActiveOutPaych.Upsert(state)
}

func (h *devHandler) ActiveOutRemove(currencyID byte, peerAddr string, chAddr string) {
	h.node.ActiveOutPaych.Remove(currencyID, peerAddr, chAddr)
}

// Inactive out payment channel store API
func (h *devHandler) InactiveOutUpsert(state paychstate.State) {
	h.node.InactiveOutPaych.Upsert(state)
}

// Active in payment channel store API
func (h *devHandler) ActiveInUpsert(state paychstate.State) {
	h.node.ActiveInPaych.Upsert(state)
}

func (h *devHandler) ActiveInRemove(currencyID byte, peerAddr string, chAddr string) {
	h.node.ActiveInPaych.Remove(currencyID, peerAddr, chAddr)
}

// Inactive in payment channel store API
func (h *devHandler) InactiveInUpsert(state paychstate.State) {
	h.node.InactiveInPaych.Upsert(state)
}

// Route store API
func (h *devHandler) RouteStoreAddDirectLink(ctx context.Context, currencyID byte, toAddr string) error {
	return h.node.RouteStore.AddDirectLink(ctx, currencyID, toAddr)
}

func (h *devHandler) RouteStoreRemoveDirectLink(ctx context.Context, currencyID byte, toAddr string) error {
	return h.node.RouteStore.RemoveDirectLink(ctx, currencyID, toAddr)
}

func (h *devHandler) RouteStoreAddRoute(ctx context.Context, currencyID byte, route []string, ttl time.Duration) error {
	return h.node.RouteStore.AddRoute(ctx, currencyID, route, ttl)
}

func (h *devHandler) RouteStoreRemoveRoute(ctx context.Context, currencyID byte, route []string) error {
	return h.node.RouteStore.RemoveRoute(ctx, currencyID, route)
}

func (h *devHandler) RouteStoreListRoutesFrom(ctx context.Context, currencyID byte, toAddr string) <-chan []string {
	routeChan, errChan := h.node.RouteStore.ListRoutesFrom(ctx, currencyID, toAddr)
	res := make(chan []string, 32)
	go func() {
		defer close(res)
		for peer := range routeChan {
			res <- peer
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list routes: %v", err.Error())
		}
	}()
	return res
}

// Subscriber store API
func (h *devHandler) SubStoreAddSubscriber(ctx context.Context, currencyID byte, fromAddr string) error {
	return h.node.SubStore.AddSubscriber(ctx, currencyID, fromAddr)
}

func (h *devHandler) SubStoreRemoveSubscriber(ctx context.Context, currencyID byte, fromAddr string) error {
	return h.node.SubStore.RemoveSubscriber(ctx, currencyID, fromAddr)
}

func (h *devHandler) SubStoreListCurrencyIDs(ctx context.Context) <-chan byte {
	curChan, errChan := h.node.SubStore.ListCurrencyIDs(ctx)
	res := make(chan byte, 32)
	go func() {
		defer close(res)
		for currencyID := range curChan {
			res <- currencyID
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list currency ids: %v", err.Error())
		}
	}()
	return res
}

func (h *devHandler) SubStoreListSubscribers(ctx context.Context, currencyID byte) <-chan string {
	peerChan, errChan := h.node.SubStore.ListSubscribers(ctx, currencyID)
	res := make(chan string, 32)
	go func() {
		defer close(res)
		for peer := range peerChan {
			res <- peer
		}
		err := <-errChan
		if err != nil {
			log.Warnf("Fail to list peers: %v", err.Error())
		}
	}()
	return res
}

// Payment manager API
func (h *devHandler) PayMgrAddInboundChannel(ctx context.Context, currencyID byte, fromAddr string, chAddr string) error {
	return h.node.PayMgr.AddInboundChannel(ctx, currencyID, fromAddr, chAddr)
}

func (h *devHandler) PayMgrRetireInboundChannel(ctx context.Context, currencyID byte, fromAddr string, chAddr string) error {
	return h.node.PayMgr.RetireInboundChannel(ctx, currencyID, fromAddr, chAddr)
}

func (h *devHandler) PayMgrAddOutboundChannel(ctx context.Context, currencyID byte, toAddr string, chAddr string) error {
	return h.node.PayMgr.AddOutboundChannel(ctx, currencyID, toAddr, chAddr)
}

func (h *devHandler) PayMgrRetireOutboundChannel(ctx context.Context, currencyID byte, toAddr string, chAddr string) error {
	return h.node.PayMgr.RetireOutboundChannel(ctx, currencyID, toAddr, chAddr)
}

// Offer manager API
func (h *devHandler) OfferMgrGetNonce(ctx context.Context) (uint64, error) {
	return h.node.OfferMgr.GetNonce(ctx)
}

func (h *devHandler) OfferMgrSetNonce(ctx context.Context, nonce uint64) error {
	return h.node.OfferMgr.SetNonce(ctx, nonce)
}

// Payment channel monitor API
func (h *devHandler) PaychMonitorTrack(ctx context.Context, outbound bool, currencyID byte, chAddr string, settlement time.Time) error {
	return h.node.PaychMonitor.Track(ctx, outbound, currencyID, chAddr, settlement)
}

func (h *devHandler) PaychMonitorRenew(ctx context.Context, outbound bool, currencyID byte, chAddr string, newSettlement time.Time) error {
	return h.node.PaychMonitor.Renew(ctx, outbound, currencyID, chAddr, newSettlement)
}

func (h *devHandler) PaychMonitorRetire(ctx context.Context, outbound bool, currencyID byte, chAddr string) error {
	return h.node.PaychMonitor.Retire(ctx, outbound, currencyID, chAddr)
}

// Miner proof store API
func (h *devHandler) MPSVerifyMinerProof(currencyID byte, addr string, minerKeyType byte, minerAddr string, proof []byte) error {
	return mpstore.VerifyMinerProof(currencyID, addr, minerKeyType, minerAddr, proof)
}

// Addr protocol API
func (h *devHandler) AddrProtoConnectToPeer(ctx context.Context, currencyID byte, toAddr string) (peer.ID, error) {
	return h.node.AddrProto.ConnectToPeer(ctx, currencyID, toAddr)
}

func (h *devHandler) AddrProtoQueryAddr(ctx context.Context, currencyID byte, pi peer.AddrInfo) (string, error) {
	return h.node.AddrProto.QueryAddr(ctx, currencyID, pi)
}

// Pay protocol API
func (h *devHandler) PayProtoReceive(ctx context.Context, currencyID byte, originalAddr string) (*big.Int, error) {
	return h.node.PayProto.Receive(ctx, currencyID, originalAddr)
}

func (h *devHandler) PayProtoPayForSelf(ctx context.Context, currencyID byte, toAddr string, resCh string, resID uint64, pettyAmtRequired *big.Int) error {
	return h.node.PayProto.PayForSelf(ctx, currencyID, toAddr, resCh, resID, pettyAmtRequired)
}

func (h *devHandler) PayProtoPayForSelfWithOffer(ctx context.Context, receivedOffer fcroffer.PayOffer, resCh string, resID uint64, pettyAmtRequired *big.Int) error {
	return h.node.PayProto.PayForSelfWithOffer(ctx, receivedOffer, resCh, resID, pettyAmtRequired)
}
