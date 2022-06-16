package routestore

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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wcgcyx/fcr/crypto"
	"github.com/wcgcyx/fcr/mdlstore"
)

// RouteStoreImpl is the implementation of the RouteStore interface.
type RouteStoreImpl struct {
	// Signer.
	signer crypto.Signer

	// Max hops.
	maxHops map[byte]uint64

	// Routes, a map from currency id ->
	// 	map from src address -> (Easy to do get routes from)
	// 		map from dest address -> (Easy to do search routes to)
	// 			map from concatenated route -> expiry time
	ds mdlstore.MultiDimLockableStore

	// Process related.
	routineCtx context.Context

	// Shutdown function.
	shutdown func()

	// Options
	cleanFreq    time.Duration
	cleanTimeout time.Duration
}

// NewRouteStoreImpl creates a new RouteStoreImpl.
//
// @input - context, signer, options.
//
// @output - route store, error.
func NewRouteStoreImpl(ctx context.Context, signer crypto.Signer, opts Opts) (*RouteStoreImpl, error) {
	log.Infof("Start route store...")
	// Parse options.
	cleanFreq := opts.CleanFreq
	if cleanFreq == 0 {
		cleanFreq = defaultCleanFreq
	}
	cleanTimeout := opts.CleanTimeout
	if cleanTimeout == 0 {
		cleanTimeout = defaultCleanTimeout
	}
	maxHopFIL := opts.MaxHopFIL
	if maxHopFIL == 0 {
		maxHopFIL = defaultMaxHopFIL
	}
	maxHops := make(map[byte]uint64)
	maxHops[crypto.FIL] = maxHopFIL
	log.Infof("Start route store...")
	// Open store.
	ds, err := mdlstore.NewMultiDimLockableStoreImpl(ctx, opts.Path)
	if err != nil {
		log.Errorf("Fail to open mdlstore: %v", err.Error())
		return nil, err
	}
	// Create routine ctx
	routineCtx, cancel := context.WithCancel(context.Background())
	s := &RouteStoreImpl{
		signer:       signer,
		maxHops:      maxHops,
		ds:           ds,
		routineCtx:   routineCtx,
		cleanFreq:    cleanFreq,
		cleanTimeout: cleanTimeout,
	}
	s.shutdown = func() {
		log.Infof("Shutdown routine...")
		cancel()
		log.Infof("Stop datastore...")
		err := ds.Shutdown(context.Background())
		if err != nil {
			log.Errorf("Fail to stop datastore: %v", err.Error())
		}
	}
	go s.cleanRoutine()
	return s, nil
}

// Shutdown safely shuts down the component.
func (s *RouteStoreImpl) Shutdown() {
	log.Infof("Start shutdown...")
	s.shutdown()
}

// MaxHop is used to get the max hop supported for given currency.
//
// @input - context, currency id.
//
// @output - max hop, error.
func (s *RouteStoreImpl) MaxHop(ctx context.Context, currencyID byte) (uint64, error) {
	log.Debugf("Get max hop for %v", currencyID)
	maxHop, ok := s.maxHops[currencyID]
	if !ok {
		return 0, fmt.Errorf("currency %v does not have max hop set", currencyID)
	}
	return maxHop, nil
}

