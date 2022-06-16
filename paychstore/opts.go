package paychstore

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
	defaultDSTimeout = 5 * time.Second
	defaultDSRetry   = 5
)

// Opts is the options for the paychstore.
type Opts struct {
	// The datastore path of the paychstore.
	Path string

	// Timeout for DS operation.
	DSTimeout time.Duration

	// Retry for DS operation.
	DSRetry uint64
}
