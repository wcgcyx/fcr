package config

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	ConfigFilename = "config.json"
)

// Config is the configuration of the client.
type Config struct {
	// Root DB Path
	RootPath string
	// Linked Miner prv key
	MinerKey string
	// Port number
	P2PPort int
	// API Port number
	APIPort int
	// Currency ID 0 Settings (For payment only)
	EnableCur0   bool
	ServIDCur0   string
	ServAddrCur0 string
	KeyCur0      string
	MaxHopCur0   int
	// Currency ID 1 Setttings (For payment only)
	EnableCur1   bool
	ServIDCur1   string
	ServAddrCur1 string
	KeyCur1      string
	MaxHopCur1   int
	// Currency ID 2 Setttings (For payment only)
	EnableCur2   bool
	ServIDCur2   string
	ServAddrCur2 string
	KeyCur2      string
	MaxHopCur2   int
	// Currency ID 3 Setttings (For payment only)
	EnableCur3   bool
	ServIDCur3   string
	ServAddrCur3 string
	KeyCur3      string
	MaxHopCur3   int
	// Currency ID 4 Setttings (For payment only)
	EnableCur4   bool
	ServIDCur4   string
	ServAddrCur4 string
	KeyCur4      string
	MaxHopCur4   int
}

// SaveToPath saves the configuration to given path.
// It takes a path as the argument.
// It returns error.
func (c Config) SaveToPath(path string) error {
	file, err := os.Create(filepath.Join(path, ConfigFilename))
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "\t")
	return encoder.Encode(&c)
}

// ReadFromPath reads a configuration from given path.
// It takes a path as the argument.
// It returns a configuration and error.
func ReadFromPath(path string) (Config, error) {
	file, err := os.Open(filepath.Join(path, ConfigFilename))
	if err != nil {
		return Config{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	conf := Config{}
	err = decoder.Decode(&conf)
	return conf, err
}
