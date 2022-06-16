package payproto

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
	defaultIOTimeout    = 1 * time.Minute
	defaultOpTimeout    = 30 * time.Second
	defaultCleanFreq    = 30 * time.Minute
	defaultCleanTimeout = 10 * time.Second
)

// Opts is the options for paych protocol.
type Opts struct {
	// The datastore path.
	Path string

	// The timeout for every network io operation.
	IOTimeout time.Duration

	// The timeout for every internal operation.
	OpTimeout time.Duration

	// The cleaning frequency.
	CleanFreq time.Duration

	// The cleaning timeout (for locking).
	CleanTimeout time.Duration
}
