package paynetmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"math/big"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wcgcyx/fcr/internal/paychoffer"
	"github.com/wcgcyx/fcr/internal/payoffer"
)

// PaymentNetworkManager is used to manage the payment network.
type PaymentNetworkManager interface {
	//////////////////////////////
	/* Outbound channel related */
	//////////////////////////////

	// QueryOutboundChOffer is used to query a peer for a outbound channel offer.
	// It takes a context, a currency ID, to address, peer addr info as arguments.
	// It returns a paych offer and error.
	QueryOutboundChOffer(ctx context.Context, currencyID uint64, toAddr string, pi peer.AddrInfo) (*paychoffer.PaychOffer, error)

	// CreateOutboundCh is used to create an outbound channel.
	// It takes a context, a paych offer, amt, peer addr info as arguments.
	// It returns error.
	CreateOutboundCh(ctx context.Context, offer *paychoffer.PaychOffer, amt *big.Int, pi peer.AddrInfo) error

	// TopupOutboundCh is used to topup an active outbound channel.
	// It takes a context, a currency id, recipient address, amt as arguments.
	// It returns error.
	TopupOutboundCh(ctx context.Context, currencyID uint64, toAddr string, amt *big.Int) error

	// ListActiveOutboundChs lists all active outbound channels.
	// It takes a context as the argument.
	// It returns a map from currency id to a list of channel addresses and error.
	ListActiveOutboundChs(ctx context.Context) (map[uint64][]string, error)

	// ListOutboundChs lists all outbound channels.
	// It takes a context as the argument.
	// It returns a map from currency id to a list of channel addresses and error.
	ListOutboundChs(ctx context.Context) (map[uint64][]string, error)

	// InspectOutboundCh inspects an outbound payment channel.
	// It takes a context, currency id and channel address as arguments.
	// It returns recipient address, redeemed amount, total amount, settlement time of this channel (as specified when creating this channel),
	// boolean indicates if channel is active, current chain height, settling height and error.
	InspectOutboundCh(ctx context.Context, currencyID uint64, chAddr string) (string, *big.Int, *big.Int, int64, bool, int64, int64, error)

	// SettleOutboundCh settles an outbound payment channel.
	// It takes a context, currency id and channel address as arguments.
	// It returns the error.
	SettleOutboundCh(ctx context.Context, currencyID uint64, chAddr string) error

	// CollectOutboundCh collects an outbound payment channel.
	// It takes a context, currency id and channel address as arguments.
	// It returns the error.
	CollectOutboundCh(ctx context.Context, currencyID uint64, chAddr string) error

	/////////////////////////////
	/* Inbound channel related */
	/////////////////////////////

	// SetInboundChOffer sets the inbound channel settlement delay.
	// It takes a context, a settlement delay as the argument.
	SetInboundChOffer(settlementDelay time.Duration)

	// ListInboundChs lists all inbound channels.
	// It takes a context as the argument.
	// It returns a map from currency id to list of paych addresses and error.
	ListInboundChs(ctx context.Context) (map[uint64][]string, error)

	// InspectInboundCh inspects an inbound payment channel.
	// It takes a context, currency id and channel address as arguments.
	// It returns from address, redeemed amount, total amount, settlement time of this channel (as specified when creating this channel),
	// boolean indicates if channel is active, current chain height, settling height and error.
	InspectInboundCh(ctx context.Context, currencyID uint64, chAddr string) (string, *big.Int, *big.Int, int64, bool, int64, int64, error)

	// SettleInboundCh settles an inbound payment channel.
	// It takes a context, currency id and channel address as arguments.
	// It returns the error.
	SettleInboundCh(ctx context.Context, currencyID uint64, chAddr string) error

	// CollectInboundCh collects an inbound payment channel.
	// It takes a context, currency id and channel address as arguments.
	// It returns the error.
	CollectInboundCh(ctx context.Context, currencyID uint64, chAddr string) error

	/////////////////////////////
	/* Payment network related */
	/////////////////////////////

	// KeyStore gets the keystore of this manager.
	// It returns the key store.
	KeyStore() datastore.Datastore

	// CurrencyIDs gets the currency id supported by this manager.
	// It returns a list of currency IDs.
	CurrencyIDs() []uint64

	// GetRootAddress gets the root address of given currency id.
	// It takes a currency id as the argument.
	// It returns the root address and error if not existed.
	GetRootAddress(currencyID uint64) (string, error)

	// Reserve will reserve given amount for a period of time.
	// It takes a currency id, recipient address and amount as arguments.
	// It returns error.
	Reserve(currencyID uint64, toAddr string, amt *big.Int) error

	// Pay will pay given amount to a recipient.
	// It takes a context, a currency id, recipient address and amount as arguments.
	// It returns error.
	Pay(ctx context.Context, currencyID uint64, toAddr string, amt *big.Int) error

	// SearchPayOffer will ask all active peers for pay offers to given recipient with given amount.
	// It takes a context, a currency id, recipient address and total amount as arguments.
	// It returns a list of pay offers and error.
	SearchPayOffer(ctx context.Context, currencyID uint64, toAddr string, amt *big.Int) ([]payoffer.PayOffer, error)

	// ReserveWithPayOffer will reserve given amount for an offer.
	// It takes a pay offer as the argument.
	// It returns error.
	ReserveWithPayOffer(offer *payoffer.PayOffer) error

	// PayWithOffer will pay given amount using an offer.
	// It takes a context, a pay offer and amount as arguments.
	// It returns error.
	PayWithOffer(ctx context.Context, offer *payoffer.PayOffer, amt *big.Int) error

	// Receive will receive from origin addr.
	// It takes a currency id and origin address as arguments.
	// It return amount received and error.
	Receive(currencyID uint64, originAddr string) (*big.Int, error)

	////////////////////////////
	/* Paych servings related */
	////////////////////////////

	// StartServing starts serving a payment channel for others to use.
	// It takes a context, a currency id, to address, ppp (price per period), period as arguments.
	// It returns error.
	StartServing(currencyID uint64, toAddr string, ppp *big.Int, period *big.Int) error

	// ListServings lists all servings.
	// It takes a context as the argument.
	// It returns a map from currency id to list of active servings and error.
	ListServings(ctx context.Context) (map[uint64][]string, error)

	// InspectServing will check a given serving.
	// It takes a currency id, the to address as arguments.
	// It returns the ppp (price per period), period, expiration and error.
	InspectServing(currencyID uint64, toAddr string) (*big.Int, *big.Int, int64, error)

	// StopServing stops a serving for payment channel, it will put a channel into retirement before it automatically expires.
	// It takes a currency id, the to address as arguments.
	// It returns the error.
	StopServing(currencyID uint64, toAddr string) error

	// ForcePublishServings will force the manager to do a publication of current servings immediately.
	ForcePublishServings()

	//////////////////////////////////////////
	/* Payment network peer manager related */
	//////////////////////////////////////////

	// ListPeers is used to list all the peers.
	// It takes a context as the argument.
	// It returns a map from currency id to list of peers and error.
	ListPeers(ctx context.Context) (map[uint64][]string, error)

	// GetPeerInfo gets the information of a peer.
	// It takes a currency id, peer address as arguments.
	// It returns the peer id, a boolean indicate whether it is blocked, success count, failure count and error.
	GetPeerInfo(currencyID uint64, toAddr string) (peer.ID, bool, int, int, error)

	// BlockPeer is used to block a peer.
	// It takes a currency id, peer address to block as arguments.
	// It returns error.
	BlockPeer(currencyID uint64, toAddr string) error

	// UnblockPeer is used to unblock a peer.
	// It takes a currency id, peer address to unblock as arguments.
	// It returns error.
	UnblockPeer(currencyID uint64, toAddr string) error
}
