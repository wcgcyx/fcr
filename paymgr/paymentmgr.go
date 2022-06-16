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
	"math/big"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/wcgcyx/fcr/fcroffer"
)

// Logger
var log = logging.Logger("paymentmgr")

// PaymentManager is an interface for a payment manager that handles all payment related functionalities.
type PaymentManager interface {
	/*
		Core Payment APIs.
	*/

	// Reserve is used to reserve a certain amount.
	//
	// @input - context, currency id, recipient address, petty amount required, whether reservation requires served channel, optional received offer, first expiration time, subsequent allowed inactivity time, peer addr who requests the reservation.
	//
	// @output - reserved inbound price-per-byte, reserved inbound period, reserved channel address, reservation id, error.
	Reserve(ctx context.Context, currencyID byte, toAddr string, pettyAmtRequired *big.Int, servedRequired bool, receivedOffer *fcroffer.PayOffer, expiration time.Time, inactivity time.Duration, peerAddr string) (*big.Int, *big.Int, string, uint64, error)

	// Pay is used to make a payment.
	//
	// @input - context, currency id, recipient address, reserved channel address, reservation id, received gross payment to drive this payment, petty amount required.
	//
	// @output - voucher, potential linked offer, function to commit & record access time, function to revert & record network loss, error.
	Pay(ctx context.Context, currencyID byte, toAddr string, chAddr string, resID uint64, grossAmtReceived *big.Int, pettyAmtRequired *big.Int) (string, *fcroffer.PayOffer, func(time.Time), func(string), error)

	// Receive is used to receive a payment in a voucher.
	//
	// @input - context, currency id, voucher.
	//
	// @output - received gross amount, function to commit, function to revert, the lastest voucher recorded previously if network loss presents, error.
	Receive(ctx context.Context, currencyID byte, voucher string) (*big.Int, func(), func(), string, error)

	/*
		Reservation APIs.
	*/

	// ReserveForSelf is used to reserve a certain amount for self to spend for configured time.
	//
	// @input - context, currency id, recipient address, petty amount, first expiration time, subsequent allowed inactivity time.
	//
	// @output - reserved channel address, reservation id, error.
	ReserveForSelf(ctx context.Context, currencyID byte, toAddr string, pettyAmt *big.Int, expiration time.Time, inactivity time.Duration) (string, uint64, error)

	// ReserveForSelfWithOffer is used to reserve a certain amount for self to spend based on a received offer.
	//
	// @input - context, received offer.
	//
	// @output - reserved channel address, reservation id, error.
	ReserveForSelfWithOffer(ctx context.Context, receivedOffer fcroffer.PayOffer) (string, uint64, error)

	// ReserveForOthersIntermediate is used to reserve some amount for others as an intermediate node based on required inbound surcharge and a received offer, peer addr who requests the reservation.
	//
	// @input - context, received offer.
	//
	// @output - reserved inbound price-per-byte, reserved inbound period, reserved channel address, reservation id, error.
	ReserveForOthersIntermediate(ctx context.Context, receivedOffer fcroffer.PayOffer, peerAddr string) (*big.Int, *big.Int, string, uint64, error)

	// ReserveForOthersFinal is used to reserve some amount for others as a final node based on required inbound surcharge, peer addr who requests the reservation.
	//
	// @input - context, currency id, recipient address, petty amount, first expiration time, subsequent allowed inactivity time.
	//
	// @output - reserved inbound price-per-byte, reserved inbound period, reserved channel address, reservation id, error.
	ReserveForOthersFinal(ctx context.Context, currencyID byte, toAddr string, pettyAmt *big.Int, expiration time.Time, inactivity time.Duration, peerAddr string) (*big.Int, *big.Int, string, uint64, error)

	/*
		Payment APIs.
	*/

	// PayForSelf is used to make a payment for self.
	//
	// @input - context, currency id, recipient address, reserved channel address, reservation id, petty amount required.
	//
	// @output - voucher, potential linked offer, function to commit & record access time, function to revert & record network loss, error.
	PayForSelf(ctx context.Context, currencyID byte, toAddr string, chAddr string, resID uint64, pettyAmtRequired *big.Int) (string, *fcroffer.PayOffer, func(time.Time), func(string), error)

	// PayForOthers is used to make a payment for others.
	//
	// @input - context, currency id, recipient address, reserved channel address, reservation id, gross amount received, petty amount required.
	//
	// @output - voucher, potential linked offer, function to commit & record access time, function to revert & record network loss, error.
	PayForOthers(ctx context.Context, currencyID byte, toAddr string, chAddr string, resID uint64, grossAmtReceived *big.Int, pettyAmtRequired *big.Int) (string, *fcroffer.PayOffer, func(time.Time), func(string), error)

	/*
		Channel APIs.
	*/

	// AddInboundChannel is used to add an inbound payment channel.
	//
	// @input - context, currency id, from address, channel address.
	//
	// @output - error.
	AddInboundChannel(ctx context.Context, currencyID byte, fromAddr string, chAddr string) error

	// RetireInboundChannel is used to retire an inbound payment channel.
	//
	// @input - context, currency id, from address, channel address.
	//
	// @output - error.
	RetireInboundChannel(ctx context.Context, currencyID byte, fromAddr string, chAddr string) error

	// AddOutboundChannel is used to add an outbound payment channel.
	//
	// @input - context, currency id, recipient address, channel address.
	//
	// @output - error.
	AddOutboundChannel(ctx context.Context, currencyID byte, toAddr string, chAddr string) error

	// RetireOutboundChannel is used to retire an outbound payment channel.
	//
	// @input - context, currency id, recipient address, channel address.
	//
	// @output - error.
	RetireOutboundChannel(ctx context.Context, currencyID byte, toAddr string, chAddr string) error

	// UpdateOutboundChannelBalance is used to update the balance of an outbound payment channel.
	//
	// @input - context, currency id, recipient address, channel address.
	//
	// @output - error.
	UpdateOutboundChannelBalance(ctx context.Context, currencyID byte, toAddr string, chAddr string) error

	// BearNetworkLoss is used to bear and ignore the network loss of a given channel.
	//
	// @input - context, currency id, recipient address, channel address.
	//
	// @output - error.
	BearNetworkLoss(ctx context.Context, currencyID byte, toAddr string, chAddr string) error
}
