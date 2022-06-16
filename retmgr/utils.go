package retmgr

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

import (
	"context"
	"math/big"
	"os"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/mdlstore"
)

// getOfferID is used to get the ID of a given offer.
//
// @input - offer.
//
// @output - offer id, error.
func getOfferID(offer fcroffer.PieceOffer) (string, error) {
	offerData, err := offer.Encode()
	if err != nil {
		return "", err
	}
	return getOfferIDRaw(offerData)
}

// getOfferIDRaw is used to get the ID of a given offer data.
//
// @input - offer data.
//
// @output - offer id, error.
func getOfferIDRaw(offerData []byte) (string, error) {
	pref := cid.Prefix{
		Version:  0,
		Codec:    cid.DagProtobuf,
		MhType:   multihash.SHA2_256,
		MhLength: -1, // Default length
	}
	id, err := pref.Sum(offerData)
	if err != nil {
		return "", err
	}
	return id.Hash().String(), nil
}

// Clean active out channels.
// Must be called after a write lock has been obtained.
func (mgr *RetrievalManager) cleanActiveOut() {
	toRemove := make([]string, 0)
	for key, state := range mgr.activeOutOfferID {
		if state.expiredAt.Before(time.Now()) {
			// Expired
			toRemove = append(toRemove, key)
		}
	}
	for _, key := range toRemove {
		delete(mgr.activeOutOfferID, key)
	}
	toRemove = make([]string, 0)
	for key, state := range mgr.activeOutReqID {
		if state.expiredAt.Before(time.Now()) {
			// Expired
			toRemove = append(toRemove, key)
		}
	}
	for _, key := range toRemove {
		delete(mgr.activeOutReqID, key)
	}
}

// Clean active in channels.
// Must be called after a write lock has been obtained.
func (mgr *RetrievalManager) cleanActiveIn() {
	toRemove := make([]string, 0)
	for key, state := range mgr.activeInOfferID {
		if state.expiredAt.Before(time.Now()) {
			// Expired
			toRemove = append(toRemove, key)
		}
	}
	for _, key := range toRemove {
		delete(mgr.activeInOfferID, key)
	}
}

// Load states from ds.
// Must be called in init stage.
//
// @input - context.
//
// @output - error.
func (mgr *RetrievalManager) loadStates(ctx context.Context) error {
	// Load the temp ds
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, mgr.tempPath)
	if err != nil {
		return err
	}
	defer ds.Shutdown(context.Background())
	txn, err := ds.NewTransaction(ctx, true)
	if err != nil {
		return err
	}
	defer txn.Discard(context.Background())
	children, err := txn.GetChildren(ctx)
	if err != nil {
		return err
	}
	for child := range children {
		val, err := txn.Get(ctx, child)
		if err != nil {
			return err
		}
		state := incomingRetrievalState{}
		err = state.decode(val)
		if err != nil {
			return err
		}
		// Extend expiration.
		state.expiredAt = time.Now().Add(state.inactivity)
		mgr.activeInOfferID[child] = &state
	}
	return nil
}

// Save states to the ds.
// Must be called in shutdown stage & after write lock is obtained.
//
// @input - context.
//
// @output - error.
func (mgr *RetrievalManager) saveStates(ctx context.Context) error {
	// Clear the temp ds.
	err := os.RemoveAll(mgr.tempPath)
	if err != nil {
		return err
	}
	err = os.Mkdir(mgr.tempPath, os.ModePerm)
	if err != nil {
		return err
	}
	// Load the temp ds.
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, mgr.tempPath)
	if err != nil {
		return err
	}
	defer ds.Shutdown(context.Background())
	txn, err := ds.NewTransaction(ctx, false)
	if err != nil {
		return err
	}
	defer txn.Discard(context.Background())
	for key, state := range mgr.activeInOfferID {
		// As we are shutting down, we will skip one block for user.
		state.index += 1
		state.paymentRequired = big.NewInt(0)
		val, err := state.encode()
		if err != nil {
			return err
		}
		err = txn.Put(ctx, val, key)
		if err != nil {
			return err
		}
	}
	return txn.Commit(ctx)
}
