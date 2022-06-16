package paychproto

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

import "time"

const (
	defaultIOTimeout   = 1 * time.Minute
	defaultOpTimeout   = 30 * time.Second
	defaultOfferExpiry = 5 * time.Minute
	defaultRenewWindow = 90
)

// Opts is the options for paych protocol.
type Opts struct {
	// The timeout for every network io operation.
	IOTimeout time.Duration

	// The timeout for every internal operation.
	OpTimeout time.Duration

	// The offer expiry time.
	OfferExpiry time.Duration

	// The renew window.
	RenewWindow uint64
}
