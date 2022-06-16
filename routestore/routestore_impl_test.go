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
	"crypto/rand"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/crypto"
)

const (
	testDS = "./test-ds"
)

func TestMain(m *testing.M) {
	os.RemoveAll(testDS)
	os.Mkdir(testDS, os.ModePerm)
	defer os.RemoveAll(testDS)
	m.Run()
}

func TestNewRouteStoreImpl(t *testing.T) {
	ctx := context.Background()

	_, rs, shutdown, err := getNode(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, rs)
	defer shutdown()

	_, err = rs.MaxHop(ctx, 0)
	assert.NotNil(t, err)

	maxHop, err := rs.MaxHop(ctx, crypto.FIL)
	assert.Nil(t, err)
	assert.Equal(t, defaultMaxHopFIL, maxHop)
}

func TestAddDirectLink(t *testing.T) {
	ctx := context.Background()

	signer, rs, shutdown, err := getNode(ctx)
	assert.Nil(t, err)
	defer shutdown()

	_, root, err := signer.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	err = rs.AddDirectLink(ctx, 0, "peer1")
	assert.NotNil(t, err)

	err = rs.AddDirectLink(ctx, crypto.FIL, root)
	assert.NotNil(t, err)

	err = rs.AddDirectLink(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)

	err = rs.AddDirectLink(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)
}

func TestRemoveDirectLink(t *testing.T) {
	ctx := context.Background()

	_, rs, shutdown, err := getNode(ctx)
	assert.Nil(t, err)
	defer shutdown()

	err = rs.RemoveDirectLink(ctx, 0, "peer1")
	assert.NotNil(t, err)

	err = rs.RemoveDirectLink(ctx, crypto.FIL, "peer2")
	assert.Nil(t, err)

	err = rs.RemoveDirectLink(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)
}

func TestAddRoute(t *testing.T) {
	ctx := context.Background()

	signer, rs, shutdown, err := getNode(ctx)
	assert.Nil(t, err)
	defer shutdown()

	_, root, err := signer.GetAddr(ctx, crypto.FIL)
	assert.Nil(t, err)

	err = rs.AddDirectLink(ctx, crypto.FIL, "peer1")
	assert.Nil(t, err)

	err = rs.AddDirectLink(ctx, crypto.FIL, "peer2")
	assert.Nil(t, err)

	err = rs.AddRoute(ctx, 0, []string{"peer1", "peer1-1", "peer1-1-1"}, time.Second)
	assert.NotNil(t, err)

	err = rs.AddRoute(ctx, crypto.FIL, []string{"peer1"}, time.Second)
	assert.NotNil(t, err)

	err = rs.AddRoute(ctx, crypto.FIL, []string{"peer3", "peer1-1", "peer1-1-1"}, time.Second)
	assert.NotNil(t, err)

	err = rs.AddRoute(ctx, crypto.FIL, []string{"peer1", "peer1-1", "peer1"}, time.Second)
	assert.NotNil(t, err)

	err = rs.AddRoute(ctx, crypto.FIL, []string{"peer1", "peer1-1", root}, time.Second)
	assert.NotNil(t, err)

	err = rs.AddRoute(ctx, crypto.FIL, []string{"peer1", "peer1-1", "peer1-1-1"}, time.Second)
	assert.Nil(t, err)

	err = rs.AddRoute(ctx, crypto.FIL, []string{"peer1", "peer1-1", "peer1-1-2"}, time.Second)
	assert.Nil(t, err)

	err = rs.AddRoute(ctx, crypto.FIL, []string{"peer1", "peer1-2", "peer1-1-2"}, 2*time.Second)
	assert.Nil(t, err)

	err = rs.AddRoute(ctx, crypto.FIL, []string{"peer2", "peer1-1", "peer1-1-2", "peer3", "peer3-1", "peer3-1-1"}, 2*time.Second)
	assert.Nil(t, err)
}