// AddDirectLink is used to add direct link from this node.
//
// @input - context, recipient address.
//
// @output - error.
func (s *RouteStoreImpl) AddDirectLink(ctx context.Context, currencyID byte, toAddr string) error {
	log.Debugf("Add direct link for %v-%v", currencyID, toAddr)
	_, ok := s.maxHops[currencyID]
	if !ok {
		return fmt.Errorf("currency %v does not have max hop set", currencyID)
	}
	// Check if signer has addr.
	_, root, err := s.signer.GetAddr(ctx, currencyID)
	if err != nil {
		log.Debugf("Fail to get addr for currency %v", err.Error())
		return err
	}
	if root == toAddr {
		return fmt.Errorf("cannot add self %v-%v", currencyID, toAddr)
	}

	// Now try to add this direct link, first check if the direct link exists.
	release, err := s.ds.RLock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
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
		// Fail silently if there is already a direct link presents.
		release()
		log.Debugf("%v-%v is already a direct link", currencyID, toAddr)
		return nil
	}
	release()

	release, err = s.ds.Lock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	defer release()

	txn, err = s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
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
		// Fail silently if there is already a direct link presents.
		log.Debugf("%v-%v is already a direct link", currencyID, toAddr)
		return nil
	}
	// Insert
	val, err := encDSVal(make(map[string]map[string]time.Time))
	if err != nil {
		log.Errorf("Fail to encode empty route state: %v", err.Error())
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

// RemoveDirectLink is used to remove direct link from this node.
//
// @input - context, currency id, recipient address.
//
// @output - error.
func (s *RouteStoreImpl) RemoveDirectLink(ctx context.Context, currencyID byte, toAddr string) error {
	log.Debugf("Remove direct link for %v-%v", currencyID, toAddr)
	_, ok := s.maxHops[currencyID]
	if !ok {
		return fmt.Errorf("currency %v does not have max hop set", currencyID)
	}
	// Check if signer has addr.
	_, _, err := s.signer.GetAddr(ctx, currencyID)
	if err != nil {
		log.Debugf("Fail to get addr for currency %v", err.Error())
		return err
	}

	// Now try to add this direct link, first check if the direct link exists.
	release, err := s.ds.RLock(ctx, currencyID, toAddr)
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
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
		// Does not exist.
		release()
		log.Info("%v-%v does not exist", currencyID, toAddr)
		return nil
	}
	release()

	release, err = s.ds.Lock(ctx, currencyID)
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v: %v", currencyID, err.Error())
		return err
	}
	defer release()

	txn, err = s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
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
		// Does not exist.
		log.Debugf("%v-%v does not exist", currencyID, toAddr)
		return nil
	}
	err = txn.Delete(ctx, currencyID, toAddr)
	if err != nil {
		log.Warnf("Fail to remove %v-%v: %v", currencyID, toAddr, err.Error())
		return err
	}
	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// AddRoute is used to add a route.
//
// @input - context, currency id, route, time to live.
//
// @output - error.
func (s *RouteStoreImpl) AddRoute(ctx context.Context, currencyID byte, route []string, ttl time.Duration) error {
	log.Debugf("Add route %v for %v with ttl %v", route, currencyID, ttl)
	maxHop, ok := s.maxHops[currencyID]
	if !ok {
		return fmt.Errorf("currency %v does not have max hop set", currencyID)
	}
	// Check if signer has addr.
	_, root, err := s.signer.GetAddr(ctx, currencyID)
	if err != nil {
		log.Debugf("Fail to get addr for currency %v", err.Error())
		return err
	}

	// Trim the route
	trimmed, err := trimRoute(root, route, maxHop)
	if err != nil {
		// Invalid route.
		log.Debugf("Received invalid route: %v", err.Error())
		return err
	}

	// Now first check if direct link exists.
	release, err := s.ds.RLock(ctx, currencyID, trimmed[0])
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, trimmed[0], err.Error())
		return err
	}

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, trimmed[0])
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, trimmed[0], err.Error())
		return err
	}
	if !exists {
		release()
		return fmt.Errorf("currency id %v does not have any direct link to %v", currencyID, trimmed[0])
	}
	release()

	release, err = s.ds.Lock(ctx, currencyID, trimmed[0])
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, trimmed[0], err.Error())
		return err
	}
	defer release()

	txn, err = s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err = txn.Has(ctx, currencyID, trimmed[0])
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, trimmed[0], err.Error())
		return err
	}
	if !exists {
		return fmt.Errorf("currency id %v does not have any direct link to %v", currencyID, trimmed[0])
	}

	val, err := txn.Get(ctx, currencyID, trimmed[0])
	if err != nil {
		log.Warnf("Fail to read ds value for %v-%v: %v", currencyID, trimmed[0], err.Error())
		return err
	}
	routesMap, err := decDSVal(val)
	if err != nil {
		log.Errorf("Fail to decode ds value %v, this should never happen: %v", val, err.Error())
		return err
	}
	routesMap = cleanup(routesMap)
	expiry := time.Now().Add(ttl)
	for i := len(trimmed) - 1; i >= 1; i-- {
		dest := trimmed[i]
		routeKey := strings.Join(trimmed[:i+1], RouteKeySeperator)
		_, ok = routesMap[dest]
		if !ok {
			routesMap[dest] = make(map[string]time.Time)
		}
		routesMap[dest][routeKey] = expiry
	}
	val, err = encDSVal(routesMap)
	if err != nil {
		log.Errorf("Fail to encode ds value this should never happen: %v", err.Error())
		return err
	}
	err = txn.Put(ctx, val, currencyID, trimmed[0])
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v-%v: %v", val, currencyID, trimmed[0], err.Error())
		return err
	}
	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// RemoveRoute is used to remove a route.
