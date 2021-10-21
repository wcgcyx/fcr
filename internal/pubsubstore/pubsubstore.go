package pubsubstore

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

// PubSubStore is used to store all the subscribers and all subscribings.
// It is used by the payment route publish/subscribe.
type PubSubStore interface {
	// AddSubscribing adds a subscribing to the store.
	// It takes a currency id, the peer id as arguments.
	// It returns error.
	AddSubscribing(currencyID uint64, pid peer.ID) error

	// RemoveSubscribing removes a subscribing from the store.
	// It takes a currency id, the peer id as arguments.
	// It returns error.
	RemoveSubscribing(currencyID uint64, pid peer.ID) error

	// AddSubscriber adds a subscriber to the store.
	// It takes a currency id, the peer id as arguments.
	// It returns error.
	AddSubscriber(currencyID uint64, pid peer.ID, ttl time.Duration) error

	// ListSubscribings lists all the subscribings.
	// It returns a map from currency id to a list of peer ids and error.
	ListSubscribings(ctx context.Context) (map[uint64][]peer.ID, error)

	// ListSubscribers lists all the subscribers.
	// It returns a map from currency id to a list of peer ids and error.
	ListSubscribers(ctx context.Context) (map[uint64][]peer.ID, error)
}
