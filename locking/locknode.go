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
	"fmt"

	golock "github.com/viney-shih/go-lock"
)

// LockNode is a node in the lock tree.
// It is used to provide multi-dimensional read and write lock.
type LockNode struct {
	lock     golock.RWMutex
	children map[string]*LockNode
}

// CreateLockNode creates a lock node.
// It will create a linked list of lock node based on given path and return the root node.
//
// @input - path.
//
// @output - root node.
func CreateLockNode(path ...interface{}) *LockNode {
	node := &LockNode{
		lock:     golock.NewCASMutex(),
		children: make(map[string]*LockNode),
	}
	if len(path) > 0 {
		// Create node recursively.
		key := fmt.Sprintf("%v", path[0])
		node.children[key] = CreateLockNode(path[1:]...)
	}
	return node
}

// RLock read locks a given path.
//
// @input - context, path.
//
// @output - function to release lock, error.
func (node *LockNode) RLock(ctx context.Context, path ...interface{}) (func(), error) {
	if len(path) == 0 {
		// This is the leaf node.
		if !node.lock.RTryLockWithContext(ctx) {
			return nil, fmt.Errorf("fail to RLock at leaf node.")
		}
		return node.lock.RUnlock, nil
	}
	if !node.lock.RTryLockWithContext(ctx) {
		return nil, fmt.Errorf("fail to RLock with remaining path %v", path)
	}
	// RLock on this node is successful, try to get child.
	key := fmt.Sprintf("%v", path[0])
	child, ok := node.children[key]
	if !ok {
		// Child does not exist, RLock so far has been successful.
		return node.lock.RUnlock, nil
	}
	// Child does exist. Try to RLock child.
	release, err := child.RLock(ctx, path[1:]...)
	if err != nil {
		// RLock child failed.
		defer node.lock.RUnlock()
		return nil, fmt.Errorf("%v - %v", path[0], err.Error())
	}
	// RLock child succeed.
	return func() { release(); node.lock.RUnlock() }, nil
}

// Lock write locks a given path.
//
// @input - context, path.
//
// @output - function to release lock, error.
func (node *LockNode) Lock(ctx context.Context, path ...interface{}) (func(), error) {
	if len(path) == 0 {
		// This is the leaf node.
		if !node.lock.TryLockWithContext(ctx) {
			return nil, fmt.Errorf("fail to Lock at leaf node.")
		}
		return node.lock.Unlock, nil
	}
	if !node.lock.RTryLockWithContext(ctx) {
		return nil, fmt.Errorf("fail to RLock with remaining path %v for Lock", path)
	}
	// RLock on this node is successful, try to get child.
	key := fmt.Sprintf("%v", path[0])
	child, ok := node.children[key]
	if !ok {
		// Child does not exist, switch from RLock to Lock
		node.lock.RUnlock()
		if !node.lock.TryLockWithContext(ctx) {
			return nil, fmt.Errorf("fail to Lock with remaining path %v", path)
		}
		// Lock succeed. Check again.
		_, ok = node.children[key]
		if ok {
			defer node.lock.Unlock()
			return nil, fmt.Errorf("fail to Lock with remaining path %v as child inserted between RUnlock and Lock", path)
		}
		// RLock so far has been successful.
		return node.lock.Unlock, nil
	}
	// Child does exist. Try to Lock child.
	release, err := child.Lock(ctx, path[1:]...)
	if err != nil {
		// Lock child failed.
		defer node.lock.RUnlock()
		return nil, fmt.Errorf("%v - %v", path[0], err.Error())
	}
	// Lock child succeed.
	return func() { release(); node.lock.RUnlock() }, nil
}

// Insert inserts a path to a given node.
// Note: In multiprocess, Lock(path) must be called prior to this function call.
//
// @input - path.
func (node *LockNode) Insert(path ...interface{}) {
	if len(path) == 0 {
		// Path existed.
		return
	}
	key := fmt.Sprintf("%v", path[0])
	child, ok := node.children[key]
	if ok {
		child.Insert(path[1:]...)
		return
	}
	node.children[key] = CreateLockNode(path[1:]...)
}

// Remove removes a path from a given node.
// Note: In multiprocess, Lock(path) must be called prior to this function call.
//
// @input - path.
func (node *LockNode) Remove(path ...interface{}) {
	if len(path) == 0 {
		// Reached leaf node.
		return
	}
	key := fmt.Sprintf("%v", path[0])
	if len(path) == 1 {
		delete(node.children, key)
		return
	}
	node.children[key].Remove(path[1:]...)
}
