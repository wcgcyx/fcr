package mdlstore

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
	"os"
	"sync"
	"testing"

	dsquery "github.com/ipfs/go-datastore/query"
	"github.com/stretchr/testify/assert"
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

func TestNewMultiDimLockableStoreImpl(t *testing.T) {
	ctx := context.Background()

	_, err := NewMultiDimLockableStoreImpl(ctx, "")
	assert.NotNil(t, err)

	mdls, err := NewMultiDimLockableStoreImpl(ctx, testDS)
	assert.Nil(t, err)
	defer mdls.Shutdown(ctx)
}

func TestInsert(t *testing.T) {
	ctx := context.Background()

	mdls, err := NewMultiDimLockableStoreImpl(ctx, testDS)
	assert.Nil(t, err)
	defer mdls.Shutdown(ctx)

	txn, err := mdls.NewTransaction(ctx, false)
	assert.Nil(t, err)

	release, err := mdls.Lock(ctx)
	assert.Nil(t, err)

	err = txn.Put(ctx, []byte{1}, "key1", "key1-1", "key1-1-1")
	assert.Nil(t, err)

	err = txn.Put(ctx, []byte{1}, "key1", "key1-1", "key1-1-2")
	assert.Nil(t, err)

	err = txn.Put(ctx, []byte{1}, "key1", "key1-2", "key1-2-1")
	assert.Nil(t, err)

	err = txn.Put(ctx, []byte{1}, "key2", "key2-1", "key2-1-1")
	assert.Nil(t, err)

	err = txn.Put(ctx, []byte{1}, "key2", "key2-2", "key2-2-1")
	assert.Nil(t, err)

	err = txn.Put(ctx, []byte{1}, "key2", "key2-2", "key2-2-2")
	assert.Nil(t, err)

	err = txn.Put(ctx, []byte{1}, "key2", "key2-2", "key2-2-3")
	assert.Nil(t, err)

	err = txn.Put(ctx, []byte{1}, "key3", "key3-1", "key3-1-1")
	assert.Nil(t, err)

	err = txn.Commit(ctx)
	assert.Nil(t, err)
	release()
}

func TestUpdateRemove(t *testing.T) {
	ctx := context.Background()

	mdls, err := NewMultiDimLockableStoreImpl(ctx, testDS)
	assert.Nil(t, err)
	defer mdls.Shutdown(ctx)

	// transaction to update value
	txn, err := mdls.NewTransaction(ctx, false)
	assert.Nil(t, err)

	release1, err := mdls.Lock(ctx, "key1", "key1-1", "key1-1-1")
	assert.Nil(t, err)

	_, err = txn.Get(ctx)
	assert.NotNil(t, err)

	_, err = txn.Get(ctx, "key1", "key1/1", "key1/1/1")
	assert.NotNil(t, err)

	val, err := txn.Get(ctx, "key1", "key1-1", "key1-1-1")
	assert.Nil(t, err)
	assert.Equal(t, []byte{1}, val)

	err = txn.Put(ctx, []byte{2})
	assert.NotNil(t, err)

	err = txn.Put(ctx, []byte{2}, "key1", "key1/1", "key1/1/1")
	assert.NotNil(t, err)

	err = txn.Put(ctx, []byte{2}, "key1", "key1-1", "key1-1-1")
	assert.Nil(t, err)

	val, err = txn.Get(ctx, "key1", "key1-1", "key1-1-1")
	assert.Nil(t, err)
	assert.Equal(t, []byte{2}, val)

	err = txn.Commit(ctx)
	assert.Nil(t, err)
	release1()

	// transaction to delete
	txn, err = mdls.NewTransaction(ctx, false)
	assert.Nil(t, err)

	release, err := mdls.Lock(ctx, "key1", "key1-1")
	assert.Nil(t, err)

	err = txn.Delete(ctx)
	assert.NotNil(t, err)

	err = txn.Delete(ctx, "key1", "key1/1", "key1/1/1")
	assert.NotNil(t, err)

	err = txn.Delete(ctx, "key1", "key1-1", "key1-1-1")
	assert.Nil(t, err)

	err = txn.Commit(ctx)
	assert.Nil(t, err)
	release()

	// transaction to test discard
	txn, err = mdls.NewTransaction(ctx, false)
	assert.Nil(t, err)

	release, err = mdls.Lock(ctx, "key1", "key1-1", "key1-1-1")
	assert.Nil(t, err)

	err = txn.Put(ctx, []byte{1}, "key1", "key1-1", "key1-1-1")
	assert.Nil(t, err)

	txn.Discard(ctx)
	release()

	// transaction to test reading
	txn, err = mdls.NewTransaction(ctx, true)
	assert.Nil(t, err)

	release, err = mdls.RLock(ctx, "key1", "key1-1", "key1-1-1")
	assert.Nil(t, err)

	exists, err := txn.Has(ctx)
	assert.Nil(t, err)
	assert.True(t, exists)

	_, err = txn.Has(ctx, "key1", "key1/1", "key1/1/1")
	assert.NotNil(t, err)

	exists, err = txn.Has(ctx, "key1", "key1-1", "key1-1-1")
	assert.Nil(t, err)
	assert.False(t, exists)
	release()

	// transaction to test query
	txn, err = mdls.NewTransaction(ctx, true)
	assert.Nil(t, err)

	release, err = mdls.RLock(ctx, "key1")
	assert.Nil(t, err)

	_, err = txn.Query(ctx, Query{KeysOnly: true}, "key1/")
	assert.NotNil(t, err)

	res, err := txn.Query(ctx, Query{KeysOnly: true}, "key1")
	tempRes := res.Results
	assert.Nil(t, err)

	output := make(chan string, dsquery.KeysOnlyBufSize)

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
				err = e.Error
				break
			}
			select {
			case output <- e.Key[1:]:
			}
		}
	}()

	children := make([]string, 0)
	for key := range output {
		children = append(children, key)
	}

	assert.ElementsMatch(t, children, []string{"key1/key1-1/key1-1-2", "key1/key1-2/key1-2-1", "key1/key1-1", "key1/key1-2"})
	release()
}

