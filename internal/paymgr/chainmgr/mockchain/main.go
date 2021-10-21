package main

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"fmt"
	"time"

	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/wcgcyx/fcr/internal/paymgr/chainmgr"
)

const (
	blockTime = 15 * time.Second
	port      = 9010
	key       = "9rQWxpExa0uNHBhvqcO5h4A3ZOZbh/1V3bKm/gVOH9bKFix2OEtAsMBB1vbiZRO8UpAoMFq+lHfJqzbnVG74dQ=="
	id        = "12D3KooWPRE8jwmDG6vjxXETqUG2PVMWZ12LuWFYTpcn7QSudJ92"
)

// This is the main program of mock chain.
func main() {
	mc, err := chainmgr.NewMockChainImplV1(blockTime, key, port)
	if err != nil {
		panic(fmt.Errorf("error starting mock chain: %v", err.Error()))
	}
	res := make([]string, 0)
	for _, maddr := range mc.GetAddr().Addrs {
		if manet.IsPrivateAddr(maddr) {
			// Put private addr behind
			res = append(res, maddr.String()+"/p2p/"+mc.GetAddr().ID.String())
		} else {
			// Put public addr ahead
			res = append([]string{maddr.String() + "/p2p/" + mc.GetAddr().ID.String()}, res...)
		}
	}
	for _, addr := range res {
		fmt.Println(addr)
	}
	for {
		// Loop forever
	}
}
