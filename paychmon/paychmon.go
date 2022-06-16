package paychmon

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
	"time"

	logging "github.com/ipfs/go-log"
)

// Logger
var log = logging.Logger("paychmonitor")

// PaychMonitor is used to track and monitor all payment channels.
// Once a payment channel is non-active, it will send request to payment manager for retirement.
type PaychMonitor interface {
	// Track starts tracking a payment channel.
	//
	// @input - context, if it is outbound channel, currency id, channel address, settlement time.
	//
	// @output - error.
	Track(ctx context.Context, outbound bool, currencyID byte, chAddr string, settlement time.Time) error

	// Check checks the current settlement of a payment channel.
	//
	// @input - context, if it is outbound channel, currency id, channel address.
	//
	// @output - last update time, settlement time, error.
	Check(ctx context.Context, outbound bool, currencyID byte, chAddr string) (time.Time, time.Time, error)

	// Renew renews a payment channel.
	//
	// @input - context, if it is outbound channel, currency id, channel address, new settlement time.
	//
	// @output - error.
	Renew(ctx context.Context, outbound bool, currencyID byte, chAddr string, newSettlement time.Time) error

	// Retire retires a payment channel.
	//
	// @input - context, if it is outbound channel, currency id, channel address.
	//
	// @output - error.
	Retire(ctx context.Context, outbound bool, currencyID byte, chAddr string) error
}