//
// @input - context, currency id, route.
//
// @output - error.
func (s *RouteStoreImpl) RemoveRoute(ctx context.Context, currencyID byte, route []string) error {
	log.Debugf("Remove route %v from %v", route, currencyID)
	maxHop, ok := s.maxHops[currencyID]
	if !ok {
		return fmt.Errorf("currency %v does not have max hop set", currencyID)
	}
	// Check if signer has addr.
	_, root, err := s.signer.GetAddr(ctx, currencyID)
	if err != nil {
		log.Debugf("Fail to get addr for currency %v", err.Error())
		return err
	}

	// Trim the route
	trimmed, err := trimRoute(root, route, maxHop)
	if err != nil {
		// Invalid route.
		log.Debugf("Received invalid route: %v", err.Error())
		return err
	}

	// Now first check if direct link exists.
	release, err := s.ds.RLock(ctx, currencyID, trimmed[0])
	if err != nil {
		log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, trimmed[0], err.Error())
		return err
	}

	txn, err := s.ds.NewTransaction(ctx, true)
	if err != nil {
		release()
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err := txn.Has(ctx, currencyID, trimmed[0])
	if err != nil {
		release()
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, trimmed[0], err.Error())
		return err
	}
	if !exists {
		release()
		return fmt.Errorf("currency id %v does not have any direct link to %v", currencyID, trimmed[0])
	}

	// Check if route exists
	val, err := txn.Get(ctx, currencyID, trimmed[0])
	if err != nil {
		release()
		log.Warnf("Fail to read ds value for %v-%v: %v", currencyID, trimmed[0], err.Error())
		return err
	}
	routesMap, err := decDSVal(val)
	if err != nil {
		release()
		log.Errorf("Fail to decode ds value %v, this should never happen: %v", val, err.Error())
		return err
	}
	_, ok = routesMap[trimmed[len(trimmed)-1]]
	if !ok {
		release()
		return fmt.Errorf("route destination %v not exists", trimmed[len(trimmed)-1])
	}
	_, ok = routesMap[trimmed[len(trimmed)-1]][strings.Join(trimmed, RouteKeySeperator)]
	if !ok {
		release()
		return fmt.Errorf("route %v not exists", route)
	}
	release()

	release, err = s.ds.Lock(ctx, currencyID, trimmed[0])
	if err != nil {
		log.Debugf("Fail to obtain write lock for %v-%v: %v", currencyID, trimmed[0], err.Error())
		return err
	}
	defer release()

	txn, err = s.ds.NewTransaction(ctx, false)
	if err != nil {
		log.Warnf("Fail to start new transaction: %v", err.Error())
		return err
	}
	defer txn.Discard(context.Background())

	exists, err = txn.Has(ctx, currencyID, trimmed[0])
	if err != nil {
		log.Warnf("Fail to check if contains %v-%v: %v", currencyID, trimmed[0], err.Error())
		return err
	}
	if !exists {
		return fmt.Errorf("currency id %v does not have any direct link to %v", currencyID, trimmed[0])
	}

	val, err = txn.Get(ctx, currencyID, trimmed[0])
	if err != nil {
		log.Warnf("Fail to read ds value for %v-%v: %v", currencyID, trimmed[0], err.Error())
		return err
	}
	routesMap, err = decDSVal(val)
	if err != nil {
		log.Errorf("Fail to decode ds value %v, this should never happen: %v", val, err.Error())
		return err
	}
	routesMap = cleanup(routesMap)
	_, ok = routesMap[trimmed[len(trimmed)-1]]
	if ok {
		delete(routesMap[trimmed[len(trimmed)-1]], strings.Join(trimmed, RouteKeySeperator))
	}
	val, err = encDSVal(routesMap)
	if err != nil {
		log.Errorf("Fail to encode ds value this should never happen: %v", err.Error())
		return err
	}
	err = txn.Put(ctx, val, currencyID, trimmed[0])
	if err != nil {
		log.Warnf("Fail to put ds value %v for %v-%v: %v", val, currencyID, trimmed[0], err.Error())
		return err
	}
	err = txn.Commit(ctx)
	if err != nil {
		log.Warnf("Fail to commit transaction: %v", err.Error())
		return err
	}
	return nil
}