func TestMultiProcessing(t *testing.T) {
	ctx := context.Background()

	mdls, err := NewMultiDimLockableStoreImpl(ctx, testDS)
	assert.Nil(t, err)
	defer mdls.Shutdown(ctx)

	// Multiple threads to read/updates value for every path
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		txn, err := mdls.NewTransaction(ctx, false)
		assert.Nil(t, err)

		release, err := mdls.Lock(ctx, "key1", "key1-1", "key1-1-1")
		assert.Nil(t, err)
		defer release()

		err = txn.Put(ctx, []byte{10}, "key1", "key1-1", "key1-1-1")
		assert.Nil(t, err)

		err = txn.Commit(ctx)
		assert.Nil(t, err)
	}()

	inc := func(wg *sync.WaitGroup, path ...interface{}) {
		defer wg.Done()

		release, err := mdls.Lock(ctx, path...)
		assert.Nil(t, err)
		defer release()

		txn, err := mdls.NewTransaction(ctx, false)
		assert.Nil(t, err)

		val, err := txn.Get(ctx, path...)
		assert.Nil(t, err)

		val[0] += 1
		err = txn.Put(ctx, val, path...)
		assert.Nil(t, err)

		err = txn.Commit(ctx)
		assert.Nil(t, err)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go inc(&wg, "key1", "key1-1", "key1-1-2")
	}

	for i := 0; i < 11; i++ {
		wg.Add(1)
		go inc(&wg, "key1", "key1-2", "key1-2-1")
	}

	for i := 0; i < 12; i++ {
		wg.Add(1)
		go inc(&wg, "key2", "key2-1", "key2-1-1")
	}

	for i := 0; i < 13; i++ {
		wg.Add(1)
		go inc(&wg, "key2", "key2-2", "key2-2-1")
	}

	for i := 0; i < 14; i++ {
		wg.Add(1)
		go inc(&wg, "key2", "key2-2", "key2-2-2")
	}

	for i := 0; i < 15; i++ {
		wg.Add(1)
		go inc(&wg, "key2", "key2-2", "key2-2-3")
	}

	for i := 0; i < 16; i++ {
		wg.Add(1)
		go inc(&wg, "key3", "key3-1", "key3-1-1")
	}

	wg.Wait()

	txn, err := mdls.NewTransaction(ctx, true)
	assert.Nil(t, err)

	val, err := txn.Get(ctx, "key1", "key1-1", "key1-1-1")
	assert.Nil(t, err)
	assert.Equal(t, byte(10), val[0])

	val, err = txn.Get(ctx, "key1", "key1-1", "key1-1-2")
	assert.Nil(t, err)
	assert.Equal(t, byte(11), val[0])

	val, err = txn.Get(ctx, "key1", "key1-2", "key1-2-1")
	assert.Nil(t, err)
	assert.Equal(t, byte(12), val[0])

	val, err = txn.Get(ctx, "key2", "key2-1", "key2-1-1")
	assert.Nil(t, err)
	assert.Equal(t, byte(13), val[0])

	val, err = txn.Get(ctx, "key2", "key2-2", "key2-2-1")
	assert.Nil(t, err)
	assert.Equal(t, byte(14), val[0])

	val, err = txn.Get(ctx, "key2", "key2-2", "key2-2-2")
	assert.Nil(t, err)
	assert.Equal(t, byte(15), val[0])

	val, err = txn.Get(ctx, "key2", "key2-2", "key2-2-3")
	assert.Nil(t, err)
	assert.Equal(t, byte(16), val[0])

	val, err = txn.Get(ctx, "key3", "key3-1", "key3-1-1")
	assert.Nil(t, err)
	assert.Equal(t, byte(17), val[0])

	children, err := txn.GetChildren(ctx)
	assert.Nil(t, err)
	assert.Equal(t, map[string]bool{"key1": true, "key2": true, "key3": true}, children)
}
