package payofferstore

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	badgerds "github.com/ipfs/go-ds-badger"
	logging "github.com/ipfs/go-log"
	"github.com/wcgcyx/fcr/internal/payoffer"
)

const (
	payofferStoreSubPath = "payofferstore"
	gcInterval           = 6 * time.Hour
)

var log = logging.Logger("payofferstore")

// PayOfferStoreImplV1 implements the PayOfferStore.
type PayOfferStoreImplV1 struct {
	// Parent context
	ctx context.Context
	// Offer store
	lock  sync.RWMutex
	store datastore.TTLDatastore
}

// NewPayOfferStoreImplV1 creates a PayOfferStore.
// It takes a context, a root db path as arguments.
// It returns a pay offer store and error.
func NewPayOfferStoreImplV1(ctx context.Context, dbPath string) (PayOfferStore, error) {
	os := &PayOfferStoreImplV1{
		ctx:  ctx,
		lock: sync.RWMutex{},
	}
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	dsopts.GcInterval = gcInterval
	store, err := badgerds.NewDatastore(filepath.Join(dbPath, payofferStoreSubPath), &dsopts)
	if err != nil {
		return nil, err
	}
	os.store = store
	go os.shutdownRoutine()
	return os, nil
}

// AddOffer is used to store a pay offer.
// It takes a pay offer as the argument.
// It returns error.
func (os *PayOfferStoreImplV1) AddOffer(offer *payoffer.PayOffer) error {
	os.lock.Lock()
	defer os.lock.Unlock()
	ttl := time.Duration(offer.Expiration()-time.Now().Unix()) * time.Second
	if ttl <= 0 {
		return fmt.Errorf("offer has expired at %v now %v", offer.Expiration(), time.Now().Unix())
	}
	return os.store.PutWithTTL(datastore.NewKey(offer.ID().String()), offer.ToBytes(), ttl)
}

// GetOffer is used to get a pay offer.
// It takes a offer id as the argument.
// It returns the pay offer and error.
func (os *PayOfferStoreImplV1) GetOffer(id cid.Cid) (*payoffer.PayOffer, error) {
	os.lock.RLock()
	defer os.lock.RUnlock()
	data, err := os.store.Get(datastore.NewKey(id.String()))
	if err != nil {
		return nil, err
	}
	return payoffer.FromBytes(data)
}

// shutdownRoutine is used to safely close the routine.
func (os *PayOfferStoreImplV1) shutdownRoutine() {
	<-os.ctx.Done()
	os.lock.Lock()
	defer os.lock.Unlock()
	if err := os.store.Close(); err != nil {
		log.Errorf("error closing storage: %v", err.Error())
	}
}
