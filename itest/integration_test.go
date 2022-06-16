package itest

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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNetworkConnection(t *testing.T) {
	// Init
	t.Log("Set key...")
	assert.Nil(t, initKey([]string{
		"provider0", "provider1", "provider2", "provider3",
		"provider4", "provider5", "provider6", "provider7",
		"broker0", "broker1", "broker2", "broker3",
		"broker4", "broker5", "broker6", "broker7",
		"user0", "user1", "user2", "user3",
		"user4", "user5", "user6", "user7",
		"user8", "user9", "user10", "user11",
		"user12", "user13", "user14", "user15",
	}))

	// Set policy
	t.Log("Set policy...")
	assert.Nil(t, setPolicy([]string{
		"provider0", "provider1", "provider2", "provider3",
		"provider4", "provider5", "provider6", "provider7",
		"broker0", "broker1", "broker2", "broker3",
		"broker4", "broker5", "broker6", "broker7",
	}))

	// Connect peers, create paychs and serve paychs.
	// Note, these connections are in order, so that every route should be
	// discovered.
	t.Log("Connect network...")
	// Region 1
	t.Log("Connect region 1 internal...")
	assert.Nil(t, connect("broker0", map[string]int{
		"provider0": 1,
		"provider1": 3,
		"provider2": 3,
	}, true))
	assert.Nil(t, connect("broker1", map[string]int{
		"provider1": 3,
		"provider2": 3,
	}, true))
	assert.Nil(t, connect("broker1", map[string]int{
		"broker0": 1,
	}, true))
	// Region 2
	t.Log("Connect region 2 internal...")
	assert.Nil(t, connect("broker2", map[string]int{
		"provider3": 2,
		"provider4": 2,
	}, true))
	assert.Nil(t, connect("broker3", map[string]int{
		"provider3": 2,
		"provider4": 2,
		"provider5": 1,
	}, true))
	assert.Nil(t, connect("broker2", map[string]int{
		"broker3": 1,
	}, true))
	// Region 3
	t.Log("Connect region 3 internal...")
	assert.Nil(t, connect("broker4", map[string]int{
		"provider6": 2,
		"provider7": 1,
	}, true))
	// Region 1 and 2
	t.Log("Connect region 1 and 2...")
	assert.Nil(t, connect("broker5", map[string]int{
		"broker1": 4,
	}, true))
	assert.Nil(t, connect("broker5", map[string]int{
		"broker2": 4,
	}, true))
	// Region 2 and 3
	t.Log("Connect region 2 and 3...")
	assert.Nil(t, connect("broker6", map[string]int{
		"broker4": 3,
	}, true))
	assert.Nil(t, connect("broker6", map[string]int{
		"broker3": 3,
	}, true))
	// Region 1 and 3
	t.Log("Connect region 1 and 3...")
	assert.Nil(t, connect("broker7", map[string]int{
		"broker5": 3,
	}, true))
	assert.Nil(t, connect("broker7", map[string]int{
		"broker6": 3,
	}, true))
	assert.Nil(t, connect("broker5", map[string]int{
		"broker7": 3,
	}, true))
	assert.Nil(t, connect("broker6", map[string]int{
		"broker7": 3,
	}, true))
	// Region 1
	t.Log("Extend region 1...")
	assert.Nil(t, connect("broker1", map[string]int{
		"broker5": 4,
	}, true))
	assert.Nil(t, connect("broker0", map[string]int{
		"broker1": 1,
	}, true))
	// Region 2
	t.Log("Extend region 2...")
	assert.Nil(t, connect("broker2", map[string]int{
		"broker5": 4,
	}, true))
	assert.Nil(t, connect("broker3", map[string]int{
		"broker6": 4,
	}, true))
	assert.Nil(t, connect("broker3", map[string]int{
		"broker2": 1,
	}, true))
	// Region 3
	t.Log("Extend region 3...")
	assert.Nil(t, connect("broker4", map[string]int{
		"broker6": 3,
	}, true))
	// Users
	t.Log("Connect users...")
	for _, node := range []string{"user0", "user1", "user2", "user3", "user4", "user5"} {
		assert.Nil(t, connect(node, map[string]int{
			"broker0": 1,
			"broker1": 1,
		}, false))
	}
	for _, node := range []string{"user6", "user7", "user8", "user9", "user10", "user11"} {
		assert.Nil(t, connect(node, map[string]int{
			"broker2": 1,
			"broker3": 1,
		}, false))
	}
	for _, node := range []string{"user12", "user13", "user14", "user15"} {
		assert.Nil(t, connect(node, map[string]int{
			"broker4": 1,
		}, false))
	}
	// Done.
	t.Log("Done")
}

func TestGenerateFiles(t *testing.T) {
	t.Log("Generate pieces in region 1...")
	err := generate("provider0", 3)
	assert.Nil(t, err)
	err = generate("provider1", 12)
	assert.Nil(t, err)
	err = generate("provider2", 12)
	assert.Nil(t, err)

	t.Log("Generate pieces in region 2...")
	err = generate("provider3", 9)
	assert.Nil(t, err)
	err = generate("provider4", 9)
	assert.Nil(t, err)
	err = generate("provider5", 3)
	assert.Nil(t, err)

	t.Log("Generate pieces in region 3...")
	err = generate("provider6", 12)
	assert.Nil(t, err)
	err = generate("provider7", 3)
	assert.Nil(t, err)
	// Done.
	t.Log("Done")
}

func TestRetrieve(t *testing.T) {
	// Load pieces
	t.Log("Load pieces...")
	totalImported := make(map[int][]string)

	// Load pieces from region 1
	imported, err := load("provider0")
	assert.Nil(t, err)
	totalImported[0] = imported
	imported, err = load("provider1")
	assert.Nil(t, err)
	totalImported[0] = append(totalImported[0], imported...)
	imported, err = load("provider2")
	assert.Nil(t, err)
	totalImported[0] = append(totalImported[0], imported...)

	// Load pieces from region 2
	imported, err = load("provider3")
	assert.Nil(t, err)
	totalImported[1] = imported
	imported, err = load("provider4")
	assert.Nil(t, err)
	totalImported[1] = append(totalImported[1], imported...)
	imported, err = load("provider5")
	assert.Nil(t, err)
	totalImported[1] = append(totalImported[1], imported...)

	// Load pieces from region 3
	imported, err = load("provider6")
	assert.Nil(t, err)
	totalImported[2] = imported
	imported, err = load("provider7")
	assert.Nil(t, err)
	totalImported[2] = append(totalImported[2], imported...)

	t.Log("Start concurrent retrievals...")
	wg := sync.WaitGroup{}
	wg.Add(16)
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user0", totalImported, 8, 3, 1)) }()
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user1", totalImported, 8, 3, 1)) }()
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user2", totalImported, 8, 3, 1)) }()
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user3", totalImported, 4, 1, 1)) }()
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user4", totalImported, 4, 1, 1)) }()
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user5", totalImported, 4, 1, 1)) }()
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user6", totalImported, 4, 7, 1)) }()
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user7", totalImported, 4, 7, 1)) }()
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user8", totalImported, 4, 7, 1)) }()
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user9", totalImported, 1, 3, 1)) }()
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user10", totalImported, 1, 3, 1)) }()
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user11", totalImported, 1, 3, 1)) }()
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user12", totalImported, 3, 2, 5)) }()
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user13", totalImported, 3, 2, 5)) }()
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user14", totalImported, 1, 1, 2)) }()
	go func() { defer wg.Done(); assert.Nil(t, retrieve("user15", totalImported, 1, 1, 2)) }()
	wg.Wait()
	// Done.
	t.Log("Done")
}
