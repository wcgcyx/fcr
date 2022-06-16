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
	"math/big"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/paychstate"
	"github.com/wcgcyx/fcr/peermgr"
)

// User APIs.
type UserAPI struct {
	// Network API
	NetAddr    func(ctx context.Context) (peer.AddrInfo, error)
	NetConnect func(ctx context.Context, pi peer.AddrInfo) error

	// Signer API
	SignerSetKey          func(ctx context.Context, currencyID byte, keyType byte, prv []byte) error
	SignerGetAddr         func(ctx context.Context, currencyID byte) (SignerGetAddrRes, error)
	SignerRetireKey       func(ctx context.Context, currencyID byte, timeout time.Duration) <-chan string
	SignerStopRetire      func(ctx context.Context, currencyID byte) error
	SignerListCurrencyIDs func(ctx context.Context) <-chan byte

	// Peer Manager API
	PeerMgrAddPeer         func(ctx context.Context, currencyID byte, toAddr string, pi peer.AddrInfo) error
	PeerMgrHasPeer         func(ctx context.Context, currencyID byte, toAddr string) (bool, error)
	PeerMgrRemovePeer      func(ctx context.Context, currencyID byte, toAddr string) error
	PeerMgrPeerAddr        func(ctx context.Context, currencyID byte, toAddr string) (peer.AddrInfo, error)
	PeerMgrIsBlocked       func(ctx context.Context, currencyID byte, toAddr string) (bool, error)
	PeerMgrBlockPeer       func(ctx context.Context, currencyID byte, toAddr string) error
	PeerMgrUnblockPeer     func(ctx context.Context, currencyID byte, toAddr string) error
	PeerMgrListCurrencyIDs func(ctx context.Context) <-chan byte
	PeerMgrListPeers       func(ctx context.Context, currencyID byte) <-chan string
	PeerMgrRemoveRecord    func(ctx context.Context, currencyID byte, toAddr string, recID *big.Int) error
	PeerMgrSetRecID        func(ctx context.Context, currencyID byte, toAddr string, recID *big.Int) error
	PeerMgrListHistory     func(ctx context.Context, currencyID byte, toAddr string) <-chan PeerMgrListHistoryRes

	// Transactor API
	TransactorCreate        func(ctx context.Context, currencyID byte, toAddr string, amt *big.Int) (string, error)
	TransactorTopup         func(ctx context.Context, currencyID byte, chAddr string, amt *big.Int) error
	TransactorCheck         func(ctx context.Context, currencyID byte, chAddr string) (TransactorCheckRes, error)
	TransactorUpdate        func(ctx context.Context, currencyID byte, chAddr string, voucher string) error
	TransactorSettle        func(ctx context.Context, currencyID byte, chAddr string) error
	TransactorCollect       func(ctx context.Context, currencyID byte, chAddr string) error
	TransactorVerifyVoucher func(currencyID byte, voucher string) (TransactorVerifyVoucherRes, error)
	TransactorGetHeight     func(ctx context.Context, currencyID byte) (int64, error)
	TransactorGetBalance    func(ctx context.Context, currencyID byte, addr string) (*big.Int, error)

	// Active out payment channel store API
	ActiveOutRead             func(ctx context.Context, currencyID byte, peerAddr string, chAddr string) (paychstate.State, error)
	ActiveOutListCurrencyIDs  func(ctx context.Context) <-chan byte
	ActiveOutListPeers        func(ctx context.Context, currencyID byte) <-chan string
	ActiveOutListPaychsByPeer func(ctx context.Context, currencyID byte, peerAddr string) <-chan string

	// Inactive out payment channel store API
	InactiveOutRemove           func(currencyID byte, peerAddr string, chAddr string)
	InactiveOutRead             func(ctx context.Context, currencyID byte, peerAddr string, chAddr string) (paychstate.State, error)
	InactiveOutListCurrencyIDs  func(ctx context.Context) <-chan byte
	InactiveOutListPeers        func(ctx context.Context, currencyID byte) <-chan string
	InactiveOutListPaychsByPeer func(ctx context.Context, currencyID byte, peerAddr string) <-chan string

	// Active in payment channel store API
	ActiveInRead             func(ctx context.Context, currencyID byte, peerAddr string, chAddr string) (paychstate.State, error)
	ActiveInListCurrencyIDs  func(ctx context.Context) <-chan byte
	ActiveInListPeers        func(ctx context.Context, currencyID byte) <-chan string
	ActiveInListPaychsByPeer func(ctx context.Context, currencyID byte, peerAddr string) <-chan string

	// Inactive in payment channel store API
	InactiveInRemove           func(currencyID byte, peerAddr string, chAddr string)
	InactiveInRead             func(ctx context.Context, currencyID byte, peerAddr string, chAddr string) (paychstate.State, error)
	InactiveInListCurrencyIDs  func(ctx context.Context) <-chan byte
	InactiveInListPeers        func(ctx context.Context, currencyID byte) <-chan string
	InactiveInListPaychsByPeer func(ctx context.Context, currencyID byte, peerAddr string) <-chan string

	// Payment serving manager API
	PservMgrServe           func(ctx context.Context, currencyID byte, toAddr string, chAddr string, ppp *big.Int, period *big.Int) error
	PservMgrStop            func(ctx context.Context, currencyID byte, toAddr string, chAddr string) error
	PservMgrInspect         func(ctx context.Context, currencyID byte, toAddr string, chAddr string) (PservMgrInspectRes, error)
	PservMgrListCurrencyIDs func(ctx context.Context) <-chan byte
	PservMgrListRecipients  func(ctx context.Context, currencyID byte) <-chan string
	PservMgrListServings    func(ctx context.Context, currencyID byte, toAddr string) <-chan string

	// Route store API
	RouteStoreMaxHop       func(ctx context.Context, currencyID byte) (uint64, error)
	RouteStoreListRoutesTo func(ctx context.Context, currencyID byte, toAddr string) <-chan []string

	// Payment manager API
	PayMgrReserveForSelf               func(ctx context.Context, currencyID byte, toAddr string, pettyAmt *big.Int, expiration time.Time, inactivity time.Duration) (PayMgrReserveForSelfRes, error)
	PayMgrReserveForSelfWithOffer      func(ctx context.Context, receivedOffer fcroffer.PayOffer) (PayMgrReserveForSelfWithOfferRes, error)
	PayMgrUpdateOutboundChannelBalance func(ctx context.Context, currencyID byte, toAddr string, chAddr string) error
	PayMgrBearNetworkLoss              func(ctx context.Context, currencyID byte, toAddr string, chAddr string) error

	// Settlement manager API
	SettleMgrSetDefaultPolicy    func(ctx context.Context, currencyID byte, duration time.Duration) error
	SettleMgrGetDefaultPolicy    func(ctx context.Context, currencyID byte) (time.Duration, error)
	SettleMgrRemoveDefaultPolicy func(ctx context.Context, currencyID byte) error
	SettleMgrSetSenderPolicy     func(ctx context.Context, currencyID byte, fromAddr string, duration time.Duration) error
	SettleMgrGetSenderPolicy     func(ctx context.Context, currencyID byte, fromAddr string) (time.Duration, error)
	SettleMgrRemoveSenderPolicy  func(ctx context.Context, currencyID byte, fromAddr string) error
	SettleMgrListCurrencyIDs     func(ctx context.Context) <-chan byte
	SettleMgrListSenders         func(ctx context.Context, currencyID byte) <-chan string

	// Renew manager API
	RenewMgrSetDefaultPolicy    func(ctx context.Context, currencyID byte, duration time.Duration) error
	RenewMgrGetDefaultPolicy    func(ctx context.Context, currencyID byte) (time.Duration, error)
	RenewMgrRemoveDefaultPolicy func(ctx context.Context, currencyID byte) error
	RenewMgrSetSenderPolicy     func(ctx context.Context, currencyID byte, fromAddr string, duration time.Duration) error
	RenewMgrGetSenderPolicy     func(ctx context.Context, currencyID byte, fromAddr string) (time.Duration, error)
	RenewMgrRemoveSenderPolicy  func(ctx context.Context, currencyID byte, fromAddr string) error
	RenewMgrSetPaychPolicy      func(ctx context.Context, currencyID byte, fromAddr string, chAddr string, duration time.Duration) error
	RenewMgrGetPaychPolicy      func(ctx context.Context, currencyID byte, fromAddr string, chAddr string) (time.Duration, error)
	RenewMgrRemovePaychPolicy   func(ctx context.Context, currencyID byte, fromAddr string, chAddr string) error
	RenewMgrListCurrencyIDs     func(ctx context.Context) <-chan byte
	RenewMgrListSenders         func(ctx context.Context, currencyID byte) <-chan string
	RenewMgrListPaychs          func(ctx context.Context, currencyID byte, fromAddr string) <-chan string

	// Reservation manager API
	ReservMgrSetDefaultPolicy    func(ctx context.Context, currencyID byte, unlimited bool, max *big.Int) error
	ReservMgrGetDefaultPolicy    func(ctx context.Context, currencyID byte) (ReservMgrPolicyRes, error)
	ReservMgrRemoveDefaultPolicy func(ctx context.Context, currencyID byte) error
	ReservMgrSetPaychPolicy      func(ctx context.Context, currencyID byte, chAddr string, unlimited bool, max *big.Int) error
	ReservMgrGetPaychPolicy      func(ctx context.Context, currencyID byte, chAddr string) (ReservMgrPolicyRes, error)
	ReservMgrRemovePaychPolicy   func(ctx context.Context, currencyID byte, chAddr string) error
	ReservMgrSetPeerPolicy       func(ctx context.Context, currencyID byte, chAddr string, peerAddr string, unlimited bool, max *big.Int) error
	ReservMgrGetPeerPolicy       func(ctx context.Context, currencyID byte, chAddr string, peerAddr string) (ReservMgrPolicyRes, error)
	ReservMgrRemovePeerPolicy    func(ctx context.Context, currencyID byte, chAddr string, peerAddr string) error
	ReservMgrListCurrencyIDs     func(ctx context.Context) <-chan byte
	ReservMgrListPaychs          func(ctx context.Context, currencyID byte) <-chan string
	ReservMgrListPeers           func(ctx context.Context, currencyID byte, chAddr string) <-chan string

	// Payment channel monitor API
	PaychMonitorCheck func(ctx context.Context, outbound bool, currencyID byte, chAddr string) (PaychMonitorCheckRes, error)

	// Piece manager API
	PieceMgrImport       func(ctx context.Context, path string) (cid.Cid, error)
	PieceMgrImportCar    func(ctx context.Context, path string) (cid.Cid, error)
	PieceMgrImportSector func(ctx context.Context, path string, copy bool) ([]cid.Cid, error)
	PieceMgrListImported func(ctx context.Context) <-chan cid.Cid
	PieceMgrInspect      func(ctx context.Context, id cid.Cid) (PieceMgrInspectRes, error)
	PieceMgrRemove       func(ctx context.Context, id cid.Cid) error

	// Piece serving manager API
	CServMgrServe           func(ctx context.Context, id cid.Cid, currencyID byte, ppb *big.Int) error
	CServMgrStop            func(ctx context.Context, id cid.Cid, currencyID byte) error
	CServMgrInspect         func(ctx context.Context, id cid.Cid, currencyID byte) (CServMgrInspectRes, error)
	CServMgrListPieceIDs    func(ctx context.Context) <-chan cid.Cid
	CServMgrListCurrencyIDs func(ctx context.Context, id cid.Cid) <-chan byte

	// Miner proof store API
	MPSUpsertMinerProof func(ctx context.Context, currencyID byte, minerKeyType byte, minerAddr string, proof []byte) error
	MPSGetMinerProof    func(ctx context.Context, currencyID byte) (MPSGetMinerProofRes, error)

	// Addr protocol API
	AddrProtoPublish func(ctx context.Context) error

	// Payment channel protocol API
	PaychProtoQueryAdd func(ctx context.Context, currencyID byte, toAddr string) (fcroffer.PaychOffer, error)
	PaychProtoAdd      func(ctx context.Context, chAddr string, offer fcroffer.PaychOffer) error
	PaychProtoRenew    func(ctx context.Context, currencyID byte, toAddr string, chAddr string) error

	// Payment offer protocol API
	POfferProtoQueryOffer func(ctx context.Context, currencyID byte, route []string, amt *big.Int) (fcroffer.PayOffer, error)

	// Route protocol API
	RouteProtoPublish func(ctx context.Context, currencyID byte, toAddr string) error

	// Piece offer protocol API
	COfferProtoFindOffersAsync func(ctx context.Context, currencyID byte, id cid.Cid, max int) <-chan fcroffer.PieceOffer
	COfferProtoQueryOffer      func(ctx context.Context, currencyID byte, toAddr string, root cid.Cid) (fcroffer.PieceOffer, error)

	// Retrieval manager API
	RetMgrRetrieve               func(ctx context.Context, pieceOffer fcroffer.PieceOffer, payOffer *fcroffer.PayOffer, resCh string, resID uint64, outPath string) <-chan RetMgrRetrieveRes
	RetMgrRetrieveFromCache      func(ctx context.Context, root cid.Cid, outPath string) (bool, error)
	RetMgrGetRetrievalCacheSize  func(ctx context.Context) (uint64, error)
	RetMgrCleanRetrievalCache    func(ctx context.Context) error
	RetMgrCleanIncomingProcesses func(ctx context.Context) error
	RetMgrCleanOutgoingProcesses func(ctx context.Context) error

	// GC API
	GC func() error
}

