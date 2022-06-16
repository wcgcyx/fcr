package peermgr

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
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	dsquery "github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wcgcyx/fcr/mdlstore"
)

// PeerManagerImpl is the implementation of PeerManager interface.
type PeerManagerImpl struct {
	// Datastore
	ds mdlstore.MultiDimLockableStore

	// Shutdown function.
	shutdown func()
}

// NewPeerManagerImpl creates a new PeerManager.
//
// @input - context, options.
//
// @output - peer manager, error.
func NewPeerManagerImpl(ctx context.Context, opts Opts) (*PeerManagerImpl, error) {
	log.Infof("Start peer manager...")
	// Create new store.
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, opts.Path)
	if err != nil {
		log.Errorf("Fail to open mdlstore: %v", err.Error())
		return nil, err
	}
	defer func() {
		if err != nil {
			log.Infof("Fail to start peer manager, close ds...")
			err = ds.Shutdown(context.Background())
			if err != nil {
				log.Errorf("Fail to close ds after failing to start peer manager: %v", err.Error())
			}
		}
	}()
	mgr := &PeerManagerImpl{
		ds: ds,
	}
	mgr.shutdown = func() {
		log.Infof("Stop datastore...")
		err := ds.Shutdown(context.Background())
		if err != nil {
			log.Errorf("Fail to stop datastore: %v", err.Error())
		}
	}
	return mgr, nil
}

// Shutdown safely shuts down the component.
func (mgr *PeerManagerImpl) Shutdown() {
	log.Infof("Start shutdown...")
	mgr.shutdown()
}

// AddPeer adds a peer. If there is an existing peer presented, overwrite the peer addr.
//
// @input - context, currency id, peer wallet address, peer p2p address.
//
// @output - error.
func (mgr *PeerManagerImpl) AddPeer(ctx context.Context, currencyID byte, toAddr string, pi peer.AddrInfo) error {
	log.Debugf("Add peer for %v-%v with %v", currencyID, toAddr, pi)
	// Check if exists.
	release, err := mgr.ds.RLock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}

	txn, err := mgr.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, toAddr)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	if exists {
		// Check if peer address needs to be updated.
		log.Debugf("Check if peer address for %v-%v need to be updated", currencyID, toAddr)
		val, err := txn.Get(ctx, currencyID, toAddr)
		if err != nil {
			release()
			log.Warnf("Fail to read ds value for %v-%v: %v", currencyID, toAddr, err.Error())
			return err
		}
		_, piOld, _, err := decDSVal(val)
		if err != nil {
			release()
			log.Errorf("Fail to decode ds value %v: %v", val, err.Error())
			return err
		}
		if piOld.String() == pi.String() {
			// Fail silently if no need to update.
			release()
			log.Debugf("No need to update")
			return nil
		}
		log.Debugf("Peer addr info needs to be updated, old %v, new %v", piOld.String(), pi.String())
	}
	release()

	release, err = mgr.ds.Lock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	defer release()

	txn, err = mgr.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	// Check again
	exists, err = txn.Has(ctx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	if exists {
		log.Debugf("Check if peer address for %v-%v need to be updated", currencyID, toAddr)
		// Check if peer address needs to be updated.
		val, err := txn.Get(ctx, currencyID, toAddr)
		if err != nil {
			log.Warnf("Fail to read ds value for %v-%v: %v", currencyID, toAddr, err.Error())
			return err
		}
		blocked, piOld, recID, err := decDSVal(val)
		if err != nil {
			log.Errorf("Fail to decode ds value %v: %v", val, err.Error())
			return err
		}
		if piOld.String() == pi.String() {
			// Fail silently if no need to update.
			log.Debugf("No need to update")
			return nil
		}
		val, err = encDSVal(blocked, pi, recID)
		if err != nil {
			log.Errorf("Fail to encode ds value for %v %v %v: %v", blocked, pi.String(), recID, err.Error())
			return err
		}
		err = txn.Put(ctx, val, currencyID, toAddr)
		if err != nil {
			log.Warnf("Fail to put ds value %v for %v-%v: %v", val, currencyID, toAddr, err.Error())
			return err
		}
	} else {
		log.Debugf("Peer does not exist for %v-%v", currencyID, toAddr)
		val, err := encDSVal(false, pi, big.NewInt(0))
		if err != nil {
			log.Errorf("Fail to encode value for %v %v %v: %v", false, pi.String(), big.NewInt(0), err.Error())
			return err
		}
		err = txn.Put(ctx, val, currencyID, toAddr)
		if err != nil {
			log.Warnf("Fail to put ds value %v for %v-%v: %v", val, currencyID, toAddr, err.Error())
			return err
		}
	}
	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// HasPeer checks if a given peer exists.
//
// @input - context, currency id, peer wallet address.
//
// @output - if exists, error.
func (mgr *PeerManagerImpl) HasPeer(ctx context.Context, currencyID byte, toAddr string) (bool, error) {
	log.Debugf("Check if contains peer %v-%v", currencyID, toAddr)
	// Check if exists.
	release, err := mgr.ds.RLock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return false, err
	}
	defer release()

	txn, err := mgr.ds.NewTransaction(ctx, true)
	if err != nil {
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return false, err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, toAddr, err.Error())
		return false, err
	}
	return exists, nil
}

