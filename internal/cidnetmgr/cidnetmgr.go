package cidnetmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"math/big"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wcgcyx/fcr/internal/cidoffer"
	"github.com/wcgcyx/fcr/internal/payoffer"
)

// CIDNetworkManager is used to manage the cid network.
type CIDNetworkManager interface {
	///////////////////////////
	/* Piece manager related */
	///////////////////////////

	// Import imports a file.
	// It takes a context and a path as arguments.
	// It returns the cid imported and error.
	Import(ctx context.Context, path string) (cid.Cid, error)

	// ImportCar imports a car file.
	// It takes a context and a path as arguments.
	// It returns the cid imported and error.
	ImportCar(ctx context.Context, path string) (cid.Cid, error)

	// ImportSector imports an filecoin lotus unsealed sector copy.
	// It takes a context and a path, a boolean indicating whether to keep a copy as arguments.
	// It returns the a list of cids imported and error.
	ImportSector(ctx context.Context, path string, copy bool) ([]cid.Cid, error)

	// ListImported lists all imported pieces.
	// It takes a context as the argument.
	// It returns a list of cids imported and error.
	ListImported(ctx context.Context) ([]cid.Cid, error)

	// Inspect inspects a piece.
	// It takes a cid as the argument.
	// It returns the path, the index, the size, a boolean indicating whether a copy is kept and error.
	Inspect(id cid.Cid) (string, int, int64, bool, error)

	// Remove removes a piece.
	// It takes a cid as the argument.
	// It returns error.
	Remove(id cid.Cid) error

	/////////////////////////
	/* CID network related */
	/////////////////////////

	// QueryOffer queries a given peer for offer.
	// It takes a context, a peer id, a currency id, the root id as arguments.
	// It returns a cid offer and error.
	QueryOffer(ctx context.Context, pid peer.ID, currencyID uint64, id cid.Cid) (*cidoffer.CIDOffer, error)

	// SearchOffers searches a list of offers from peers using DHT.
	// It takes a context, a currency id, the root id and maximum offer as arguments.
	// It returns a list of cid offers.
	SearchOffers(ctx context.Context, currencyID uint64, id cid.Cid, max int) []cidoffer.CIDOffer

	// RetrieveFromCache retrieves a content based on given cid from cache.
	// It takes a context, a cid and outpath as arguments.
	// It returns boolean indicating if found in cache and error.
	RetrieveFromCache(ctx context.Context, id cid.Cid, outPath string) (bool, error)

	// Retrieve retrieves a content based on an offer to a given path.
	// It takes a context, a cid offer, a pay offer (can be nil if retrieve directly from connected peers), an outpath as arguments.
	// It returns error.
	Retrieve(ctx context.Context, cidOffer cidoffer.CIDOffer, payOffer *payoffer.PayOffer, outPath string) error

	// GetRetrievalCacheSize gets the current cache DB size in bytes.
	// It returns the size and error.
	GetRetrievalCacheSize() (uint64, error)

	// CleanRetrievalCache will clean the cache DB.
	// It returns the error.
	CleanRetrievalCache() error

	////////////////////////////
	/* Paych servings related */
	////////////////////////////

	// StartServing starts serving a cid for others to use.
	// It takes a context, a currency id, the cid, ppn (price per byte) as arguments.
	// It returns error.
	StartServing(ctx context.Context, currencyID uint64, id cid.Cid, ppb *big.Int) error

	// ListServings lists all servings.
	// It takes a context as the argument.
	// It returns a map from currency id to list of active servings and error.
	ListServings(ctx context.Context) (map[uint64][]cid.Cid, error)

	// InspectServing will check a given serving.
	// It takes a currency id, the cid as arguments.
	// It returns the ppb (price per byte), expiration and error.
	InspectServing(currencyID uint64, id cid.Cid) (*big.Int, int64, error)

	// StopServing stops a serving for cid, it will put a cid into retirement before it automatically expires.
	// It takes a currency id, the cid as arguments.
	// It returns the error.
	StopServing(currencyID uint64, id cid.Cid) error

	// ForcePublishServings will force the manager to do a publication of current servings immediately.
	ForcePublishServings()

	// ListInUseServings will list all the in use servings.
	ListInUseServings(ctx context.Context) ([]cid.Cid, error)

	//////////////////////////////////////
	/* CID network peer manager related */
	//////////////////////////////////////

	// ListPeers is used to list all the peers.
	// It takes a context as the argument.
	// It returns a map from currency id to list of peers and error.
	ListPeers(ctx context.Context) (map[uint64][]string, error)

	// GetPeerInfo gets the information of a peer.
	// It takes a currency id, peer address as arguments.
	// It returns the peer id, a boolean indicate whether it is blocked, success count, failure count and error.
	GetPeerInfo(currencyID uint64, toAddr string) (peer.ID, bool, int, int, error)

	// BlockPeer is used to block a peer.
	// It takes a currency id, peer address to block as arguments.
	// It returns error.
	BlockPeer(currencyID uint64, toAddr string) error

	// UnblockPeer is used to unblock a peer.
	// It takes a currency id, peer address to unblock as arguments.
	// It returns error.
	UnblockPeer(currencyID uint64, toAddr string) error
}
