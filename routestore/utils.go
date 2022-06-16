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
	"encoding/json"
	"fmt"
	"time"
)

// trimRoute is used to check if a route is valid and also trim any valid route.
//
// @input - root address, route to add, max hop.
//
// @output - trimmed route, error.
func trimRoute(root string, route []string, maxHop uint64) ([]string, error) {
	if len(route) == 1 {
		// Not valid route.
		// Every route needs to be at least length in 2.
		return nil, fmt.Errorf("invalid route provided, need to be at least 2")
	}
	if len(route) > int(maxHop) {
		// Trim the route
		route = route[:maxHop]
	}
	// Check if trimmed route is acyclic
	visited := make(map[string]bool)
	visited[root] = true
	for _, node := range route {
		_, ok := visited[node]
		if ok {
			return nil, fmt.Errorf("route provided is cyclic")
		}
		visited[node] = true
	}
	return route, nil
}

// cleanup will remove all expired routes.
//
// @input - routes map.
//
// @output - trimmed routes.
func cleanup(routesMap map[string]map[string]time.Time) map[string]map[string]time.Time {
	newRoutesMap := make(map[string]map[string]time.Time)
	for dest, routes := range routesMap {
		temp := make(map[string]time.Time)
		for route, expiry := range routes {
			if time.Now().Before(expiry) {
				// Still valid.
				temp[route] = expiry
			}
		}
		if len(temp) > 0 {
			newRoutesMap[dest] = temp
		}
	}
	return newRoutesMap
}

// encDSVal encodes the data to the ds value.
//
// @input - routes map.
//
// @output - value.
func encDSVal(routesMap map[string]map[string]time.Time) ([]byte, error) {
	return json.Marshal(routesMap)
}

// decDSVal decodes the ds value into data.
//
// @input - value.
//
// @output - routes map, error.
func decDSVal(val []byte) (map[string]map[string]time.Time, error) {
	var valDec map[string]map[string]time.Time
	err := json.Unmarshal(val, &valDec)
	return valDec, err
}