func TestList(t *testing.T) {
	ctx := context.Background()

	_, rs, shutdown, err := getNode(ctx)
	assert.Nil(t, err)
	defer shutdown()

	routesChan, errChan := rs.ListRoutesFrom(ctx, crypto.FIL, "peer0")
	routes := make([][]string, 0)
	for route := range routesChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(routes))

	routesChan, errChan = rs.ListRoutesFrom(ctx, crypto.FIL, "peer1")
	routes = make([][]string, 0)
	for route := range routesChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 6, len(routes))

	routesChan, errChan = rs.ListRoutesFrom(ctx, crypto.FIL, "peer2")
	routes = make([][]string, 0)
	for route := range routesChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 5, len(routes))

	routesChan, errChan = rs.ListRoutesTo(ctx, crypto.FIL, "peer2")
	routes = make([][]string, 0)
	for route := range routesChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 0, len(routes))

	routesChan, errChan = rs.ListRoutesTo(ctx, crypto.FIL, "peer1-1")
	routes = make([][]string, 0)
	for route := range routesChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 2, len(routes))

	routesChan, errChan = rs.ListRoutesTo(ctx, crypto.FIL, "peer1-1-1")
	routes = make([][]string, 0)
	for route := range routesChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(routes))

	routesChan, errChan = rs.ListRoutesTo(ctx, crypto.FIL, "peer1-1-2")
	routes = make([][]string, 0)
	for route := range routesChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 3, len(routes))

	err = rs.RemoveRoute(ctx, 0, []string{"peer1", "peer1-1"})
	assert.NotNil(t, err)

	err = rs.RemoveRoute(ctx, crypto.FIL, []string{"peer3", "peer1-1"})
	assert.NotNil(t, err)

	err = rs.RemoveRoute(ctx, crypto.FIL, []string{"peer1", "peer1-1"})
	assert.Nil(t, err)

	routesChan, errChan = rs.ListRoutesTo(ctx, crypto.FIL, "peer1-1")
	routes = make([][]string, 0)
	for route := range routesChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(routes))

	// Test the routines.
	time.Sleep(time.Second + time.Millisecond)

	routesChan, errChan = rs.ListRoutesFrom(ctx, crypto.FIL, "peer1")
	routes = make([][]string, 0)
	for route := range routesChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 3, len(routes))

	routesChan, errChan = rs.ListRoutesFrom(ctx, crypto.FIL, "peer2")
	routes = make([][]string, 0)
	for route := range routesChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 5, len(routes))

	time.Sleep(time.Second + time.Millisecond)

	routesChan, errChan = rs.ListRoutesFrom(ctx, crypto.FIL, "peer1")
	routes = make([][]string, 0)
	for route := range routesChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(routes))

	routesChan, errChan = rs.ListRoutesFrom(ctx, crypto.FIL, "peer2")
	routes = make([][]string, 0)
	for route := range routesChan {
		routes = append(routes, route)
	}
	err = <-errChan
	assert.Nil(t, err)
	assert.Equal(t, 1, len(routes))
}

// getNode is used to create a route store.
func getNode(ctx context.Context) (*crypto.SignerImpl, *RouteStoreImpl, func(), error) {
	// New signer
	signer, err := crypto.NewSignerImpl(ctx, crypto.Opts{Path: fmt.Sprintf("%v/%v", testDS, "signer")})
	if err != nil {
		return nil, nil, nil, err
	}
	prv := make([]byte, 32)
	rand.Read(prv)
	signer.SetKey(ctx, crypto.FIL, crypto.SECP256K1, prv)
	rs, err := NewRouteStoreImpl(ctx, signer, Opts{Path: fmt.Sprintf("%v/%v", testDS, "routestore"), CleanFreq: time.Second})
	if err != nil {
		return nil, nil, nil, err
	}
	shutdown := func() {
		rs.Shutdown()
		signer.Shutdown()
	}
	return signer, rs, shutdown, nil
}
