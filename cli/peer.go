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
	"math/big"
	"strconv"
	"strings"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/fcr/api"
)

// Peer manager command.
var PeerCMD = &cli.Command{
	Name:        "peer",
	Usage:       "access peer functions",
	Description: "This command outputs a list of peer functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		PeerAddCMD,
		PeerListCMD,
		PeerInspectCMD,
		PeerRemoveCMD,
		PeerBlockCMD,
		PeerUnblockCMD,
		PeerListHistoryCMD,
		PeerRemoveRecordCMD,
		PeerRemoveSetRecIDCMD,
	},
}

var PeerAddCMD = &cli.Command{
	Name:        "add",
	Usage:       "add a peer",
	Description: "Add a new peer or update the peer network address",
	ArgsUsage:   "[currency id, peer addr, p2p addr]",
	Action: func(c *cli.Context) error {
		currency, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		currencyID := byte(currency)
		peerAddr := c.Args().Get(1)
		if peerAddr == "" {
			return usageError(c, fmt.Errorf("received empty peer addr"))
		}
		p2pAddrs := strings.Split(c.Args().Get(2), "/p2p/")
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
		err = client.PeerMgrAddPeer(c.Context, currencyID, peerAddr, pi)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var PeerListCMD = &cli.Command{
	Name:        "list",
	Aliases:     []string{"ls"},
	Usage:       "list peers",
	Description: "List all peers stored in the peer manager",
	ArgsUsage:   " ",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "long",
			Aliases: []string{"l"},
			Usage:   "display details",
		},
	},
	Action: func(c *cli.Context) error {
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		curChan := client.PeerMgrListCurrencyIDs(c.Context)
		for currencyID := range curChan {
			fmt.Printf("Currency %v\n", currencyID)
			peerChan := client.PeerMgrListPeers(c.Context, currencyID)
			for peer := range peerChan {
				if c.IsSet("long") {
					fmt.Printf("\t%v:\n", peer)
					// Get addr
					pi, err := client.PeerMgrPeerAddr(c.Context, currencyID, peer)
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
					fmt.Printf("\t\tNetwork Addrs: %v\n", res)
					// Get if blocked
					blocked, err := client.PeerMgrIsBlocked(c.Context, currencyID, peer)
					if err != nil {
						return err
					}
					fmt.Printf("\t\tBlocked: %v\n", blocked)
					fmt.Printf("\t\tRecent history:\n")
					// Get recent 10 history
					i := 0
					recChan := client.PeerMgrListHistory(c.Context, currencyID, peer)
					for rec := range recChan {
						if i == 10 {
							break
						}
						i++
						fmt.Printf("\t\t\t%v: %v - %v\n", rec.RecID, rec.Record.CreatedAt, rec.Record.Description)
					}
				} else {
					fmt.Printf("\t%v\n", peer)
				}
			}
		}
		fmt.Println("Done")
		return nil
	},
}

