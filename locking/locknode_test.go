package locking

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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLockNode(t *testing.T) {
	node := CreateLockNode("key1", "key1-1", "key1-1-1")
	assert.Equal(t, len(node.children), 1)
	assert.NotNil(t, node.children["key1"])
	assert.Equal(t, len(node.children["key1"].children), 1)
	assert.NotNil(t, node.children["key1"].children["key1-1"])
	assert.Equal(t, len(node.children["key1"].children["key1-1"].children), 1)
	assert.NotNil(t, node.children["key1"].children["key1-1"].children["key1-1-1"])
}

func TestInsert(t *testing.T) {
	node := CreateLockNode()
	assert.Equal(t, 0, len(node.children))

	node.Insert()
	assert.Equal(t, 0, len(node.children))

	node.Insert("key1")
	assert.Equal(t, 1, len(node.children))

	node.Insert("key1")
	assert.Equal(t, 1, len(node.children))

	node.Insert("key2")
	assert.Equal(t, 2, len(node.children))

	node.Insert("key3", "key3-1", "key3-1-1")
	assert.Equal(t, 3, len(node.children))
	assert.NotNil(t, node.children["key3"])
	assert.NotNil(t, node.children["key3"].children["key3-1"])
	assert.NotNil(t, node.children["key3"].children["key3-1"].children["key3-1-1"])

	node.Insert("key3", "key3-1", "key3-1-2")
	assert.Equal(t, 3, len(node.children))
	assert.NotNil(t, node.children["key3"])
	assert.NotNil(t, node.children["key3"].children["key3-1"])
	assert.NotNil(t, node.children["key3"].children["key3-1"].children["key3-1-2"])
}

func TestRemove(t *testing.T) {
	node := CreateLockNode()
	node.Insert("key1", "key1-1", "key1-1-1")
	node.Insert("key1", "key1-1", "key1-1-2")
	node.Insert("key2", "key2-1")
	node.Insert("key2", "key2-2", "key2-2-1")

	node.Remove()
	assert.Equal(t, 2, len(node.children))

	node.Remove("key3")
	assert.Equal(t, 2, len(node.children))

	node.Remove("key1", "key1-1", "key1-1-1", "key1-1-1-1")
	assert.Equal(t, 2, len(node.children))

	assert.NotNil(t, node.children["key1"].children["key1-1"].children["key1-1-2"])
	node.Remove("key1", "key1-1", "key1-1-2")
	assert.Nil(t, node.children["key1"].children["key1-1"].children["key1-1-2"])

	assert.NotNil(t, node.children["key2"])
	node.Remove("key2")
	assert.Nil(t, node.children["key2"])
}

func TestLock(t *testing.T) {
	node := CreateLockNode()
	node.Insert("key1", "key1-1", "key1-1-1")
	node.Insert("key1", "key1-1", "key1-1-2")
	node.Insert("key2", "key2-1")
	node.Insert("key2", "key2-2", "key2-2-1")

	release1, err := node.RLock(context.Background(), "key1", "key1-1", "key1-1-3")
	assert.Nil(t, err)

	release2, err := node.RLock(context.Background(), "key1", "key1-1")
	assert.Nil(t, err)

	release3, err := node.RLock(context.Background(), "key1")
	assert.Nil(t, err)
	defer release3()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = node.Lock(ctx, "key1", "key1-1")
	// fail as key1/key1-1 is RLocked
	assert.NotNil(t, err)

	// release one RLock
	release1()

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = node.Lock(ctx, "key1", "key1-1")
	// still fail as key1/key1-1 is still RLocked
	assert.NotNil(t, err)

	// release one more RLock
	release2()

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	release1, err = node.Lock(ctx, "key1", "key1-1")
	// succeed, no RLock in place
	assert.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = node.RLock(ctx, "key1", "key1-1")
	// fail as key1/key1-1 is being locked
	assert.NotNil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = node.RLock(ctx, "key1", "key1-1", "key1-1-1")
	// path is being locked, so RLocked failed
	assert.NotNil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	// path is being locked
	_, err = node.Lock(ctx, "key1", "key1-1", "key1-1-1")
	assert.NotNil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = node.Lock(ctx, "key1", "key1-3")
	// failed as key1-3 not existed, so attempt to lock key1 still failed
	assert.NotNil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	release4, err := node.Lock(ctx, "key2", "key2-1", "key1-1-1")
	// succeed, no lock for key2
	assert.Nil(t, err)
	release4()

	release1()
}

func TestMultiProcessing(t *testing.T) {
	node := CreateLockNode()
	node.Insert("key1", "key1-1", "key1-1-1")
	node.Insert("key1", "key1-1", "key1-1-2")
	node.Insert("key2", "key2-1", "key2-1-1")
	node.Insert("key2", "key2-2", "key2-2-1")

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		release, err := node.RLock(ctx, "key1", "key1-1", "key1-1-1")
		assert.Nil(t, err)

		time.Sleep(200 * time.Millisecond)
		release()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		release, err := node.RLock(ctx, "key1", "key1-1", "key1-1-2")
		assert.Nil(t, err)

		time.Sleep(200 * time.Millisecond)
		release()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// delay so it won't obtain the lock
		time.Sleep(50 * time.Millisecond)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		_, err := node.Lock(ctx, "key1", "key1-1", "key1-1-2")
		// failed as RLock not released for 50 ms
		assert.NotNil(t, err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// delay so it won't obtain the lock
		time.Sleep(50 * time.Millisecond)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		release, err := node.Lock(ctx, "key1", "key1-1", "key1-1-2")
		// succeed as RLock has been released long ago
		assert.Nil(t, err)

		release()
	}()

	wg.Wait()
}