// ListRoutesTo is used to list all routes that has given destination.
//
// @input - context, currency id.
//
// @output - route chan out, error chan out.
func (s *RouteStoreImpl) ListRoutesTo(ctx context.Context, currencyID byte, toAddr string) (<-chan []string, <-chan error) {
	out := make(chan []string, 128)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Listing routes to %v-%v", currencyID, toAddr)
		defer func() {
			close(out)
			close(errChan)
		}()
		// Check if signer has addr.
		_, _, err := s.signer.GetAddr(ctx, currencyID)
		if err != nil {
			errChan <- err
			log.Debugf("Fail to get addr for currency %v", err.Error())
			return
		}

		release0, err := s.ds.RLock(ctx, currencyID)
		if err != nil {
			errChan <- err
			log.Debugf("Fail to obtain read lock for %v: %v", currencyID, err.Error())
			return
		}
		defer release0()

		txn0, err := s.ds.NewTransaction(ctx, true)
		if err != nil {
			errChan <- err
			log.Warnf("Fail to start new transaction: %v", err.Error())
			return
		}
		defer txn0.Discard(context.Background())

		children, err := txn0.GetChildren(ctx, currencyID)
		if err != nil {
			errChan <- err
			log.Warnf("Fail to get children for %v: %v", currencyID, err.Error())
			return
		}
		wg0 := sync.WaitGroup{}
		for child := range children {
			wg0.Add(1)
			go func(srcAddr string) {
				defer wg0.Done()
				release1, err := s.ds.RLock(ctx, currencyID, srcAddr)
				if err != nil {
					log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, srcAddr, err.Error())
					return
				}
				defer release1()

				txn1, err := s.ds.NewTransaction(ctx, true)
				if err != nil {
					log.Warnf("Fail to start new transaction: %v", err.Error())
					return
				}
				defer txn1.Discard(context.Background())

				val, err := txn1.Get(ctx, currencyID, srcAddr)
				if err != nil {
					log.Warnf("Fail to read ds value for %v-%v: %v", currencyID, srcAddr, err.Error())
					return
				}
				routesMap, err := decDSVal(val)
				if err != nil {
					log.Errorf("Fail to decode ds value %v, this should never happen: %v", val, err.Error())
					return
				}
				routesMap = cleanup(routesMap)
				routes, ok := routesMap[toAddr]
				if ok {
					for routeKey := range routes {
						route := strings.Split(routeKey, RouteKeySeperator)
						select {
						case out <- route:
						case <-ctx.Done():
							return
						}
					}
				}
			}(child)
		}
		wg0.Wait()
	}()
	return out, errChan
}

// ListRoutesFrom is used to list all routes that has given source.
//
// @input - context, currency id.
//
// @output - route chan out, error chan out.
func (s *RouteStoreImpl) ListRoutesFrom(ctx context.Context, currencyID byte, toAddr string) (<-chan []string, <-chan error) {
	out := make(chan []string, 128)
	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Listing routes from %v-%v", currencyID, toAddr)
		defer func() {
			close(out)
			close(errChan)
		}()
		_, ok := s.maxHops[currencyID]
		if !ok {
			errChan <- fmt.Errorf("currency %v does not have max hop set", currencyID)
			return
		}
		// Check if signer has addr.
		_, _, err := s.signer.GetAddr(ctx, currencyID)
		if err != nil {
			errChan <- err
			log.Debugf("Fail to get addr for currency %v", err.Error())
			return
		}

		release, err := s.ds.RLock(ctx, currencyID, toAddr)
		if err != nil {
			errChan <- err
			log.Debugf("Fail to obtain read lock for %v-%v: %v", currencyID, toAddr, err.Error())
			return
		}

		txn, err := s.ds.NewTransaction(ctx, true)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to start new transaction: %v", err.Error())
			return
		}
		defer txn.Discard(context.Background())

		exists, err := txn.Has(ctx, currencyID, toAddr)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to check if contains %v-%v: %v", currencyID, toAddr, err.Error())
			return
		}
		if !exists {
			release()
			return
		}
		// Exist, the direct link itself is a route
		select {
		case out <- []string{toAddr}:
		case <-ctx.Done():
			return
		}
		val, err := txn.Get(ctx, currencyID, toAddr)
		if err != nil {
			release()
			errChan <- err
			log.Warnf("Fail to read ds value for %v-%v: %v", currencyID, toAddr, err.Error())
			return
		}
		routesMap, err := decDSVal(val)
		if err != nil {
			release()
			errChan <- err
			log.Errorf("Fail to decode ds value %v, this should never happen: %v", val, err.Error())
			return
		}
		release()
		routesMap = cleanup(routesMap)
		for _, routes := range routesMap {
			for routeKey := range routes {
				route := strings.Split(routeKey, RouteKeySeperator)
				// TODO: In route publish, when listing routes from, who call this function
				// will need to process the result so only longest route will be published.
				select {
				case out <- route:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, errChan
}
