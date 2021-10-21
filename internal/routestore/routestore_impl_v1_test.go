package routestore

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	testDB = "./testdb"
)

func TestNewRouteStore(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	rs, err := NewRouteStoreImplV1(ctx, testDB, []uint64{0, 1}, map[uint64]int{0: 3, 1: 4}, map[uint64]string{0: "A", 1: "A"})
	time.Sleep(1 * time.Second)
	assert.Empty(t, err)
	assert.NotEmpty(t, rs)
	assert.Equal(t, 3, rs.MaxHop(0))
	assert.Equal(t, 4, rs.MaxHop(1))
	assert.Equal(t, 0, rs.MaxHop(2))
	cancel()
	time.Sleep(1 * time.Second)
}

func TestAddRoute(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	rs, err := NewRouteStoreImplV1(ctx, testDB, []uint64{0, 1}, map[uint64]int{0: 3, 1: 4}, map[uint64]string{0: "A", 1: "A"})
	time.Sleep(1 * time.Second)
	assert.Empty(t, err)
	assert.NotEmpty(t, rs)

	err = rs.AddDirectLink(3, "B")
	assert.NotEmpty(t, err)
	err = rs.AddDirectLink(0, "B")
	assert.Empty(t, err)
	err = rs.AddDirectLink(0, "B")
	assert.NotEmpty(t, err)

	err = rs.RemoveDirectLink(3, "B")
	assert.NotEmpty(t, err)
	err = rs.RemoveDirectLink(0, "C")
	assert.NotEmpty(t, err)
	err = rs.RemoveDirectLink(0, "B")
	assert.Empty(t, err)
	err = rs.AddDirectLink(0, "B")
	assert.Empty(t, err)
	err = rs.AddDirectLink(0, "C")
	assert.Empty(t, err)
	err = rs.AddDirectLink(1, "C")
	assert.Empty(t, err)
	err = rs.AddDirectLink(1, "D")
	assert.Empty(t, err)

	err = rs.AddRoute(1, []string{"D", "B1", "A"}, time.Hour)
	assert.NotEmpty(t, err)

	err = rs.AddRoute(0, []string{"B", "B1"}, time.Hour)
	assert.Empty(t, err)

	err = rs.AddRoute(0, []string{"C", "B1"}, time.Hour)
	assert.Empty(t, err)

	routes, err := rs.GetRoutesFrom(0, "B")
	assert.ElementsMatch(t, [][]string{{"A", "B"}, {"A", "B", "B1"}}, routes)
	assert.Empty(t, err)

	err = rs.AddRoute(0, []string{"B", "B2", "C"}, time.Hour)
	assert.Empty(t, err)

	routes, err = rs.GetRoutesFrom(0, "B")
	assert.ElementsMatch(t, [][]string{{"A", "B"}, {"A", "B", "B1"}, {"A", "B", "B2"}}, routes)
	assert.Empty(t, err)

	err = rs.AddRoute(1, []string{"D", "D1"}, time.Hour)
	assert.Empty(t, err)

	err = rs.AddRoute(1, []string{"D", "D2", "C"}, time.Hour)
	assert.Empty(t, err)

	err = rs.AddRoute(3, []string{"D", "D2", "D1"}, time.Hour)
	assert.NotEmpty(t, err)

	err = rs.AddRoute(1, []string{"D", "D2", "D1"}, time.Hour)
	assert.Empty(t, err)

	_, err = rs.GetRoutesFrom(1, "B")
	assert.Empty(t, err)

	routes, err = rs.GetRoutesFrom(1, "D")
	assert.ElementsMatch(t, [][]string{{"A", "D"}, {"A", "D", "D1"}, {"A", "D", "D2"}, {"A", "D", "D2", "D1"}}, routes)
	assert.Empty(t, err)

	routes, err = rs.GetRoutesTo(0, "B1")
	assert.ElementsMatch(t, [][]string{{"A", "B", "B1"}, {"A", "C", "B1"}}, routes)
	assert.Empty(t, err)

	err = rs.RemoveDirectLink(0, "B")
	assert.Empty(t, err)

	routes, err = rs.GetRoutesTo(0, "B1")
	assert.ElementsMatch(t, [][]string{{"A", "C", "B1"}}, routes)
	assert.Empty(t, err)

	_, err = rs.GetRoutesFrom(0, "B")
	assert.Empty(t, err)

	cancel()
	time.Sleep(1 * time.Second)
}

func TestGC(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	rs, err := NewRouteStoreImplV1(ctx, testDB, []uint64{0, 1}, map[uint64]int{0: 4, 1: 4}, map[uint64]string{0: "A", 1: "A"})
	time.Sleep(1 * time.Second)
	assert.Empty(t, err)
	assert.NotEmpty(t, rs)

	err = rs.AddDirectLink(0, "B")
	assert.Empty(t, err)

	err = rs.AddRoute(0, []string{"B", "C", "D"}, time.Second)
	assert.Empty(t, err)
	err = rs.AddRoute(0, []string{"B", "C", "E"}, time.Minute)
	assert.Empty(t, err)
	err = rs.AddRoute(0, []string{"B", "F", "D"}, time.Minute)
	assert.Empty(t, err)

	routes, err := rs.GetRoutesFrom(0, "B")
	assert.Empty(t, err)
	assert.ElementsMatch(t, [][]string{{"A", "B"}, {"A", "B", "C", "D"}, {"A", "B", "C", "E"}, {"A", "B", "F", "D"}, {"A", "B", "C"}, {"A", "B", "F"}}, routes)

	cancel()
	time.Sleep(2 * time.Second)

	ctx, cancel = context.WithCancel(context.Background())
	rs, err = NewRouteStoreImplV1(ctx, testDB, []uint64{0, 1}, map[uint64]int{0: 4, 1: 4}, map[uint64]string{0: "A", 1: "A"})
	time.Sleep(1 * time.Second)

	routes, err = rs.GetRoutesFrom(0, "B")
	assert.Empty(t, err)
	assert.ElementsMatch(t, [][]string{{"A", "B"}, {"A", "B", "C", "E"}, {"A", "B", "F", "D"}, {"A", "B", "C"}, {"A", "B", "F"}}, routes)

	cancel()
	time.Sleep(1 * time.Second)
}
