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
	"math/big"
	"time"

	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/paychstate"
)

// cachedPaychState is the cached payment channel state.
type cachedPaychState struct {
	// The channel state
	state *paychstate.State
	// Current reserved amount
	reservedAmt *big.Int
	// reserve nonce, also used as the reservation id.
	reservedNonce uint64
	// Map from reservation id -> reservation
	reservations map[uint64]*reservation
}

// reservation is the reservation state.
type reservation struct {
	// For calculating petty/surcharge amount received.
	inboundPPP        *big.Int
	inboundPeriod     *big.Int
	pettyAmtReceived  *big.Int
	surchargeReceived *big.Int
	// For calculating petty/surcharge amount to pay.
	outboundPPP    *big.Int
	outboundPeriod *big.Int
	pettyAmtPaid   *big.Int
	surchargePaid  *big.Int
	// The remaining gross amount in this reservation. (remain >= pettyAmtPaid + surchargePaid).
	remain *big.Int
	// Time related.
	lastAccessed time.Time
	expiration   time.Time
	inactivity   time.Duration
	// Linked offer
	offer *fcroffer.PayOffer
}
