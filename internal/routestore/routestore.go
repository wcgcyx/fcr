package routestore

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import "time"

// RouteStore is used to store all the possible payment routes originating from root address.
// It is used to store all published routes from subscribed peers.
// It is also used by the route publisher to publish all possible routes originating frmo root to subscribers.
type RouteStore interface {
	// MaxHop obtain the maximum hop supported for a given currency id.
	// It taktes a currency id as the argument.
	// It returns the max hop count.
	MaxHop(currencyID uint64) int

	// AddDirectLink will add a direct link to the root address for a given currency id.
	// It takes a currency id and the address of a direct link as arguments.
	// It returns the error.
	AddDirectLink(currencyID uint64, toAddr string) error

	// RemoveDirectLink will remove a direct link from the root address.
	// It takes a currency id and the address of the direct link to remove as arguments.
	// It returns the error.
	RemoveDirectLink(currencyID uint64, toAddr string) error

	// AddRoute will add a route to the store. Once added, the route will
	// add the root in front.
	// It takes a currency id and the route to add as arguments.
	// It returns the error.
	AddRoute(currencyID uint64, route []string, ttl time.Duration) error

	// GetRoutesTo will get all the routes that have the given destination.
	// It takes a currency id and the destination address as arguments.
	// It returns a list of routes.
	// It returns the error.
	GetRoutesTo(currencyID uint64, toAddr string) ([][]string, error)

	// GetRoutesFrom will get all the routes that start with the given direct link (from address).
	// It takes a currency id and the from address as arguments.
	// It returns a list of routes and error.
	GetRoutesFrom(currencyID uint64, toAddr string) ([][]string, error)
}
