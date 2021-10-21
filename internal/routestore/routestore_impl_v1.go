package routestore

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	badgerds "github.com/ipfs/go-ds-badger"
	logging "github.com/ipfs/go-log"
)

const (
	fwdRouteStoreSubPath = "fwdroutestore"
	bwdRouteStoreSubPath = "bwdroutestore"
	seperator            = ";"
	gcInterval           = 2 * time.Hour
)

var log = logging.Logger("routestore")

// RouteStoreImplV1 implements RouteStore interface.
type RouteStoreImplV1 struct {
	// Parent context
	ctx context.Context
	// Roots
	roots map[uint64]string
	// MaxHops
	maxHops map[uint64]int
	// Forward & Backward stores
	storesLock map[uint64]*sync.RWMutex
	fwdStores  map[uint64]datastore.Datastore
	bwdStores  map[uint64]datastore.Datastore
}

// NewRouteStoreImplV1 creates a RouteStore.
// It takes a context, a root db path, a list of currency ids, a map of max hops, a map of roots as arguments.
// It returns a route store and error.
func NewRouteStoreImplV1(ctx context.Context, dbPath string, currencyIDs []uint64, maxHops map[uint64]int, roots map[uint64]string) (RouteStore, error) {
	rs := &RouteStoreImplV1{
		ctx:        ctx,
		roots:      make(map[uint64]string),
		maxHops:    make(map[uint64]int),
		storesLock: make(map[uint64]*sync.RWMutex),
		fwdStores:  make(map[uint64]datastore.Datastore),
		bwdStores:  make(map[uint64]datastore.Datastore),
	}
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	for _, currencyID := range currencyIDs {
		root, ok := roots[currencyID]
		if !ok {
			return nil, fmt.Errorf("root for currency id %v not provided", currencyID)
		}
		maxHop, ok := maxHops[currencyID]
		if !ok {
			return nil, fmt.Errorf("max hop for currency id %v not provided", currencyID)
		}
		rs.roots[currencyID] = root
		rs.maxHops[currencyID] = maxHop
		store, err := badgerds.NewDatastore(filepath.Join(dbPath, fmt.Sprintf("%v-%v", fwdRouteStoreSubPath, currencyID)), &dsopts)
		if err != nil {
			return nil, err
		}
		rs.fwdStores[currencyID] = store
		store, err = badgerds.NewDatastore(filepath.Join(dbPath, fmt.Sprintf("%v-%v", bwdRouteStoreSubPath, currencyID)), &dsopts)
		if err != nil {
			return nil, err
		}
		rs.bwdStores[currencyID] = store
		rs.storesLock[currencyID] = &sync.RWMutex{}
	}
	go rs.shutdownRoutine()
	go rs.gcRoutine()
	return rs, nil
}

// MaxHop obtain the maximum hop supported for a given currency id.
// It taktes a currency id as the argument.
// It returns the max hop count.
func (rs *RouteStoreImplV1) MaxHop(currencyID uint64) int {
	maxHop, _ := rs.maxHops[currencyID]
	return maxHop
}

// AddDirectLink will add a direct link to the root address for a given currency id.
// It takes a currency id and the address of a direct link as arguments.
// It returns the error.
func (rs *RouteStoreImplV1) AddDirectLink(currencyID uint64, toAddr string) error {
	store, ok := rs.fwdStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	lock := rs.storesLock[currencyID]
	lock.RLock()
	key := datastore.NewKey(toAddr)
	exists, err := store.Has(key)
	if err != nil {
		lock.RUnlock()
		return fmt.Errorf("error checking storage: %v", err.Error())
	}
	if exists {
		lock.RUnlock()
		return fmt.Errorf("direct link %v already exists", toAddr)
	}
	lock.RUnlock()
	lock.Lock()
	defer lock.Unlock()
	routes := make(map[string]int64)
	data, _ := json.Marshal(routes)
	if err = store.Put(key, data); err != nil {
		return fmt.Errorf("error writing to storage: %v", err.Error())
	}
	return nil
}

