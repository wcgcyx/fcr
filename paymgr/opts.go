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

import "time"

const (
	defaultCacheSyncFreq    = 1 * time.Minute
	defaultResCleanFreq     = 10 * time.Minute
	defaultResCleanTimeout  = 10 * time.Second
	defaultPeerCleanFreq    = 1 * time.Hour
	defaultPeerCleanTimeout = 10 * time.Second
)

// Opts is the options for payment manager.
type Opts struct {
	// The datastore path of the payment manager.
	Path string

	// The cache and store sync frequency.
	CacheSyncFreq time.Duration

	// The reservation cleaning frequency.
	ResCleanFreq time.Duration

	// The reservation cleaning timeout (for locking).
	ResCleanTimeout time.Duration

	// The peer cleaning frequency.
	PeerCleanFreq time.Duration

	// The peer cleaning timeout (for locking).
	PeerCleanTimeout time.Duration
}
