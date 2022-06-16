package renewmgr

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
var log = logging.Logger("renewmgr")

// RenewManager is the interface for a manager that is used to manage all renew period policies.
// If a policy has a duration of 0, it means the policy is to reject.
type RenewManager interface {
	// SetDefaultPolicy sets the default renew policy.
	//
	// @input - context, currency id, renew duration.
	//
	// @output - error.
	SetDefaultPolicy(ctx context.Context, currencyID byte, duration time.Duration) error

	// GetDefaultPolicy gets the default renew policy.
	//
	// @input - context, currency id.
	//
	// @output - renew duration, error.
	GetDefaultPolicy(ctx context.Context, currencyID byte) (time.Duration, error)

	// RemoveDefaultPolicy removes the default renew policy.
	//
	// @input - context, currency id.
	//
	// @output - error.
	RemoveDefaultPolicy(ctx context.Context, currencyID byte) error

	// SetSenderPolicy sets a renew policy for a given sender.
	//
	// @input - context, currency id, payment channel sender, renew duration.
	//
	// @output - error.
	SetSenderPolicy(ctx context.Context, currencyID byte, fromAddr string, duration time.Duration) error

	// GetSenderPolicy gets the policy for a given currency id and sender pair.
	//
	// @input - context, currency id, payment channel sender.
	//
	// @output - renew duration, error.
	GetSenderPolicy(ctx context.Context, currencyID byte, fromAddr string) (time.Duration, error)

	// RemoveSenderPolicy removes a policy.
	//
	// @input - context, currency id, payment channel sender.
	//
	// @output - error.
	RemoveSenderPolicy(ctx context.Context, currencyID byte, fromAddr string) error

	// SetPaychPolicy sets a renew policy for a given paych.
	//
	// @input - context, currency id, payment channel sender, paych addr, renew duration.
	//
	// @output - error.
	SetPaychPolicy(ctx context.Context, currencyID byte, fromAddr string, chAddr string, duration time.Duration) error

	// GetPaychPolicy gets the policy for a given currency id and paych pair.
	//
	// @input - context, currency id, payment channel sender, paych addr.
	//
	// @output - renew duration, error.
	GetPaychPolicy(ctx context.Context, currencyID byte, fromAddr string, chAddr string) (time.Duration, error)

	// RemoveSenderPolicy removes a policy.
	//
	// @input - context, currency id, payment channel sender, paych addr.
	//
	// @output - error.
	RemovePaychPolicy(ctx context.Context, currencyID byte, fromAddr string, chAddr string) error

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

	// ListSenders lists all paychs with a policy set.
	//
	// @input - context, currency id, sender addr.
	//
	// @output - paych addr chan out (could be default policy id or paych addr), error chan out.
	ListPaychs(ctx context.Context, currencyID byte, fromAddr string) (<-chan string, <-chan error)
}