// Developer APIs.
type DevAPI struct {
	/****************************************************************************/
	/* Note:
	 * The following APIs are for developers for testing purpose ONLY.
	 * They provide direct access to system components.
	 * Calling these APIs is likely to cause unexpected behaviour of the system.
	 * Use only if you understand how every API works.
	 */
	/****************************************************************************/

	// Network API
	NetAddr    func(ctx context.Context) (peer.AddrInfo, error)
	NetConnect func(ctx context.Context, pi peer.AddrInfo) error

	// Signer API
	SignerSetKey          func(ctx context.Context, currencyID byte, keyType byte, prv []byte) error
	SignerGetAddr         func(ctx context.Context, currencyID byte) (SignerGetAddrRes, error)
	SignerSign            func(ctx context.Context, currencyID byte, data []byte) (SignerSignRes, error)
	SignerRetireKey       func(ctx context.Context, currencyID byte, timeout time.Duration) <-chan string
	SignerStopRetire      func(ctx context.Context, currencyID byte) error
	SignerListCurrencyIDs func(ctx context.Context) <-chan byte

	// Peer Manager API
	PeerMgrAddPeer         func(ctx context.Context, currencyID byte, toAddr string, pi peer.AddrInfo) error
	PeerMgrHasPeer         func(ctx context.Context, currencyID byte, toAddr string) (bool, error)
	PeerMgrRemovePeer      func(ctx context.Context, currencyID byte, toAddr string) error
	PeerMgrPeerAddr        func(ctx context.Context, currencyID byte, toAddr string) (peer.AddrInfo, error)
	PeerMgrIsBlocked       func(ctx context.Context, currencyID byte, toAddr string) (bool, error)
	PeerMgrBlockPeer       func(ctx context.Context, currencyID byte, toAddr string) error
	PeerMgrUnblockPeer     func(ctx context.Context, currencyID byte, toAddr string) error
	PeerMgrListCurrencyIDs func(ctx context.Context) <-chan byte
	PeerMgrListPeers       func(ctx context.Context, currencyID byte) <-chan string
	PeerMgrAddToHistory    func(ctx context.Context, currencyID byte, toAddr string, rec peermgr.Record) error
	PeerMgrRemoveRecord    func(ctx context.Context, currencyID byte, toAddr string, recID *big.Int) error
	PeerMgrSetRecID        func(ctx context.Context, currencyID byte, toAddr string, recID *big.Int) error
	PeerMgrListHistory     func(ctx context.Context, currencyID byte, toAddr string) <-chan PeerMgrListHistoryRes

	// Transactor API
	TransactorCreate          func(ctx context.Context, currencyID byte, toAddr string, amt *big.Int) (string, error)
	TransactorTopup           func(ctx context.Context, currencyID byte, chAddr string, amt *big.Int) error
	TransactorCheck           func(ctx context.Context, currencyID byte, chAddr string) (TransactorCheckRes, error)
	TransactorUpdate          func(ctx context.Context, currencyID byte, chAddr string, voucher string) error
	TransactorSettle          func(ctx context.Context, currencyID byte, chAddr string) error
	TransactorCollect         func(ctx context.Context, currencyID byte, chAddr string) error
	TransactorGenerateVoucher func(ctx context.Context, currencyID byte, chAddr string, lane uint64, nonce uint64, redeemed *big.Int) (string, error)
	TransactorVerifyVoucher   func(currencyID byte, voucher string) (TransactorVerifyVoucherRes, error)
	TransactorGetHeight       func(ctx context.Context, currencyID byte) (int64, error)
	TransactorGetBalance      func(ctx context.Context, currencyID byte, addr string) (*big.Int, error)

	// Active out payment channel store API
	ActiveOutUpsert           func(state paychstate.State)
	ActiveOutRemove           func(currencyID byte, peerAddr string, chAddr string)
	ActiveOutRead             func(ctx context.Context, currencyID byte, peerAddr string, chAddr string) (paychstate.State, error)
	ActiveOutListCurrencyIDs  func(ctx context.Context) <-chan byte
	ActiveOutListPeers        func(ctx context.Context, currencyID byte) <-chan string
	ActiveOutListPaychsByPeer func(ctx context.Context, currencyID byte, peerAddr string) <-chan string

	// Inactive out payment channel store API
	InactiveOutUpsert           func(state paychstate.State)
	InactiveOutRemove           func(currencyID byte, peerAddr string, chAddr string)
	InactiveOutRead             func(ctx context.Context, currencyID byte, peerAddr string, chAddr string) (paychstate.State, error)
	InactiveOutListCurrencyIDs  func(ctx context.Context) <-chan byte
	InactiveOutListPeers        func(ctx context.Context, currencyID byte) <-chan string
	InactiveOutListPaychsByPeer func(ctx context.Context, currencyID byte, peerAddr string) <-chan string

	// Active in payment channel store API
	ActiveInUpsert           func(state paychstate.State)
	ActiveInRemove           func(currencyID byte, peerAddr string, chAddr string)
	ActiveInRead             func(ctx context.Context, currencyID byte, peerAddr string, chAddr string) (paychstate.State, error)
	ActiveInListCurrencyIDs  func(ctx context.Context) <-chan byte
	ActiveInListPeers        func(ctx context.Context, currencyID byte) <-chan string
	ActiveInListPaychsByPeer func(ctx context.Context, currencyID byte, peerAddr string) <-chan string

	// Inactive in payment channel store API
	InactiveInUpsert           func(state paychstate.State)
	InactiveInRemove           func(currencyID byte, peerAddr string, chAddr string)
	InactiveInRead             func(ctx context.Context, currencyID byte, peerAddr string, chAddr string) (paychstate.State, error)
	InactiveInListCurrencyIDs  func(ctx context.Context) <-chan byte
	InactiveInListPeers        func(ctx context.Context, currencyID byte) <-chan string
	InactiveInListPaychsByPeer func(ctx context.Context, currencyID byte, peerAddr string) <-chan string

	// Payment serving manager API
	PservMgrServe           func(ctx context.Context, currencyID byte, toAddr string, chAddr string, ppp *big.Int, period *big.Int) error
	PservMgrStop            func(ctx context.Context, currencyID byte, toAddr string, chAddr string) error
	PservMgrInspect         func(ctx context.Context, currencyID byte, toAddr string, chAddr string) (PservMgrInspectRes, error)
	PservMgrListCurrencyIDs func(ctx context.Context) <-chan byte
	PservMgrListRecipients  func(ctx context.Context, currencyID byte) <-chan string
	PservMgrListServings    func(ctx context.Context, currencyID byte, toAddr string) <-chan string

	// Route store API
	RouteStoreMaxHop           func(ctx context.Context, currencyID byte) (uint64, error)
	RouteStoreAddDirectLink    func(ctx context.Context, currencyID byte, toAddr string) error
	RouteStoreRemoveDirectLink func(ctx context.Context, currencyID byte, toAddr string) error
	RouteStoreAddRoute         func(ctx context.Context, currencyID byte, route []string, ttl time.Duration) error
	RouteStoreRemoveRoute      func(ctx context.Context, currencyID byte, route []string) error
	RouteStoreListRoutesTo     func(ctx context.Context, currencyID byte, toAddr string) <-chan []string
	RouteStoreListRoutesFrom   func(ctx context.Context, currencyID byte, toAddr string) <-chan []string

	// Subscriber store API
	SubStoreAddSubscriber    func(ctx context.Context, currencyID byte, fromAddr string) error
	SubStoreRemoveSubscriber func(ctx context.Context, currencyID byte, fromAddr string) error
	SubStoreListCurrencyIDs  func(ctx context.Context) <-chan byte
	SubStoreListSubscribers  func(ctx context.Context, currencyID byte) <-chan string

	// Payment manager API
	PayMgrReserveForSelf               func(ctx context.Context, currencyID byte, toAddr string, pettyAmt *big.Int, expiration time.Time, inactivity time.Duration) (PayMgrReserveForSelfRes, error)
	PayMgrReserveForSelfWithOffer      func(ctx context.Context, receivedOffer fcroffer.PayOffer) (PayMgrReserveForSelfWithOfferRes, error)
	PayMgrAddInboundChannel            func(ctx context.Context, currencyID byte, fromAddr string, chAddr string) error
	PayMgrRetireInboundChannel         func(ctx context.Context, currencyID byte, fromAddr string, chAddr string) error
	PayMgrAddOutboundChannel           func(ctx context.Context, currencyID byte, toAddr string, chAddr string) error
	PayMgrRetireOutboundChannel        func(ctx context.Context, currencyID byte, toAddr string, chAddr string) error
	PayMgrUpdateOutboundChannelBalance func(ctx context.Context, currencyID byte, toAddr string, chAddr string) error
	PayMgrBearNetworkLoss              func(ctx context.Context, currencyID byte, toAddr string, chAddr string) error

	// Settlement manager API
	SettleMgrSetDefaultPolicy    func(ctx context.Context, currencyID byte, duration time.Duration) error
	SettleMgrGetDefaultPolicy    func(ctx context.Context, currencyID byte) (time.Duration, error)
	SettleMgrRemoveDefaultPolicy func(ctx context.Context, currencyID byte) error
	SettleMgrSetSenderPolicy     func(ctx context.Context, currencyID byte, fromAddr string, duration time.Duration) error
	SettleMgrGetSenderPolicy     func(ctx context.Context, currencyID byte, fromAddr string) (time.Duration, error)
	SettleMgrRemoveSenderPolicy  func(ctx context.Context, currencyID byte, fromAddr string) error
	SettleMgrListCurrencyIDs     func(ctx context.Context) <-chan byte
	SettleMgrListSenders         func(ctx context.Context, currencyID byte) <-chan string

	// Renew manager API
	RenewMgrSetDefaultPolicy    func(ctx context.Context, currencyID byte, duration time.Duration) error
	RenewMgrGetDefaultPolicy    func(ctx context.Context, currencyID byte) (time.Duration, error)
	RenewMgrRemoveDefaultPolicy func(ctx context.Context, currencyID byte) error
	RenewMgrSetSenderPolicy     func(ctx context.Context, currencyID byte, fromAddr string, duration time.Duration) error
	RenewMgrGetSenderPolicy     func(ctx context.Context, currencyID byte, fromAddr string) (time.Duration, error)
	RenewMgrRemoveSenderPolicy  func(ctx context.Context, currencyID byte, fromAddr string) error
	RenewMgrSetPaychPolicy      func(ctx context.Context, currencyID byte, fromAddr string, chAddr string, duration time.Duration) error
	RenewMgrGetPaychPolicy      func(ctx context.Context, currencyID byte, fromAddr string, chAddr string) (time.Duration, error)
	RenewMgrRemovePaychPolicy   func(ctx context.Context, currencyID byte, fromAddr string, chAddr string) error
	RenewMgrListCurrencyIDs     func(ctx context.Context) <-chan byte
	RenewMgrListSenders         func(ctx context.Context, currencyID byte) <-chan string
	RenewMgrListPaychs          func(ctx context.Context, currencyID byte, fromAddr string) <-chan string

	// Reservation manager API
	ReservMgrSetDefaultPolicy    func(ctx context.Context, currencyID byte, unlimited bool, max *big.Int) error
	ReservMgrGetDefaultPolicy    func(ctx context.Context, currencyID byte) (ReservMgrPolicyRes, error)
	ReservMgrRemoveDefaultPolicy func(ctx context.Context, currencyID byte) error
	ReservMgrSetPaychPolicy      func(ctx context.Context, currencyID byte, chAddr string, unlimited bool, max *big.Int) error
	ReservMgrGetPaychPolicy      func(ctx context.Context, currencyID byte, chAddr string) (ReservMgrPolicyRes, error)
	ReservMgrRemovePaychPolicy   func(ctx context.Context, currencyID byte, chAddr string) error
	ReservMgrSetPeerPolicy       func(ctx context.Context, currencyID byte, chAddr string, peerAddr string, unlimited bool, max *big.Int) error
	ReservMgrGetPeerPolicy       func(ctx context.Context, currencyID byte, chAddr string, peerAddr string) (ReservMgrPolicyRes, error)
	ReservMgrRemovePeerPolicy    func(ctx context.Context, currencyID byte, chAddr string, peerAddr string) error
	ReservMgrListCurrencyIDs     func(ctx context.Context) <-chan byte
	ReservMgrListPaychs          func(ctx context.Context, currencyID byte) <-chan string
	ReservMgrListPeers           func(ctx context.Context, currencyID byte, chAddr string) <-chan string

	// Offer manager API
	OfferMgrGetNonce func(ctx context.Context) (uint64, error)
	OfferMgrSetNonce func(ctx context.Context, nonce uint64) error

	// Payment channel monitor API
	PaychMonitorTrack  func(ctx context.Context, outbound bool, currencyID byte, chAddr string, settlement time.Time) error
	PaychMonitorCheck  func(ctx context.Context, outbound bool, currencyID byte, chAddr string) (PaychMonitorCheckRes, error)
	PaychMonitorRenew  func(ctx context.Context, outbound bool, currencyID byte, chAddr string, newSettlement time.Time) error
	PaychMonitorRetire func(ctx context.Context, outbound bool, currencyID byte, chAddr string) error

	// Piece manager API
	PieceMgrImport       func(ctx context.Context, path string) (cid.Cid, error)
	PieceMgrImportCar    func(ctx context.Context, path string) (cid.Cid, error)
	PieceMgrImportSector func(ctx context.Context, path string, copy bool) ([]cid.Cid, error)
	PieceMgrListImported func(ctx context.Context) <-chan cid.Cid
	PieceMgrInspect      func(ctx context.Context, id cid.Cid) (PieceMgrInspectRes, error)
	PieceMgrRemove       func(ctx context.Context, id cid.Cid) error

	// Piece serving manager API
	CServMgrServe           func(ctx context.Context, id cid.Cid, currencyID byte, ppb *big.Int) error
	CServMgrStop            func(ctx context.Context, id cid.Cid, currencyID byte) error
	CServMgrInspect         func(ctx context.Context, id cid.Cid, currencyID byte) (CServMgrInspectRes, error)
	CServMgrListPieceIDs    func(ctx context.Context) <-chan cid.Cid
	CServMgrListCurrencyIDs func(ctx context.Context, id cid.Cid) <-chan byte

	// Miner proof store API
	MPSUpsertMinerProof func(ctx context.Context, currencyID byte, minerKeyType byte, minerAddr string, proof []byte) error
	MPSGetMinerProof    func(ctx context.Context, currencyID byte) (MPSGetMinerProofRes, error)
	MPSVerifyMinerProof func(currencyID byte, addr string, minerKeyType byte, minerAddr string, proof []byte) error

	// Addr protocol API
	AddrProtoPublish       func(ctx context.Context) error
	AddrProtoConnectToPeer func(ctx context.Context, currencyID byte, toAddr string) (peer.ID, error)
	AddrProtoQueryAddr     func(ctx context.Context, currencyID byte, pi peer.AddrInfo) (string, error)

	// Payment channel protocol API
	PaychProtoQueryAdd func(ctx context.Context, currencyID byte, toAddr string) (fcroffer.PaychOffer, error)
	PaychProtoAdd      func(ctx context.Context, chAddr string, offer fcroffer.PaychOffer) error
	PaychProtoRenew    func(ctx context.Context, currencyID byte, toAddr string, chAddr string) error

	// Payment offer protocol API
	POfferProtoQueryOffer func(ctx context.Context, currencyID byte, route []string, amt *big.Int) (fcroffer.PayOffer, error)

	// Route protocol API
	RouteProtoPublish func(ctx context.Context, currencyID byte, toAddr string) error

	// Piece offer protocol API
	COfferProtoFindOffersAsync func(ctx context.Context, currencyID byte, id cid.Cid, max int) <-chan fcroffer.PieceOffer
	COfferProtoQueryOffer      func(ctx context.Context, currencyID byte, toAddr string, root cid.Cid) (fcroffer.PieceOffer, error)

	// Pay protocol API
	PayProtoReceive             func(ctx context.Context, currencyID byte, originalAddr string) (*big.Int, error)
	PayProtoPayForSelf          func(ctx context.Context, currencyID byte, toAddr string, resCh string, resID uint64, pettyAmtRequired *big.Int) error
	PayProtoPayForSelfWithOffer func(ctx context.Context, receivedOffer fcroffer.PayOffer, resCh string, resID uint64, pettyAmtRequired *big.Int) error

	// Retrieval manager API
	RetMgrRetrieve               func(ctx context.Context, pieceOffer fcroffer.PieceOffer, payOffer *fcroffer.PayOffer, resCh string, resID uint64, outPath string) <-chan RetMgrRetrieveRes
	RetMgrRetrieveFromCache      func(ctx context.Context, root cid.Cid, outPath string) (bool, error)
	RetMgrGetRetrievalCacheSize  func(ctx context.Context) (uint64, error)
	RetMgrCleanRetrievalCache    func(ctx context.Context) error
	RetMgrCleanIncomingProcesses func(ctx context.Context) error
	RetMgrCleanOutgoingProcesses func(ctx context.Context) error

	// GC API
	GC func() error
}
