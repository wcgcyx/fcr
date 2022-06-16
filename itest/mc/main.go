package main

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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/filecoin-project/specs-actors/v7/actors/builtin/paych"
	"github.com/wcgcyx/fcr/trans"
)

type History struct {
	Height   int                 `json:"height"`
	Balances map[string]*big.Int `json:"bals"`
	States   map[string][]byte   `json:"states"`
}

// This is the main program for mock chain.
func main() {
	l, err := net.Listen("tcp", "0.0.0.0:9010")
	if err != nil {
		fmt.Println(err.Error())
	}
	mc := trans.NewMockFil(1 * time.Second)
	temp := mc.Server.Listener
	mc.Server.Listener = l
	temp.Close()

	// Start the server
	defer mc.Server.Close()

	// Try to load previous data from ./mc.data
	data, err := os.ReadFile("./mc.data")
	if err == nil {
		history := History{}
		err := json.Unmarshal(data, &history)
		if err != nil {
			fmt.Printf("Fail to load previous data: %v, continue 0.\n", err.Error())
		} else {
			states := make(map[string]*paych.State)
			for key, val := range history.States {
				state := &paych.State{}
				err = state.UnmarshalCBOR(bytes.NewReader(val))
				if err != nil {
					fmt.Printf("Fail to load previous data: %v, continue 1.\n", err.Error())
				}
				states[key] = state
			}
			mc.Lock.Lock()
			mc.Height = history.Height
			mc.PaychBals = history.Balances
			mc.PaychStats = states
			mc.Lock.Unlock()
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	fmt.Printf("Mock chain served at: %v\n", mc.GetAPI())
	// Loop forever.
	for {
		// Loop forever, until exit
		<-c
		break
	}
	fmt.Println("Graceful shutdown mc...")
	mc.Lock.Lock()
	// Save data to ./mc.data
	history := History{}
	history.Height = mc.Height
	history.Balances = mc.PaychBals
	states := make(map[string][]byte)
	for key, val := range mc.PaychStats {
		var b bytes.Buffer
		tmp := bufio.NewWriter(&b)
		err := val.MarshalCBOR(tmp)
		if err != nil {
			fmt.Printf("Has error saving data: %v, exit 0\n", err.Error())
			return
		}
		tmp.Flush()
		states[key] = b.Bytes()
	}
	history.States = states
	data, err = json.Marshal(history)
	if err != nil {
		fmt.Printf("Has error saving data: %v, exit 1\n", err.Error())
		return
	}
	err = os.WriteFile("./mc.data", data, os.ModePerm)
	if err != nil {
		fmt.Printf("Has error saving data: %v, exit 2\n", err.Error())
		return
	}
}
