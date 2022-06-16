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

import "time"

const (
	defaultCheckFreq = 1 * time.Hour
)

// Opts is the options for paych monitor.
type Opts struct {
	// The datastore path of the signer.
	Path string

	// The check frequency of payment channels.
	CheckFreq time.Duration
}
