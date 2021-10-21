package config

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	TestDir = "./testdir"
)

func TestRoundTrip(t *testing.T) {
	os.RemoveAll(TestDir)
	os.Mkdir(TestDir, os.ModePerm)
	defer os.RemoveAll(TestDir)
	conf := Config{
		RootPath:     "test1",
		MinerKey:     "test2",
		P2PPort:      999,
		APIPort:      999,
		EnableCur0:   true,
		ServIDCur0:   "id0",
		ServAddrCur0: "addr0",
		KeyCur0:      "key",
		MaxHopCur0:   10,
	}
	err := conf.SaveToPath(TestDir)
	assert.Empty(t, err)
	conf2, err := ReadFromPath(TestDir)
	assert.Empty(t, err)
	assert.Equal(t, conf, conf2)
}
