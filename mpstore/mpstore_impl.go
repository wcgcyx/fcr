package mpstore

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

	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/mdlstore"
)

// MinerProofStoreImpl is the implementation of the MinerProofStore interface.
type MinerProofStoreImpl struct {
	// Datastore
	ds mdlstore.MultiDimLockableStore

	// Signer
	signer crypto.Signer

	// Shutdown function.
	shutdown func()
}

// NewMinerProofStoreImpl creates a new MinerProofStore.
//
// @input - context, signer, options.
//
// @output - miner proof store, error.
func NewMinerProofStoreImpl(ctx context.Context, signer crypto.Signer, opts Opts) (*MinerProofStoreImpl, error) {
	log.Infof("Start miner proof store...")
	// Open store.
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, opts.Path)
	if err != nil {
		log.Errorf("Fail to open mdlstore: %v", err.Error())
		return nil, err
	}
	s := &MinerProofStoreImpl{
		ds:     ds,
		signer: signer,
	}
	s.shutdown = func() {
		log.Infof("Stop datastore...")
		err := ds.Shutdown(context.Background())
		if err != nil {
			log.Errorf("Fail to stop datastore: %v", err.Error())
		}
	}
	return s, nil
}

// Shutdown safely shuts down the component.
func (s *MinerProofStoreImpl) Shutdown() {
	log.Infof("Start shutdown...")
	s.shutdown()
}

// UpsertMinerProof is used to set the miner proof for a given currency id.
//
// @input - context, currency id, miner key type, miner address, proof.
//
// @output - error.
func (s *MinerProofStoreImpl) UpsertMinerProof(ctx context.Context, currencyID byte, minerKeyType byte, minerAddr string, proof []byte) error {
	log.Debugf("Upsert miner proof for %v with %v-%v-%v", currencyID, minerKeyType, minerAddr, proof)
	// First check if miner proof is valid.
	_, addr, err := s.signer.GetAddr(ctx, currencyID)
	if err != nil {
		log.Warnf("Fail to get addr for %v: %v", currencyID, err.Error())
		return err
	}
	err = VerifyMinerProof(currencyID, addr, minerKeyType, minerAddr, proof)
	if err != nil {
		log.Debugf("Fail to verify miner proof %v from %v-%v for %v-%v: %v", proof, minerKeyType, minerAddr, currencyID, addr, err.Error())
		return err
	}

	release, err := s.ds.Lock(ctx)
	if err != nil {
		log.Debugf("Fail to obtain write lock for root: %v", err.Error())
		return err
	}
	defer release()

	txn, err := s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	val, err := encDSVal(minerKeyType, minerAddr, proof)
	if err != nil {
		log.Errorf("Fail to encode ds value for %v %v %v: %v", minerKeyType, minerAddr, proof, err.Error())
		return err
	}

	err = txn.Put(ctx, val, currencyID)
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v: %v", val, currencyID, err.Error())
		return err
	}

	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// GetMinerProof is used to get the miner proof for a given currency id.
//
// @input - context, currency id.
//
// @output - if exists, miner key type, miner address, proof, error.
func (s *MinerProofStoreImpl) GetMinerProof(ctx context.Context, currencyID byte) (bool, byte, string, []byte, error) {
	log.Debugf("Get miner proof for %v", currencyID)
	release, err := s.ds.RLock(ctx, currencyID)
	if err != nil {
		log.Debugf("Fail to obtain read lock for root: %v", err.Error())
		return false, 0, "", nil, err
	}
	defer release()

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return false, 0, "", nil, err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID)
	if err != nil {
		log.Warnf("Fail to check if contains %v: %v", currencyID, err.Error())
		return false, 0, "", nil, err
	}
	if !exists {
		log.Debugf("Miner proof for %v doesn not exist", currencyID)
		return false, 0, "", nil, nil
	}

	val, err := txn.Get(ctx, currencyID)
	if err != nil {
		log.Debugf("Fail to get ds value for %v: %v", currencyID, err.Error())
		return false, 0, "", nil, err
	}

	minerKeyType, minerAddr, proof, err := decDSVal(val)
	if err != nil {
		log.Errorf("Fail to decode ds value %v: %v", val, err.Error())
		return false, 0, "", nil, err
	}
	return true, minerKeyType, minerAddr, proof, nil
}