// RemovePeer removes a peer from the manager.
//
// @input - context, currency id, peer wallet address.
//
// @output - error.
func (mgr *PeerManagerImpl) RemovePeer(ctx context.Context, currencyID byte, toAddr string) error {
	log.Debugf("Remove peer for %v-%v", currencyID, toAddr)
	// Check if exists.
	release, err := mgr.ds.RLock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}

	txn, err := mgr.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, toAddr)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	if !exists {
		release()
		log.Debugf("Peer %v-%v doesn not exist", currencyID, toAddr)
		return fmt.Errorf("peer %v-%v does not exist", currencyID, toAddr)
	}
	release()

	release, err = mgr.ds.Lock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	defer release()

	txn, err = mgr.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	// Check again.
	exists, err = txn.Has(ctx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	if !exists {
		log.Debugf("Peer %v-%v doesn not exist", currencyID, toAddr)
		return fmt.Errorf("peer %v-%v does not exist", currencyID, toAddr)
	}

	// The record can be very large, so instead of loading all children into memory, use query.
	tempRes, err := txn.Query(ctx, mdlstore.Query{KeysOnly: true}, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to query for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	defer tempRes.Results.Close()
	for {
		e, ok := tempRes.Results.NextSync()
		if !ok {
			log.Debugf("Finished all query results")
			break
		}
		if e.Error != nil {
			log.Warnf("Fail to get next entry: %v", e.Error.Error())
			return e.Error
		}
		temp := strings.Split(e.Key[1:], mdlstore.DatastoreKeySeperator)
		if len(temp) != 3 {
			log.Debugf("Invalid key detected: %v", e.Key[1:])
			continue
		}
		err = txn.Delete(ctx, temp[0], temp[1], temp[2])
		if err != nil {
			log.Warnf("Fail to delete %v", temp)
			continue
		}
	}

	// Remove root key
	err = txn.Delete(ctx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to remove root key %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction")
		return err
	}
	return nil
}

// PeerAddr gets the peer addr by given wallet address.
//
// @input - context, currency id, peer wallet address.
//
// @output - peer p2p address, error.
func (mgr *PeerManagerImpl) PeerAddr(ctx context.Context, currencyID byte, toAddr string) (peer.AddrInfo, error) {
	log.Debugf("Get peer addr for %v-%v", currencyID, toAddr)
	release, err := mgr.ds.RLock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return peer.AddrInfo{}, err
	}
	defer release()

	txn, err := mgr.ds.NewTransaction(ctx, true)
	if err != nil {
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return peer.AddrInfo{}, err
	}
	defer txn.Discard(context.Background())

	val, err := txn.Get(ctx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to read ds value for %v-%v: %v", currencyID, toAddr, err.Error())
		return peer.AddrInfo{}, err
	}
	_, pi, _, err := decDSVal(val)
	if err != nil {
		log.Errorf("Fail to decode ds value %v: %v", val, err.Error())
		return peer.AddrInfo{}, err
	}
	return pi, nil
}