// RemoveDirectLink will remove a direct link from the root address.
// It takes a currency id and the address of the direct link to remove as arguments.
// It returns the error.
func (rs *RouteStoreImplV1) RemoveDirectLink(currencyID uint64, toAddr string) error {
	fwdStore, ok := rs.fwdStores[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	bwdStore := rs.bwdStores[currencyID]
	lock := rs.storesLock[currencyID]
	lock.RLock()
	key := datastore.NewKey(toAddr)
	data, err := fwdStore.Get(key)
	if err != nil {
		lock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	lock.RUnlock()
	lock.Lock()
	defer lock.Unlock()
	// Remove from fwdstore
	if err = fwdStore.Delete(key); err != nil {
		return fmt.Errorf("error removing key from storage: %v", err.Error())
	}
	var routes map[string]int64
	json.Unmarshal(data, &routes)
	if len(routes) > 0 {
		// Clean bwdstore
		for route := range routes {
			nodes := strings.Split(route, seperator)
			toRemove := nodes[len(nodes)-1]
			key = datastore.NewKey(toRemove)
			data, err = bwdStore.Get(key)
			if err != nil {
				return fmt.Errorf("error reading storage: %v", err.Error())
			}
			var toUpdate map[string]int64
			json.Unmarshal(data, &toUpdate)
			// Mark it as expired for the GC to clean
			toUpdate[route] = time.Now().Unix() - 1
			// Put it back
			data, _ = json.Marshal(toUpdate)
			if err = bwdStore.Put(key, data); err != nil {
				return fmt.Errorf("error writing to storage: %v", err.Error())
			}
		}
	}
	return nil
}

// AddRoute will add a route to the store. Once added, the route will
// add the root in front.
// It takes a currency id and the route to add as arguments.
// It returns the error.
func (rs *RouteStoreImplV1) AddRoute(currencyID uint64, route []string, ttl time.Duration) error {
	root, ok := rs.roots[currencyID]
	if !ok {
		return fmt.Errorf("currency id %v is not supported", currencyID)
	}
	maxHop := rs.maxHops[currencyID]
	// Add root to the route
	route = append([]string{root}, route...)
	if len(route) <= 2 {
		return fmt.Errorf("route is supposed to have at least length in 3 but got %v", len(route))
	}
	if len(route) > maxHop {
		route = route[:maxHop]
	}
	if !validRoute(route) {
		return fmt.Errorf("route is not valid")
	}
	fwdStore := rs.fwdStores[currencyID]
	bwdStore := rs.bwdStores[currencyID]
	lock := rs.storesLock[currencyID]
	lock.RLock()
	key := datastore.NewKey(route[1])
	data, err := fwdStore.Get(key)
	if err != nil {
		lock.RUnlock()
		return fmt.Errorf("error reading storage: %v", err.Error())
	}
	lock.RUnlock()
	lock.Lock()
	defer lock.Unlock()
	var routes map[string]int64
	json.Unmarshal(data, &routes)
	for i := 2; i < len(route); i++ {
		bKey := datastore.NewKey(route[i])
		// Check if bKey is part of the direct link
		exists, err := fwdStore.Has(bKey)
		if err != nil {
			return fmt.Errorf("error checking storage: %v", err.Error())
		}
		if exists {
			break
		}
		// Update bwd store
		var bRoutes map[string]int64
		data, err = bwdStore.Get(bKey)
		if err != nil {
			bRoutes = make(map[string]int64)
		} else {
			json.Unmarshal(data, &bRoutes)
		}
		routeKey := strings.Join(route[:(i+1)], seperator)
		expiration := time.Now().Add(ttl).Unix()
		routes[routeKey] = expiration
		bRoutes[routeKey] = expiration
		data, _ = json.Marshal(bRoutes)
		err = bwdStore.Put(bKey, data)
		if err != nil {
			return fmt.Errorf("error writing to storage: %v", err.Error())
		}
	}
	// Update fwdstore
	data, _ = json.Marshal(routes)
	err = fwdStore.Put(key, data)
	if err != nil {
		return fmt.Errorf("error writing to storage: %v", err.Error())
	}
	return nil
}

// GetRoutesTo will get all the routes that have the given destination.
// It takes a currency id and the destination address as arguments.
// It returns a list of routes.
// It returns the error.
func (rs *RouteStoreImplV1) GetRoutesTo(currencyID uint64, toAddr string) ([][]string, error) {
	store, ok := rs.bwdStores[currencyID]
	if !ok {
		return nil, fmt.Errorf("currency id %v is not supported", currencyID)
	}
	lock := rs.storesLock[currencyID]
	lock.RLock()
	defer lock.RUnlock()
	key := datastore.NewKey(toAddr)
	res := make([][]string, 0)
	exists, err := store.Has(key)
	if err != nil {
		return nil, fmt.Errorf("error checking storage: %v", err.Error())
	}
	if exists {
		data, err := store.Get(key)
		if err != nil {
			return nil, fmt.Errorf("error reading storage: %v", err.Error())
		}

		var routes map[string]int64
		json.Unmarshal(data, &routes)
		for routeStr, exp := range routes {
			if time.Now().Unix() > exp {
				continue
			}
			res = append(res, strings.Split(routeStr, seperator))
		}
	}
	// Need to add direct served routes.
	store = rs.fwdStores[currencyID]
	exists, err = store.Has(key)
	if err != nil {
		return nil, fmt.Errorf("error checking storage: %v", err.Error())
	}
	if exists {
		root := rs.roots[currencyID]
		res = append(res, []string{root, toAddr})
	}
	return res, nil
}

// GetRoutesFrom will get all the routes that start with the given direct link (from address).
// It takes a currency id and the from address as arguments.
// It returns a list of routes and error.
func (rs *RouteStoreImplV1) GetRoutesFrom(currencyID uint64, toAddr string) ([][]string, error) {
	store, ok := rs.fwdStores[currencyID]
	if !ok {
		return nil, fmt.Errorf("currency id %v is not supported", currencyID)
	}
	lock := rs.storesLock[currencyID]
	lock.RLock()
	defer lock.RUnlock()
	key := datastore.NewKey(toAddr)
	res := make([][]string, 0)
	exists, err := store.Has(key)
	if err != nil {
		return nil, fmt.Errorf("error checking storage: %v", err.Error())
	}
	if exists {
		data, err := store.Get(key)
		if err != nil {
			return nil, fmt.Errorf("error reading storage: %v", err.Error())
		}
		var routes map[string]int64
		json.Unmarshal(data, &routes)
		for routeStr, exp := range routes {
			if time.Now().Unix() > exp {
				continue
			}
			res = append(res, strings.Split(routeStr, seperator))
		}
		root := rs.roots[currencyID]
		res = append(res, []string{root, toAddr})
	}
	return res, nil
}

// gcRoutine is the garbage collection routine.
func (rs *RouteStoreImplV1) gcRoutine() {
	for {
		// Do a gc
		for currencyID := range rs.roots {
			go rs.gc(currencyID)
		}
		tc := time.After(gcInterval)
		select {
		case <-rs.ctx.Done():
			return
		case <-tc:
		}
	}
}

// gc is the garbage collection process.
func (rs *RouteStoreImplV1) gc(currencyID uint64) {
	fwdStore := rs.fwdStores[currencyID]
	bwdStore := rs.bwdStores[currencyID]
	lock := rs.storesLock[currencyID]
	lock.RLock()

	q := dsq.Query{KeysOnly: false}
	tempRes, err := fwdStore.Query(q)
	if err != nil {
		log.Errorf("error querying storage: %v", err.Error())
		return
	}
	output := make(chan string, dsq.KeysOnlyBufSize)

	go func() {
		defer func() {
			tempRes.Close()
			close(output)
		}()
		for {
			e, ok := tempRes.NextSync()
			if !ok {
				break
			}
			if e.Error != nil {
				log.Errorf("error querying storage: %v", e.Error.Error())
				return
			}
			var routes map[string]int64
			json.Unmarshal(e.Value, &routes)
			for route, exp := range routes {
				if time.Now().Unix() >= exp {
					select {
					case <-rs.ctx.Done():
						return
					case output <- route:
					}
				}
			}
		}
	}()

	toRemove := make([]string, 0)
	for expired := range output {
		toRemove = append(toRemove, expired)
	}
	lock.RUnlock()

	if len(toRemove) > 0 {
		lock.Lock()
		defer lock.Unlock()
		for _, routeStr := range toRemove {
			route := strings.Split(routeStr, seperator)
			fKey := datastore.NewKey(route[1])
			bKey := datastore.NewKey(route[len(route)-1])
			// Remove from fwdstore
			data, err := fwdStore.Get(fKey)
			if err != nil {
				log.Errorf("error reading storage: %v", err.Error())
				return
			}
			var toUpdate map[string]int64
			json.Unmarshal(data, &toUpdate)
			delete(toUpdate, routeStr)
			data, _ = json.Marshal(toUpdate)
			if err = fwdStore.Put(fKey, data); err != nil {
				log.Errorf("error writing to storage: %v", err.Error())
				return
			}
			// Remove from bwdstore
			data, err = bwdStore.Get(bKey)
			if err != nil {
				log.Errorf("error reading storage: %v", err.Error())
				return
			}
			json.Unmarshal(data, &toUpdate)
			delete(toUpdate, routeStr)
			if len(toUpdate) > 0 {
				data, _ = json.Marshal(toUpdate)
				if err = bwdStore.Put(bKey, data); err != nil {
					log.Errorf("error writing to storage: %v", err.Error())
					return
				}
			} else {
				if err = bwdStore.Delete(bKey); err != nil {
					log.Errorf("error removing from storage: %v", err.Error())
					return
				}
			}
		}
	}
}

// shutdownRoutine is used to safely close the routine.
func (rs *RouteStoreImplV1) shutdownRoutine() {
	<-rs.ctx.Done()
	for currencyID, fwdStore := range rs.fwdStores {
		bwdStore := rs.bwdStores[currencyID]
		lock := rs.storesLock[currencyID]
		lock.Lock()
		defer lock.Unlock()
		if err := fwdStore.Close(); err != nil {
			log.Errorf("error closing storage for currency id %v: %v", currencyID, err.Error())
		}
		if err := bwdStore.Close(); err != nil {
			log.Errorf("error closing storage for currency id %v: %v", currencyID, err.Error())
		}
	}
}

// validRoute is a helper function to check if a given route is valid (acyclic).
func validRoute(route []string) bool {
	visited := make(map[string]bool)
	for _, node := range route {
		_, ok := visited[node]
		if ok {
			return false
		}
		visited[node] = true
	}
	return true
}
