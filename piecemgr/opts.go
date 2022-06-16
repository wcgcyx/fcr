package piecemgr

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

//  Opts is the options for piece manager.
type Opts struct {
	// The datastore path of the piece info store.
	PsPath string
	// The datastore path of the block store.
	BsPath string
	// The datastore path of the block reference store.
	BrefsPath string
}
