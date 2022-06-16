package routeproto

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
	defaultPublishFreq = 1 * time.Hour
	defaultRouteExpiry = 90 * time.Minute
	defaultPublishWait = 1 * time.Minute
)

// Opts is the options for route protocol.
type Opts struct {
	// The timeout for every network io operation.
	IOTimeout time.Duration

	// The timeout for every internal operation.
	OpTimeout time.Duration

	// The publish frequency
	PublishFreq time.Duration

	// The route expiry
	RouteExpiry time.Duration

	// Initial wait time for publish.
	PublishWait time.Duration
}
