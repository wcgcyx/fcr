package routestore

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
	defaultCleanFreq    = 2 * time.Hour
	defaultCleanTimeout = 30 * time.Second
	defaultMaxHopFIL    = uint64(5)
)

// Opts is the options for route store.
type Opts struct {
	// The datastore path of the route store.
	Path string

	// Cleaning frequency.
	CleanFreq time.Duration

	// Cleaning timeout.
	CleanTimeout time.Duration

	// The max hop for FIL.
	MaxHopFIL uint64
}
