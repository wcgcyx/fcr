package cli

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
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/fcr/api"
)

// System command.
var SystemCMD = &cli.Command{
	Name:        "system",
	Usage:       "access system functions",
	Description: "This command outputs a list of system functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		SystemAddrCMD,
		SystemConnectCMD,
		SystemPublishCMD,
		SystemCacheSizeCMD,
		SystemCachePruneCMD,
		SystemCleanCMD,
		SystemGCCMD,
	},
}

var SystemAddrCMD = &cli.Command{
	Name:        "addr",
	Usage:       "get the node network address",
	Description: "Get the network address of this node",
	ArgsUsage:   " ",
	Action: func(c *cli.Context) error {
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		pi, err := client.NetAddr(c.Context)
		if err != nil {
			return err
		}
		res := make([]string, 0)
		for _, maddr := range pi.Addrs {
			if manet.IsPrivateAddr(maddr) {
				// Put private addr behind
				res = append(res, maddr.String()+"/p2p/"+pi.ID.String())
			} else {
				res = append([]string{maddr.String() + "/p2p/" + pi.ID.String()}, res...)
			}
		}
		fmt.Println(res)
		return nil
	},
}

var SystemConnectCMD = &cli.Command{
	Name:        "connect",
	Usage:       "connect to peer network address",
	Description: "Attempt to connect to given peer network address",
	ArgsUsage:   "[p2p addr]",
	Action: func(c *cli.Context) error {
		p2pAddrs := strings.Split(c.Args().Get(0), "/p2p/")
		if len(p2pAddrs) != 2 {
			return usageError(c, fmt.Errorf("fail to parse p2p addr, should contain net addr and peer id"))
		}
		maddr, err := multiaddr.NewMultiaddr(p2pAddrs[0])
		if err != nil {
			return fmt.Errorf("error parsing multiaddr: %v", err.Error())
		}
		pid, err := peer.Decode(p2pAddrs[1])
		if err != nil {
			return fmt.Errorf("error parsing peer id: %v", err.Error())
		}
		pi := peer.AddrInfo{Addrs: []multiaddr.Multiaddr{maddr}, ID: pid}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.NetConnect(c.Context, pi)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var SystemPublishCMD = &cli.Command{
	Name:        "publish",
	Usage:       "force publish addr to network",
	Description: "Force the node to publish presence to the network",
	ArgsUsage:   " ",
	Action: func(c *cli.Context) error {
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.AddrProtoPublish(c.Context)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var SystemCacheSizeCMD = &cli.Command{
	Name:        "cache-size",
	Usage:       "get the retrieval cache size",
	Description: "Get the current retrieval cache size",
	ArgsUsage:   " ",
	Action: func(c *cli.Context) error {
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		size, err := client.RetMgrGetRetrievalCacheSize(c.Context)
		if err != nil {
			return err
		}
		fmt.Println(size)
		return nil
	},
}

var SystemCachePruneCMD = &cli.Command{
	Name:        "cache-prune",
	Usage:       "prune the retrieval cache",
	Description: "Prune the current retrieval cache size",
	ArgsUsage:   " ",
	Action: func(c *cli.Context) error {
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.RetMgrCleanRetrievalCache(c.Context)
		if err != nil {
			return err
		}
		fmt.Println("Done")
		return nil
	},
}

var SystemCleanCMD = &cli.Command{
	Name:        "clean",
	Usage:       "run retrieval cleanning",
	Description: "Run cleaning process to clean stuck retrieval process",
	ArgsUsage:   " ",
	Action: func(c *cli.Context) error {
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.RetMgrCleanIncomingProcesses(c.Context)
		if err != nil {
			return err
		}
		err = client.RetMgrCleanOutgoingProcesses(c.Context)
		if err != nil {
			return err
		}
		fmt.Println("Done")
		return nil
	},
}

var SystemGCCMD = &cli.Command{
	Name:        "gc",
	Usage:       "run garbage collection",
	Description: "Force to run garbage collection",
	ArgsUsage:   " ",
	Action: func(c *cli.Context) error {
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.GC()
		if err != nil {
			return err
		}
		fmt.Println("Done")
		return nil
	},
}