var PeerInspectCMD = &cli.Command{
	Name:        "inspect",
	Usage:       "inspect peer details",
	Description: "Inspect the details of a peer",
	ArgsUsage:   "[currency id, peer addr]",
	Action: func(c *cli.Context) error {
		currency, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		currencyID := byte(currency)
		peerAddr := c.Args().Get(1)
		if peerAddr == "" {
			return usageError(c, fmt.Errorf("received empty peer addr"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		fmt.Printf("Peer %v (Currency %v):\n", peerAddr, currency)
		exists, err := client.PeerMgrHasPeer(c.Context, currencyID, peerAddr)
		if err != nil {
			return err
		}
		if !exists {
			fmt.Printf("\tDoes not exist\n")
			return nil
		}
		// Get addr
		pi, err := client.PeerMgrPeerAddr(c.Context, currencyID, peerAddr)
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
		fmt.Printf("\tNetwork Addrs: %v\n", res)
		blocked, err := client.PeerMgrIsBlocked(c.Context, currencyID, peerAddr)
		if err != nil {
			return err
		}
		fmt.Printf("\tBlocked: %v\n", blocked)
		fmt.Printf("\tRecent history:\n")
		// Get recent 10 history
		i := 0
		recChan := client.PeerMgrListHistory(c.Context, currencyID, peerAddr)
		for rec := range recChan {
			if i == 10 {
				break
			}
			i++
			fmt.Printf("\t\t%v: %v - %v\n", rec.RecID, rec.Record.CreatedAt, rec.Record.Description)
		}
		return nil
	},
}

var PeerRemoveCMD = &cli.Command{
	Name:        "remove",
	Usage:       "remove peer and history",
	Description: "Remove a peer and all history recorded",
	ArgsUsage:   "[currency id, peer addr]",
	Action: func(c *cli.Context) error {
		currency, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		currencyID := byte(currency)
		peerAddr := c.Args().Get(1)
		if peerAddr == "" {
			return usageError(c, fmt.Errorf("received empty peer addr"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.PeerMgrRemovePeer(c.Context, currencyID, peerAddr)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var PeerBlockCMD = &cli.Command{
	Name:        "block",
	Usage:       "block peer",
	Description: "Block a peer based on given currency id and peer address",
	ArgsUsage:   "[currency id, peer addr]",
	Action: func(c *cli.Context) error {
		currency, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		currencyID := byte(currency)
		peerAddr := c.Args().Get(1)
		if peerAddr == "" {
			return usageError(c, fmt.Errorf("received empty peer addr"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.PeerMgrBlockPeer(c.Context, currencyID, peerAddr)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var PeerUnblockCMD = &cli.Command{
	Name:        "unblock",
	Usage:       "unblock peer",
	Description: "Unblock a peer based on given currency id and peer address",
	ArgsUsage:   "[currency id, peer addr]",
	Action: func(c *cli.Context) error {
		currency, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		currencyID := byte(currency)
		peerAddr := c.Args().Get(1)
		if peerAddr == "" {
			return usageError(c, fmt.Errorf("received empty peer addr"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.PeerMgrUnblockPeer(c.Context, currencyID, peerAddr)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var PeerListHistoryCMD = &cli.Command{
	Name:        "list-history",
	Usage:       "list history of given peer",
	Description: "List the history of a given peer based on given currency id and peer address",
	ArgsUsage:   "[currency id, peer addr]",
	Action: func(c *cli.Context) error {
		currency, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		currencyID := byte(currency)
		peerAddr := c.Args().Get(1)
		if peerAddr == "" {
			return usageError(c, fmt.Errorf("received empty peer addr"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		recChan := client.PeerMgrListHistory(c.Context, currencyID, peerAddr)
		for rec := range recChan {
			fmt.Printf("%v: %v - %v\n", rec.RecID, rec.Record.CreatedAt, rec.Record.Description)
		}
		fmt.Println("Done")
		return nil
	},
}

var PeerRemoveRecordCMD = &cli.Command{
	Name:        "remove-record",
	Usage:       "remove the record of a peer",
	Description: "Remove a record of a peer based on record id",
	ArgsUsage:   "[currency id, peer addr, record id]",
	Action: func(c *cli.Context) error {
		currency, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		currencyID := byte(currency)
		peerAddr := c.Args().Get(1)
		if peerAddr == "" {
			return usageError(c, fmt.Errorf("received empty peer addr"))
		}
		recID, ok := big.NewInt(0).SetString(c.Args().Get(2), 10)
		if !ok {
			return usageError(c, fmt.Errorf("fail to parse record id"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.PeerMgrRemoveRecord(c.Context, currencyID, peerAddr, recID)
		if err != nil {
			return err
		}
		fmt.Println("Done")
		return nil
	},
}

var PeerRemoveSetRecIDCMD = &cli.Command{
	Name:        "set-recid",
	Usage:       "set the record id of a peer",
	Description: "Set a record id of a peer to be the given record id",
	ArgsUsage:   "[currency id, peer addr, record id]",
	Action: func(c *cli.Context) error {
		currency, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		currencyID := byte(currency)
		peerAddr := c.Args().Get(1)
		if peerAddr == "" {
			return usageError(c, fmt.Errorf("received empty peer addr"))
		}
		recID, ok := big.NewInt(0).SetString(c.Args().Get(2), 10)
		if !ok {
			return usageError(c, fmt.Errorf("fail to parse record id"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.PeerMgrSetRecID(c.Context, currencyID, peerAddr, recID)
		if err != nil {
			return err
		}
		fmt.Println("Done")
		return nil
	},
}
