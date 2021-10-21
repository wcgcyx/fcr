package payservstore

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	testDB = "./testdb"
)

func TestNewPayServStore(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	ss, err := NewPayServStoreImplV1(ctx, testDB, []uint64{0, 1, 2})
	assert.Empty(t, err)
	assert.NotEmpty(t, ss)
	cancel()
	time.Sleep(1 * time.Second)
}

func TestAddServing(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	ctx, cancel := context.WithCancel(context.Background())
	ss, err := NewPayServStoreImplV1(ctx, testDB, []uint64{0, 1, 2})
	assert.Empty(t, err)
	assert.NotEmpty(t, ss)

	err = ss.Serve(3, "addr1", big.NewInt(1), big.NewInt(1001))
	assert.NotEmpty(t, err)

	err = ss.Serve(0, "addr1", big.NewInt(1), big.NewInt(1001))
	assert.Empty(t, err)

	err = ss.Serve(1, "addr1", big.NewInt(1), big.NewInt(1001))
	assert.Empty(t, err)

	err = ss.Serve(1, "addr2", big.NewInt(2), big.NewInt(1002))
	assert.Empty(t, err)

	err = ss.Serve(2, "addr3", big.NewInt(3), big.NewInt(1003))
	assert.Empty(t, err)

	err = ss.Serve(0, "addr1", big.NewInt(1), big.NewInt(1001))
	assert.NotEmpty(t, err)

	err = ss.Retire(3, "addr1", time.Minute)
	assert.NotEmpty(t, err)

	err = ss.Retire(0, "addr3", time.Minute)
	assert.NotEmpty(t, err)

	err = ss.Retire(0, "addr1", time.Minute)
	assert.Empty(t, err)

	err = ss.Serve(0, "addr1", big.NewInt(1), big.NewInt(1001))
	assert.NotEmpty(t, err)

	_, _, _, err = ss.Inspect(3, "addr1")
	assert.NotEmpty(t, err)

	_, _, _, err = ss.Inspect(0, "addr3")
	assert.NotEmpty(t, err)

	ppp, period, exp, err := ss.Inspect(0, "addr1")
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1), ppp)
	assert.Equal(t, big.NewInt(1001), period)
	assert.NotEmpty(t, exp)

	ppp, period, exp, err = ss.Inspect(1, "addr1")
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1), ppp)
	assert.Equal(t, big.NewInt(1001), period)
	assert.Empty(t, exp)

	res, err := ss.ListActive(ctx)
	assert.Empty(t, err)
	assert.Empty(t, res[0])
	assert.ElementsMatch(t, []string{"addr1", "addr2"}, res[1])
	assert.ElementsMatch(t, []string{"addr3"}, res[2])

	res, err = ss.ListRetiring(ctx)
	assert.Empty(t, err)
	assert.ElementsMatch(t, []string{"addr1"}, res[0])
	assert.Empty(t, res[1])
	assert.Empty(t, res[2])

	err = ss.Retire(2, "addr3", time.Second)
	assert.Empty(t, err)
	_, _, _, err = ss.Inspect(2, "addr3")
	assert.Empty(t, err)
	time.Sleep(2 * time.Second)
	_, _, _, err = ss.Inspect(2, "addr3")
	assert.NotEmpty(t, err)

	cancel()
	time.Sleep(1 * time.Second)
}