// IsBlocked checks if given peer is currently blocked.
//
// @input - context, currency id, peer wallet address.
//
// @output - boolean indicating whether it is blocked, error.
func (mgr *PeerManagerImpl) IsBlocked(ctx context.Context, currencyID byte, toAddr string) (bool, error) {
	log.Debugf("Check if peer %v-%v is blocked", currencyID, toAddr)
	release, err := mgr.ds.RLock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return false, err
	}
	defer release()

	txn, err := mgr.ds.NewTransaction(ctx, true)
	if err != nil {
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return false, err
	}
	defer txn.Discard(context.Background())

	val, err := txn.Get(ctx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to read ds value for %v-%v: %v", currencyID, toAddr, err.Error())
		return false, err
	}
	blocked, _, _, err := decDSVal(val)
	if err != nil {
		log.Errorf("Fail to decode ds value %v: %v", val, err.Error())
		return false, err
	}
	return blocked, nil
}

// BlockPeer blocks a peer.
//
// @input - context, currency id, peer wallet address.
//
// @output - error.
func (mgr *PeerManagerImpl) BlockPeer(ctx context.Context, currencyID byte, toAddr string) error {
	log.Debugf("Block peer %v-%v", currencyID, toAddr)
	release, err := mgr.ds.RLock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}

	txn, err := mgr.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	val, err := txn.Get(ctx, currencyID, toAddr)
	if err != nil {
		release()
		log.Warnf("Fail to read ds value for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	blocked, _, _, err := decDSVal(val)
	if err != nil {
		release()
		log.Errorf("Fail to decode ds value %v: %v", val, err.Error())
		return err
	}
	if blocked {
		release()
		log.Debugf("Peer for %v-%v is already blocked", currencyID, toAddr)
		return fmt.Errorf("peer is already blocked")
	}
	release()

	release, err = mgr.ds.Lock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	defer release()

	txn, err = mgr.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	// Check again
	val, err = txn.Get(ctx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to read ds value for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	blocked, pi, recID, err := decDSVal(val)
	if err != nil {
		log.Errorf("Fail to decode ds value %v: %v", val, err.Error())
		return err
	}
	if blocked {
		log.Debugf("Peer for %v-%v is already blocked", currencyID, toAddr)
		return fmt.Errorf("peer is already blocked")
	}
	val, err = encDSVal(true, pi, recID)
	if err != nil {
		log.Errorf("Fail to encode ds value for %v %v %v: %v", true, pi.String(), recID, err.Error())
		return err
	}
	err = txn.Put(ctx, val, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v-%v: %v", val, currencyID, toAddr, err.Error())
		return err
	}
	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// UnblockPeer unblocks a peer.
//
// @input - context, currency id, peer wallet address.
//
// @output - error.
func (mgr *PeerManagerImpl) UnblockPeer(ctx context.Context, currencyID byte, toAddr string) error {
	log.Debugf("Unblock peer %v-%v", currencyID, toAddr)
	release, err := mgr.ds.RLock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}

	txn, err := mgr.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	val, err := txn.Get(ctx, currencyID, toAddr)
	if err != nil {
		release()
		log.Warnf("Fail to read ds value for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	blocked, _, _, err := decDSVal(val)
	if err != nil {
		release()
		log.Errorf("Fail to decode ds value %v: %v", val, err.Error())
		return err
	}
	if !blocked {
		release()
		log.Debugf("Peer for %v-%v is already unblocked", currencyID, toAddr)
		return fmt.Errorf("peer is already unblocked")
	}
	release()

	release, err = mgr.ds.Lock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	defer release()

	txn, err = mgr.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	// Check again
	val, err = txn.Get(ctx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to read ds value for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	blocked, pi, recID, err := decDSVal(val)
	if err != nil {
		log.Errorf("Fail to decode ds value %v: %v", val, err.Error())
		return err
	}
	if !blocked {
		log.Debugf("Peer for %v-%v is already unblocked", currencyID, toAddr)
		return fmt.Errorf("peer is already unblocked")
	}
	val, err = encDSVal(false, pi, recID)
	if err != nil {
		log.Errorf("Fail to encode ds value for %v %v %v: %v", false, pi.String(), recID, err.Error())
		return err
	}
	err = txn.Put(ctx, val, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v-%v: %v", val, currencyID, toAddr, err.Error())
		return err
	}
	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// ListCurrencyIDs lists all currencies.
//
// @input - context.
//
// @output - currency chan out, error chan out.
func (mgr *PeerManagerImpl) ListCurrencyIDs(ctx context.Context) (<-chan byte, <-chan error) {
	out := make(chan byte, 32)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Listing currency ids")
		defer func() {
			close(out)
			close(errChan)
		}()
		release, err := mgr.ds.RLock(ctx)
		if err != nil {
			errChan <- err
			log.Debugf("Fail to obtain read lock for root: %v", err.Error())
			return
		}

		txn, err := mgr.ds.NewTransaction(ctx, true)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to start a new transaction: %v", err.Error())
			return
		}
		defer txn.Discard(context.Background())

		children, err := txn.GetChildren(ctx)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to get children for root: %v", err.Error())
			return
		}
		release()

		for child := range children {
			currencyID, err := strconv.Atoi(child)
			if err != nil {
				log.Errorf("Fail to convert child key %v to currency id: %v", child, err.Error())
				errChan <- err
				return
			}
			select {
			case out <- byte(currencyID):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errChan
}

// ListPeers lists all peers stored by the manager.
//
// @input - context, currency id.
//
// @output - peer wallet address channel out, error channel out.
func (mgr *PeerManagerImpl) ListPeers(ctx context.Context, currencyID byte) (<-chan string, <-chan error) {
	out := make(chan string, 128)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Listing peers for %v", currencyID)
		defer func() {
			close(out)
			close(errChan)
		}()
		release, err := mgr.ds.RLock(ctx, currencyID)
		if err != nil {
			errChan <- err
			log.Debugf("Fail to obtain read lock for %v: %v", currencyID, err.Error())
			return
		}

		txn, err := mgr.ds.NewTransaction(ctx, true)
		if err != nil {
			release()
			log.Warnf("Fail to start a new transaction: %v", err.Error())
			errChan <- err
			return
		}
		defer txn.Discard(context.Background())

		exists, err := txn.Has(ctx, currencyID)
		if err != nil {
			release()
			log.Warnf("Fail to check if contains %v: %v", currencyID, err.Error())
			errChan <- err
			return
		}
		if !exists {
			release()
			log.Debugf("Currency %v not exists", currencyID)
			return
		}
		children, err := txn.GetChildren(ctx, currencyID)
		if err != nil {
			release()
			log.Warnf("Fail to get children for %v: %v", currencyID, err.Error())
			errChan <- err
			return
		}
		release()

		for child := range children {
			select {
			case out <- child:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errChan
}

// AddToHistory adds a record to a given peer's history.
//
// @input - context, currency id, peer wallet address, record.
//
// @output - error.
func (mgr *PeerManagerImpl) AddToHistory(ctx context.Context, currencyID byte, toAddr string, rec Record) error {
	log.Debugf("Add %v to history of %v-%v", rec, currencyID, toAddr)
	// Check if exists.
	release, err := mgr.ds.RLock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}

	txn, err := mgr.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, toAddr)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	if !exists {
		release()
		log.Debugf("Peer %v-%v does not exist", currencyID, toAddr)
		return fmt.Errorf("peer %v-%v does not exist", currencyID, toAddr)
	}
	release()

	release, err = mgr.ds.Lock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	defer release()

	txn, err = mgr.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	// Check again.
	exists, err = txn.Has(ctx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	if !exists {
		log.Debugf("Peer %v-%v does not exist", currencyID, toAddr)
		return fmt.Errorf("peer %v-%v does not exist", currencyID, toAddr)
	}
	val, err := txn.Get(ctx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to read ds value for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	blocked, pi, recID, err := decDSVal(val)
	if err != nil {
		log.Errorf("Fail to decode ds value %v: %v", val, err.Error())
		return err
	}
	recID.Add(recID, big.NewInt(1))
	val, err = encDSVal(blocked, pi, recID)
	if err != nil {
		log.Errorf("Fail to encode ds value for %v %v %v: %v", true, pi.String(), recID, err.Error())
		return err
	}
	err = txn.Put(ctx, val, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v-%v: %v", val, currencyID, toAddr, err.Error())
		return err
	}
	recVal, err := rec.Encode()
	if err != nil {
		log.Errorf("Fail to encode record value for %v: %v", rec, err.Error())
		return err
	}
	err = txn.Put(ctx, recVal, currencyID, toAddr, recID)
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v-%v-%v: %v", val, currencyID, toAddr, recID, err.Error())
		return err
	}
	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// RemoveRecord removes a record from a given peer's history.
//
// @input - context, currency id, peer wallet address, record id.
//
// @output - error.
func (mgr *PeerManagerImpl) RemoveRecord(ctx context.Context, currencyID byte, toAddr string, recID *big.Int) error {
	log.Debugf("Remove record %v-%v-%v", currencyID, toAddr, recID)
	// Check if exists.
	release, err := mgr.ds.RLock(ctx, currencyID, toAddr, recID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v-%v: %v", currencyID, toAddr, recID, err.Error())
		return err
	}

	txn, err := mgr.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, toAddr, recID)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, toAddr, recID, err.Error())
		return err
	}
	if !exists {
		release()
		log.Debugf("Record %v-%v-%v does not exist", currencyID, toAddr, recID)
		return fmt.Errorf("record %v-%v-%v does not exist", currencyID, toAddr, recID)
	}
	release()

	release, err = mgr.ds.Lock(ctx, currencyID, toAddr, recID)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v-%v: %v", currencyID, toAddr, recID, err.Error())
		return err
	}
	defer release()

	txn, err = mgr.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	// Check again
	exists, err = txn.Has(ctx, currencyID, toAddr, recID)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v-%v: %v", currencyID, toAddr, recID, err.Error())
		return err
	}
	if !exists {
		log.Debugf("Record %v-%v-%v does not exist", currencyID, toAddr, recID)
		return fmt.Errorf("record %v-%v-%v does not exist", currencyID, toAddr, recID)
	}
	err = txn.Delete(ctx, currencyID, toAddr, recID)
	if err != nil {
		log.Warnf("Fail to remove %v-%v-%v: %v", currencyID, toAddr, recID, err.Error())
		return err
	}
	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// SetRecID sets the next record id to use for a given peer's history.
//
// @input - context, currency id, peer wallet address, record id.
//
// @output - error.
func (mgr *PeerManagerImpl) SetRecID(ctx context.Context, currencyID byte, toAddr string, recID *big.Int) error {
	log.Debugf("Set the record id of %v-%v to be %v", currencyID, toAddr, recID)
	// Check if exists.
	release, err := mgr.ds.RLock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}

	txn, err := mgr.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, toAddr)
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	if !exists {
		release()
		log.Debugf("Peer %v-%v does not exist", currencyID, toAddr)
		return fmt.Errorf("peer %v-%v does not exist", currencyID, toAddr)
	}
	release()

	release, err = mgr.ds.Lock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	defer release()

	txn, err = mgr.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start a new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	// Check again
	exists, err = txn.Has(ctx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	if !exists {
		log.Debugf("Peer %v-%v does not exist", currencyID, toAddr)
		return fmt.Errorf("peer %v-%v does not exist", currencyID, toAddr)
	}
	val, err := txn.Get(ctx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to read ds value for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	blocked, pi, _, err := decDSVal(val)
	if err != nil {
		log.Errorf("Fail to decode ds value %v: %v", val, err.Error())
		return err
	}
	val, err = encDSVal(blocked, pi, recID)
	if err != nil {
		log.Errorf("Fail to encode ds value for %v %v %v: %v", true, pi.String(), recID, err.Error())
		return err
	}
	err = txn.Put(ctx, val, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v-%v: %v", val, currencyID, toAddr, err.Error())
		return err
	}
	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// ListHistory lists all recorded history of given peer. The result will be given from
// latest to oldest.
//
// @input - context, currency id, peer wallet address.
//
// @output - record id channel out, record channel out, error channel out.
func (mgr *PeerManagerImpl) ListHistory(ctx context.Context, currencyID byte, toAddr string) (<-chan *big.Int, <-chan Record, <-chan error) {
	out1 := make(chan *big.Int, 1)
	out2 := make(chan Record, 1)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Listing history for %v-%v", currencyID, toAddr)
		defer func() {
			close(out1)
			close(out2)
			close(errChan)
		}()
		release, err := mgr.ds.RLock(ctx, currencyID, toAddr)
		if err != nil {
			errChan <- err
			log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, toAddr, err.Error())
			return
		}
		defer release()

		txn, err := mgr.ds.NewTransaction(ctx, true)
		if err != nil {
			errChan <- err
			log.Warnf("Fail to start a new transaction: %v", err.Error())
			return
		}
		defer txn.Discard(context.Background())

		exists, err := txn.Has(ctx, currencyID, toAddr)
		if err != nil {
			errChan <- err
			log.Warnf("Fail to check if contains %v: %v", currencyID, toAddr, err.Error())
			return
		}
		if !exists {
			log.Debugf("Peer %v-%v does not exist", currencyID, toAddr)
			return
		}

		tempRes, err := txn.Query(ctx, mdlstore.Query{Orders: []dsquery.Order{historyOrder{}}}, currencyID, toAddr)
		if err != nil {
			errChan <- err
			log.Warnf("Fail to query for %v-%v: %v", currencyID, toAddr, err.Error())
			return
		}
		defer tempRes.Results.Close()
		for {
			e, ok := tempRes.Results.NextSync()
			if !ok {
				log.Debugf("Finished all query results")
				return
			}
			if e.Error != nil {
				errChan <- e.Error
				log.Warnf("Fail to get next entry: %v", e.Error.Error())
				return
			}
			temp := strings.Split(e.Key[1:], mdlstore.DatastoreKeySeperator)
			if len(temp) != 3 {
				log.Debugf("Invalid key detected: %v", e.Key[1:])
				continue
			}
			val, err := txn.Get(ctx, temp[0], temp[1], temp[2])
			if err != nil {
				log.Warnf("Fail to read %v", temp)
				continue
			}
			recID, ok := big.NewInt(0).SetString(temp[2], 10)
			if !ok {
				log.Errorf("Fail to decode res id from %v", temp)
				continue
			}
			rec, err := DecodeRecord(val)
			if err != nil {
				log.Errorf("Fail to decode record %v:%v", temp, hex.EncodeToString(val))
				continue
			}
			select {
			case out1 <- recID:
				out2 <- rec
			case <-ctx.Done():
				return
			}
		}
	}()
	return out1, out2, errChan
}
