package trans

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

const (
	defaultFilecoinConfidence = uint64(4)
)

// Opts is the options for the transactor.
type Opts struct {
	/* Filecoin related */
	// Boolean indicate if Filecoin is enabled.
	FilecoinEnabled bool

	// Filecoin network API address.
	FilecoinAPI string

	// Filecoin API auth token (empty if using Infura).
	FilecoinAuthToken string

	// Filecoin confidence epoch.
	FilecoinConfidence *uint64
}
