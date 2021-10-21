package initpath

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	crypto2 "github.com/libp2p/go-libp2p-core/crypto"

	"github.com/wcgcyx/fcr/internal/config"
	"github.com/wcgcyx/fcr/internal/crypto"
	"github.com/wcgcyx/fcr/internal/paymgr/chainmgr"
)

const (
	p2pKeyFilename = "p2pkey"
	defaultP2PPort = 19955
	defaultAPIPort = 9424
	defaultMaxHop  = 5
)

// InitPath initialise the root directory.
// It takes a path, a debug mode as arguments.
// It returns error.
func InitPath(path string, debug bool) error {
	// First check if given path exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("path %v has already exists", path)
	}
	// Create the config file by prompting the user.
	conf := config.Config{}
	conf.RootPath = path
	// Ask for miner's key
	minerKey := ""
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Do you have a linked miner? (yes/no):")
	input, _ := reader.ReadString('\n')
	input = input[:len(input)-1]
	if input == "yes" {
		fmt.Println("Please provide miner private key in base64 string:")
		input, _ = reader.ReadString('\n')
		input = input[:len(input)-1]
		minerKey = input
	}
	conf.MinerKey = minerKey
	// Ask for p2p port
	p2pPort := defaultP2PPort
	fmt.Printf("Do you want to set the p2p port? Or it will be set to default port of %v. (yes/no):\n", defaultP2PPort)
	input, _ = reader.ReadString('\n')
	input = input[:len(input)-1]
	if input == "yes" {
		fmt.Println("Please provide p2p port:")
		input, _ = reader.ReadString('\n')
		input = input[:len(input)-1]
		port, err := strconv.Atoi(input)
		if err != nil {
			return fmt.Errorf("error parsing port: %v", err.Error())
		}
		if port < 0 || port > 65535 {
			return fmt.Errorf("port out of range of 0-65535")
		}
		p2pPort = port
	}
	conf.P2PPort = p2pPort
	// Ask for API Port
	apiPort := defaultAPIPort
	fmt.Printf("Do you want to set the api port? Or it will be set to default port of %v. (yes/no):\n", defaultAPIPort)
	input, _ = reader.ReadString('\n')
	input = input[:len(input)-1]
	if input == "yes" {
		fmt.Println("Please provide api port:")
		input, _ = reader.ReadString('\n')
		input = input[:len(input)-1]
		port, err := strconv.Atoi(input)
		if err != nil {
			return fmt.Errorf("error parsing port: %v", err.Error())
		}
		if port < 0 || port > 65535 {
			return fmt.Errorf("port out of range of 0-65535")
		}
		apiPort = port
	}
	conf.APIPort = apiPort
	if debug {
		// Ask for currency 0
		fmt.Printf("Do you want to enable currency for debug. (yes/no):\n")
		input, _ = reader.ReadString('\n')
		input = input[:len(input)-1]
		if input == "yes" {
			fmt.Printf("Please provide the serv p2p addr of debug chain\n")
			input, _ = reader.ReadString('\n')
			input = input[:len(input)-1]
			p2pAddrs := strings.Split(input, "/p2p/")
			conf.ServAddrCur0 = p2pAddrs[0]
			conf.ServIDCur0 = p2pAddrs[1]
			// Generate a key
			key, addr, err := crypto.GenerateKeyPair()
			if err != nil {
				return fmt.Errorf("error generating key: %v", err.Error())
			}
			conf.KeyCur0 = base64.StdEncoding.EncodeToString(key)
			fmt.Printf("Account generated with address: %v\n", addr)
			conf.MaxHopCur0 = 10
			conf.EnableCur0 = true
		}
	}
	// Ask for currency id 1
	fmt.Printf("Do you want to enable currency for %v. (yes/no):\n", chainmgr.CurrencyID1Name)
	input, _ = reader.ReadString('\n')
	input = input[:len(input)-1]
	if input == "yes" {
		fmt.Printf("Please provide the access path (url) of %v\n", chainmgr.CurrencyID1Name)
		input, _ = reader.ReadString('\n')
		input = input[:len(input)-1]
		conf.ServAddrCur1 = input
		fmt.Printf("Please provide the access token of %v\n", chainmgr.CurrencyID1Name)
		input, _ = reader.ReadString('\n')
		input = input[:len(input)-1]
		conf.ServIDCur1 = input
		fmt.Println("Do you have an existing account with to use? Or a new key will be generated (yes/no):")
		input, _ = reader.ReadString('\n')
		input = input[:len(input)-1]
		if input == "yes" {
			fmt.Println("Please provide account private key in base64 string:")
			input, _ = reader.ReadString('\n')
			input = input[:len(input)-1]
			conf.KeyCur1 = input
		} else {
			key, addr, err := crypto.GenerateKeyPair()
			if err != nil {
				return fmt.Errorf("error generating key: %v", err.Error())
			}
			conf.KeyCur1 = base64.StdEncoding.EncodeToString(key)
			fmt.Printf("Account generated with address: %v\n", addr)
		}
		conf.MaxHopCur1 = defaultMaxHop
		fmt.Printf("Do you want to set the max hop for payment network? Or it will be set to default of %v. (yes/no)\n", defaultMaxHop)
		input, _ = reader.ReadString('\n')
		input = input[:len(input)-1]
		if input == "yes" {
			fmt.Println("Please provide max hop:")
			input, _ = reader.ReadString('\n')
			input = input[:len(input)-1]
			maxHop, err := strconv.Atoi(input)
			if err != nil {
				return fmt.Errorf("error parsing max hop: %v", err.Error())
			}
			conf.MaxHopCur1 = maxHop
		}
		conf.EnableCur1 = true
	}
	// Ask for currency id 2
	fmt.Printf("Do you want to enable currency for %v. (yes/no):\n", chainmgr.CurrencyID2Name)
	input, _ = reader.ReadString('\n')
	input = input[:len(input)-1]
	if input == "yes" {
		fmt.Printf("Please provide the access path (url) of %v\n", chainmgr.CurrencyID2Name)
		input, _ = reader.ReadString('\n')
		input = input[:len(input)-1]
		conf.ServAddrCur2 = input
		fmt.Printf("Please provide the access token of %v\n", chainmgr.CurrencyID2Name)
		input, _ = reader.ReadString('\n')
		input = input[:len(input)-1]
		conf.ServIDCur2 = input
		fmt.Println("Do you have an existing account with to use? Or a new key will be generated (yes/no):")
		input, _ = reader.ReadString('\n')
		input = input[:len(input)-1]
		if input == "yes" {
			fmt.Println("Please provide account private key in base64 string:")
			input, _ = reader.ReadString('\n')
			input = input[:len(input)-1]
			conf.KeyCur2 = input
		} else {
			key, addr, err := crypto.GenerateKeyPair()
			if err != nil {
				return fmt.Errorf("error generating key: %v", err.Error())
			}
			conf.KeyCur2 = base64.StdEncoding.EncodeToString(key)
			fmt.Printf("Account generated with address: %v\n", addr)
		}
		conf.MaxHopCur2 = defaultMaxHop
		fmt.Printf("Do you want to set the max hop for payment network? Or it will be set to default of %v. (yes/no)\n", defaultMaxHop)
		input, _ = reader.ReadString('\n')
		input = input[:len(input)-1]
		if input == "yes" {
			fmt.Println("Please provide max hop:")
			input, _ = reader.ReadString('\n')
			input = input[:len(input)-1]
			maxHop, err := strconv.Atoi(input)
			if err != nil {
				return fmt.Errorf("error parsing max hop: %v", err.Error())
			}
			conf.MaxHopCur2 = maxHop
		}
		conf.EnableCur2 = true
	}
	// TODO, Support currency id 3 and currency id 4.
	// Create the path
	if err := os.Mkdir(path, os.ModePerm); err != nil {
		return fmt.Errorf("error creating path %v: %v", path, err.Error())
	}
	// Create a p2p key
	prv, _, err := crypto2.GenerateKeyPair(
		crypto2.Ed25519,
		-1,
	)
	if err != nil {
		return fmt.Errorf("error generating p2p key: %v", err.Error())
	}
	prvBytes, err := prv.Raw()
	if err != nil {
		return fmt.Errorf("error getting private key bytes: %v", err.Error())
	}
	err = ioutil.WriteFile(filepath.Join(path, p2pKeyFilename), []byte(base64.StdEncoding.EncodeToString(prvBytes)), os.ModePerm)
	if err != nil {
		return fmt.Errorf("error saving p2p key: %v", err.Error())
	}
	// Save configuration.
	return conf.SaveToPath(path)
}
