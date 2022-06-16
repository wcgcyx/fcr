package settlemgr

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

const (
	DefaultPolicyID = "default"
)

// Logger
var log = logging.Logger("settlemgr")

// SettlementManager is the interface for a manager that is used to manage all settlement period policies.
// If a policy has a duration of 0, it means the policy is to reject.
type SettlementManager interface {
	// SetDefaultPolicy sets the default settlement policy.
	//
	// @input - context, currency id, settlement duration.
	//
	// @output - error.
	SetDefaultPolicy(ctx context.Context, currencyID byte, duration time.Duration) error

	// GetDefaultPolicy gets the default settlement policy.
	//
	// @input - context, currency id.
	//
	// @output - settlement duration, error.
	GetDefaultPolicy(ctx context.Context, currencyID byte) (time.Duration, error)

	// RemoveDefaultPolicy removes the default settlement policy.
	//
	// @input - context, currency id.
	//
	// @output - error.
	RemoveDefaultPolicy(ctx context.Context, currencyID byte) error

	// SetSenderPolicy sets a settlement policy for a given sender.
	//
	// @input - context, currency id, payment channel sender, settlement duration.
	//
	// @output - error.
	SetSenderPolicy(ctx context.Context, currencyID byte, fromAddr string, duration time.Duration) error

	// GetSenderPolicy gets the policy for a given currency id and sender pair.
	//
	// @input - context, currency id, payment channel sender.
	//
	// @output - settlement duration, error.
	GetSenderPolicy(ctx context.Context, currencyID byte, fromAddr string) (time.Duration, error)

	// RemoveSenderPolicy removes a policy.
	//
	// @input - context, currency id, payment channel sender.
	//
	// @output - error.
	RemoveSenderPolicy(ctx context.Context, currencyID byte, fromAddr string) error

	// ListCurrencyIDs lists all currency ids.
	//
	// @input - context.
	//
	// @output - currency id chan out, error chan out.
	ListCurrencyIDs(ctx context.Context) (<-chan byte, <-chan error)

	// ListSenders lists all senders with a policy set.
	//
	// @input - context, currency id.
	//
	// @output - sender chan out (could be default policy id or sender addr), error chan out.
	ListSenders(ctx context.Context, currencyID byte) (<-chan string, <-chan error)
}
