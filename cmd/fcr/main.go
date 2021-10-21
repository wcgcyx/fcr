package main

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"bufio"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/fcr/internal/api"
	"github.com/wcgcyx/fcr/internal/cidoffer"
	"github.com/wcgcyx/fcr/internal/config"
	"github.com/wcgcyx/fcr/internal/daemon"
	"github.com/wcgcyx/fcr/internal/initpath"
	"github.com/wcgcyx/fcr/internal/payoffer"
	"github.com/wcgcyx/fcr/internal/version"
)

const (
	DefaultFCRPath    = ".fcr/"
	DefaultFCRPathEnv = "FCR_PATH"
)

var log = logging.Logger("main")

// This is the main program of fcr.
func main() {
	// Get path
	home, err := os.UserHomeDir()
	if err != nil {
		log.Errorf("error getting home dir: %v", err.Error())
		return
	}
	path := filepath.Join(home, DefaultFCRPath)
	if os.Getenv(DefaultFCRPathEnv) != "" {
		path = os.Getenv(DefaultFCRPathEnv)
	}
	// New cli
	app := &cli.App{
		Name:     "fcr",
		HelpName: "fcr",
		Version:  version.Version,
		Usage:    "a filecoin secondary retrieval client",
		Description: "\n\t This is a filecoin secondary retrieval client featured with\n" +
			"\t the ability to participate in an ipld retrieval network and\n" +
			"\t a payment proxy network.\n\n" +
			"\t You can earn filecoin (FIL) by importing a regular file or\n" +
			"\t a .car file or a lotus unsealed sector and serving the ipld\n" +
			"\t to the network for paid retrieval.\n\n" +
			"\t -OR-\n\n" +
			"\t You can earn the surcharge by creating a payment channel and\n" +
			"\t serving the payment channel to the network for others to do \n" +
			"\t a proxy payment.\n",
		Authors: []*cli.Author{
			{
				Name:  "wcgcyx",
				Email: "wcgcyx@gmail.com",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Usage:   "specifiy the `FCR_PATH` to use for the commands",
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "init",
				Usage:     "Perform an initialisation to FCR_PATH",
				ArgsUsage: " ",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:   "debug",
						Usage:  "set to true for debug mode, enable currency id 0",
						Hidden: true,
					},
				},
				Action: func(c *cli.Context) error {
					if c.String("path") != "" {
						path = c.String("path")
					}
					debug := c.Bool("debug")
					return initpath.InitPath(path, debug)
				},
			},
			{
				Name:      "daemon",
				Usage:     "Start the daemon from FCR_PATH",
				ArgsUsage: " ",
				Action: func(c *cli.Context) error {
					if c.String("path") != "" {
						path = c.String("path")
					}
					return daemon.Daemon(c.Context, path)
				},
			},
			{
				Name:      "cidnet",
				Usage:     "Access cid network functions",
				ArgsUsage: " ",
				Subcommands: []*cli.Command{
					{
						Name:      "query-offer",
						Usage:     "Query the network for a cid offer",
						ArgsUsage: "[currency id, cid, max]",
						Action: func(c *cli.Context) error {
							currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
							if err != nil {
								return fmt.Errorf("error parsing currency id: %v", err.Error())
							}
							id := c.Args().Get(1)
							if id == "" {
								return fmt.Errorf("got empty cid")
							}
							max, err := strconv.ParseInt(c.Args().Get(2), 10, 64)
							if err != nil {
								return fmt.Errorf("error parsing max: %v", err.Error())
							}
							conf, err := config.ReadFromPath(path)
							if err != nil {
								return err
							}
							offers, err := api.RequestCIDNetQueryOffer(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, id, int(max))
							if err != nil {
								return err
							}
							fmt.Printf("Found offers for currency id %v and cid %v\n", currencyID, id)
							for i, offer := range offers {
								fmt.Printf("Offer %v:\n", i)
								fmt.Printf("\t peer addr: %v\n", offer.PeerAddr().String())
								fmt.Printf("\t root: %v\n", offer.Root().String())
								fmt.Printf("\t currency id: %v\n", offer.CurrencyID())
								fmt.Printf("\t price per byte: %v\b", offer.PPB().String())
								fmt.Printf("\t size: %v\n", offer.Size())
								fmt.Printf("\t recipient addr: %v\n", offer.ToAddr())
								fmt.Printf("\t linked miner addr: %v\n", offer.LinkedMiner())
								fmt.Printf("\t expiration: %v\n", offer.Expiration())
							}
							return nil
						},
					},
					{
						Name:      "piece",
						Usage:     "Access piece manager functions",
						ArgsUsage: " ",
						Subcommands: []*cli.Command{
							{
								Name:      "import",
								Usage:     "Import a file into the piece manager",
								ArgsUsage: "[filename]",
								Action: func(c *cli.Context) error {
									filename := c.Args().Get(0)
									if filename == "" {
										return fmt.Errorf("got empty filename")
									}
									filenameAbs, err := filepath.Abs(filename)
									if err != nil {
										return fmt.Errorf("error getting path: %v", err.Error())
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									id, err := api.RequestCIDNetPieceImport(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), filenameAbs)
									if err != nil {
										return err
									}
									fmt.Printf("Imported cid: %v\n", id)
									return nil
								},
							},
							{
								Name:      "import-car",
								Usage:     "Import a .car file into the piece manager",
								ArgsUsage: "[filename]",
								Action: func(c *cli.Context) error {
									filename := c.Args().Get(0)
									if filename == "" {
										return fmt.Errorf("got empty filename")
									}
									filenameAbs, err := filepath.Abs(filename)
									if err != nil {
										return fmt.Errorf("error getting path: %v", err.Error())
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									id, err := api.RequestCIDNetPieceImportCar(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), filenameAbs)
									if err != nil {
										return err
									}
									fmt.Printf("Imported cid: %v\n", id)
									return nil
								},
							},
							{
								Name:      "import-sector",
								Usage:     "Import a lotus unsealed sector into the piece manager",
								ArgsUsage: "[filename, copy]",
								Action: func(c *cli.Context) error {
									filename := c.Args().Get(0)
									if filename == "" {
										return fmt.Errorf("got empty filename")
									}
									filenameAbs, err := filepath.Abs(filename)
									if err != nil {
										return fmt.Errorf("error getting path: %v", err.Error())
									}
									copy, err := strconv.ParseBool(c.Args().Get(1))
									if err != nil {
										return fmt.Errorf("error parsing copy: %v", err.Error())
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									ids, err := api.RequestCIDNetPieceImportSector(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), filenameAbs, copy)
									if err != nil {
										return err
									}
									fmt.Printf("Imported cids: %v\n", ids)
									return nil
								},
							},
							{
								Name:      "list",
								Usage:     "List all imported cids from the piece manager",
								ArgsUsage: " ",
								Action: func(c *cli.Context) error {
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									ids, err := api.RequestCIDNetPieceList(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort))
									if err != nil {
										return err
									}
									fmt.Println("Imported cids:")
									for _, id := range ids {
										fmt.Printf("\t %v\n", id)
									}
									return nil
								},
							},
							{
								Name:      "inspect",
								Usage:     "Inspect the details of a cid from the piece manager",
								ArgsUsage: "[cid]",
								Action: func(c *cli.Context) error {
									id := c.Args().Get(0)
									if id == "" {
										return fmt.Errorf("got empty cid")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									path, index, size, copy, err := api.RequestCIDNetPieceInspect(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), id)
									if err != nil {
										return err
									}
									fmt.Printf("cid %v:\n", id)
									fmt.Printf("\t path: %v\n", path)
									fmt.Printf("\t index: %v\n", index)
									fmt.Printf("\t size: %v\n", size)
									fmt.Printf("\t copy: %v\n", copy)
									return nil
								},
							},
							{
								Name:      "remove",
								Usage:     "Remove a cid from the piece manager",
								ArgsUsage: "[cid]",
								Action: func(c *cli.Context) error {
									id := c.Args().Get(0)
									if id == "" {
										return fmt.Errorf("got empty cid")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestCIDNetPieceRemove(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), id)
									if err != nil {
										return err
									}
									fmt.Printf("cid %v removed\n", id)
									return nil
								},
							},
						},
					},
					{
						Name:      "serving",
						Usage:     "Access cid serving manager functions",
						ArgsUsage: " ",
						Subcommands: []*cli.Command{
							{
								Name:      "serve",
								Usage:     "Start serving a cid to the network",
								ArgsUsage: "[currency id, cid, ppb (price per byte)]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									id := c.Args().Get(1)
									if id == "" {
										return fmt.Errorf("got empty cid")
									}
									ppb := c.Args().Get(2)
									if ppb == "" {
										return fmt.Errorf("got empty ppb")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestCIDNetServingServe(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, id, ppb)
									if err != nil {
										return err
									}
									fmt.Printf("Served %v at %v with currency id %v\n", id, ppb, currencyID)
									return nil
								},
							},
							{
								Name:      "list",
								Usage:     "List all cid servings",
								ArgsUsage: " ",
								Action: func(c *cli.Context) error {
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									res, err := api.RequestCIDNetServingList(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort))
									if err != nil {
										return err
									}
									fmt.Println("Currently serving:")
									for currencyID, ids := range res {
										fmt.Printf("\t Currency ID: %v\n", currencyID)
										for _, id := range ids {
											fmt.Printf("\t\t %v\n", id)
										}
									}
									return nil
								},
							},
							{
								Name:      "inspect",
								Usage:     "Inspect the details of a cid serving",
								ArgsUsage: "[currency id, cid]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									id := c.Args().Get(1)
									if id == "" {
										return fmt.Errorf("got empty cid")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									ppb, expiration, err := api.RequestCIDNetServingInspect(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, id)
									if err != nil {
										return err
									}
									fmt.Printf("Serving %v has currency id %v price per byte of %v and expiration of %v\n", id, currencyID, ppb.String(), expiration)
									return nil
								},
							},
							{
								Name:      "retire",
								Usage:     "Stop serving a cid to the network",
								ArgsUsage: "[currency id, cid]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									id := c.Args().Get(1)
									if id == "" {
										return fmt.Errorf("got empty cid")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestCIDNetServingRetire(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, id)
									if err != nil {
										return err
									}
									fmt.Printf("Retired %v for currency id %v\n", id, currencyID)
									return nil
								},
							},
							{
								Name:      "force-publish",
								Usage:     "Force publish of all cid servings to the network immediately",
								ArgsUsage: " ",
								Action: func(c *cli.Context) error {
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestCIDNetServingForcePublish(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort))
									if err != nil {
										return err
									}
									fmt.Println("Force publish servings done")
									return nil
								},
							},
							{
								Name:      "list-in-use",
								ArgsUsage: " ",
								Usage:     "List all cids that are currently in use",
								Action: func(c *cli.Context) error {
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									ids, err := api.RequestCIDNetServingListInUse(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort))
									if err != nil {
										return err
									}
									fmt.Println("In use cids:")
									for _, id := range ids {
										fmt.Printf("\t %v\n", id)
									}
									return nil
								},
							},
						},
					},
					{
						Name:      "peer",
						Usage:     "Access retrieval peer manager functions",
						ArgsUsage: " ",
						Subcommands: []*cli.Command{
							{
								Name:      "list",
								Usage:     "List all retrieval peers",
								ArgsUsage: " ",
								Action: func(c *cli.Context) error {
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									res, err := api.RequestCIDNetPeerList(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort))
									if err != nil {
										return err
									}
									fmt.Println("Current peers:")
									for currencyID, ids := range res {
										fmt.Printf("\t Currency ID: %v\n", currencyID)
										for _, id := range ids {
											fmt.Printf("\t\t %v\n", id)
										}
									}
									return nil
								},
							},
							{
								Name:      "inspect",
								Usage:     "Inspect the details of a retrieval peer",
								ArgsUsage: "[currency id, recipient addr]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									toAddr := c.Args().Get(1)
									if toAddr == "" {
										return fmt.Errorf("got empty recipient addr")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									peerID, blocked, success, fail, err := api.RequestCIDNetPeerInspect(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, toAddr)
									if err != nil {
										return err
									}
									fmt.Printf("Peer %v with currency id %v has a peer id: %v, blocked: %v, success count: %v, failed count: %v\n", toAddr, currencyID, peerID, blocked, success, fail)
									return nil
								},
							},
							{
								Name:      "block",
								Usage:     "Block a retrieval peer",
								ArgsUsage: "[currency id, recipient addr]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									toAddr := c.Args().Get(1)
									if toAddr == "" {
										return fmt.Errorf("got empty recipient addr")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestCIDNetPeerBlock(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, toAddr)
									if err != nil {
										return err
									}
									fmt.Printf("Blocked peer %v with currency id %v\n", toAddr, currencyID)
									return nil
								},
							},
							{
								Name:      "unblock",
								Usage:     "Unblock a retrieval peer",
								ArgsUsage: "[currency id, recipient addr]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									toAddr := c.Args().Get(1)
									if toAddr == "" {
										return fmt.Errorf("got empty recipient addr")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestCIDNetPeerUnblock(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, toAddr)
									if err != nil {
										return err
									}
									fmt.Printf("Unblocked peer %v with currency id %v\n", toAddr, currencyID)
									return nil
								},
							},
						},
					},
				},
			},
			{
				Name:      "paynet",
				Usage:     "Access payment network functions",
				ArgsUsage: " ",
				Subcommands: []*cli.Command{
					{
						Name:      "root",
						Usage:     "Obtain the root address",
						ArgsUsage: "[currency id]",
						Action: func(c *cli.Context) error {
							currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
							if err != nil {
								return fmt.Errorf("error parsing currency id: %v", err.Error())
							}
							conf, err := config.ReadFromPath(path)
							if err != nil {
								return err
							}
							root, err := api.RequestPayNetRoot(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID)
							if err != nil {
								return err
							}
							fmt.Printf("Root address for currency id %v is: %v\n", currencyID, root)
							return nil
						},
					},
					{
						Name:      "outbound",
						Usage:     "Access outbound channel functions",
						ArgsUsage: " ",
						Subcommands: []*cli.Command{
							{
								Name:      "create",
								Usage:     "Create an outbound payment channel",
								ArgsUsage: "[currency id, recipient addr, amount, peer p2p addr]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									toAddr := c.Args().Get(1)
									if toAddr == "" {
										return fmt.Errorf("got empty recipient addr")
									}
									amt := c.Args().Get(2)
									if amt == "" {
										return fmt.Errorf("got empty amount")
									}
									p2pAddrs := strings.Split(c.Args().Get(3), "/p2p/")
									if len(p2pAddrs) != 2 {
										return fmt.Errorf("error parsing p2p addrs, should contain net addr and peer id")
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
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									offer, err := api.RequestPayNetOutboundQueryPaych(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, toAddr, pi)
									if err != nil {
										return err
									}
									// Got an offer
									fmt.Println("Received paych offer:")
									fmt.Printf("\t Currency ID: %v\n", offer.CurrencyID())
									fmt.Printf("\t Recipient Addr: %v\n", offer.Addr())
									fmt.Printf("\t Settlement: %v\n", offer.Settlement())
									fmt.Printf("\t Expiration: %v\n", offer.Expiration())
									fmt.Println("Do you want to use this offer to create a payment channel? (yes/no):")
									reader := bufio.NewReader(os.Stdin)
									input, _ := reader.ReadString('\n')
									input = input[:len(input)-1]
									if input == "yes" {
										err = api.RequestPayNetOutboundCreate(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), offer, amt, pi)
										if err != nil {
											return err
										}
										fmt.Println("Payment channel created")
									} else {
										return fmt.Errorf("operation cancelled by user")
									}
									return nil
								},
							},
							{
								Name:      "topup",
								Usage:     "Topup an outbound payment channel",
								ArgsUsage: "[currency id, recipient addr, amount]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									toAddr := c.Args().Get(1)
									if toAddr == "" {
										return fmt.Errorf("got empty recipient addr")
									}
									amt := c.Args().Get(2)
									if amt == "" {
										return fmt.Errorf("got empty amount")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestPayNetOutboundTopup(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, toAddr, amt)
									if err != nil {
										return err
									}
									fmt.Println("Topup succeed")
									return nil
								},
							},
							{
								Name:      "list",
								Usage:     "List all outbound payment channels",
								ArgsUsage: " ",
								Action: func(c *cli.Context) error {
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									res, err := api.RequestPayNetOutboundList(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort))
									if err != nil {
										return err
									}
									fmt.Println("Current outbound channels:")
									for currencyID, ids := range res {
										fmt.Printf("\t Currency ID: %v\n", currencyID)
										for _, id := range ids {
											fmt.Printf("\t\t %v\n", id)
										}
									}
									return nil
								},
							},
							{
								Name:      "inspect",
								Usage:     "Inspect the details of an outbound payment channel",
								ArgsUsage: "[currency id, payment channel addr]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									chAddr := c.Args().Get(1)
									if chAddr == "" {
										return fmt.Errorf("got empty channel addr")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									toAddr, redeemed, balance, settlement, active, curHeight, settlingAt, err := api.RequestPayNetOutboundInspect(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, chAddr)
									if err != nil {
										return err
									}
									fmt.Printf("Outbound payment channel %v with currency id %v:\n", chAddr, currencyID)
									fmt.Printf("\t Recipient addr: %v\n", toAddr)
									fmt.Printf("\t Redeemed: %v\n", redeemed.String())
									fmt.Printf("\t Balance: %v\n", balance.String())
									fmt.Printf("\t Settlement: %v\n", settlement)
									fmt.Printf("\t Active: %v\n", active)
									fmt.Printf("\t Settling at: %v\n", settlingAt)
									fmt.Printf("\t Current chain height: %v\n", curHeight)
									return nil
								},
							},
							{
								Name:      "settle",
								Usage:     "Settle an outbound payment channel",
								ArgsUsage: "[currency id, payment channel addr]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									chAddr := c.Args().Get(1)
									if chAddr == "" {
										return fmt.Errorf("got empty channel addr")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestPayNetOutboundSettle(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, chAddr)
									if err != nil {
										return err
									}
									fmt.Println("Settle succeed")
									return nil
								},
							},
							{
								Name:      "collect",
								Usage:     "Collect an outbound payment channel",
								ArgsUsage: "[currency id, payment channel addr]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									chAddr := c.Args().Get(1)
									if chAddr == "" {
										return fmt.Errorf("got empty channel addr")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestPayNetOutboundCollect(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, chAddr)
									if err != nil {
										return err
									}
									fmt.Println("Collect succeed")
									return nil
								},
							},
						},
					},
					{
						Name:      "inbound",
						Usage:     "Access inbound channel functions",
						ArgsUsage: " ",
						Subcommands: []*cli.Command{
							{
								Name:      "list",
								Usage:     "List all inbound payment channels",
								ArgsUsage: " ",
								Action: func(c *cli.Context) error {
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									res, err := api.RequestPayNetInboundList(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort))
									if err != nil {
										return err
									}
									fmt.Println("Current inbound channels:")
									for currencyID, ids := range res {
										fmt.Printf("\t Currency ID: %v\n", currencyID)
										for _, id := range ids {
											fmt.Printf("\t\t %v\n", id)
										}
									}
									return nil
								},
							},
							{
								Name:      "inspect",
								Usage:     "Inspect the details of an inbound payment channel",
								ArgsUsage: "[currency id, payment channel addr]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									chAddr := c.Args().Get(1)
									if chAddr == "" {
										return fmt.Errorf("got empty channel addr")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									toAddr, redeemed, balance, settlement, active, curHeight, settlingAt, err := api.RequestPayNetInboundInspect(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, chAddr)
									if err != nil {
										return err
									}
									fmt.Printf("Inbound payment channel %v with currency id %v:\n", chAddr, currencyID)
									fmt.Printf("\t Recipient addr: %v\n", toAddr)
									fmt.Printf("\t Redeemed: %v\n", redeemed.String())
									fmt.Printf("\t Balance: %v\n", balance.String())
									fmt.Printf("\t Settlement: %v\n", settlement)
									fmt.Printf("\t Active: %v\n", active)
									fmt.Printf("\t Settling at: %v\n", settlingAt)
									fmt.Printf("\t Current chain height: %v\n", curHeight)
									return nil
								},
							},
							{
								Name:      "settle",
								Usage:     "Settle an inbound payment channel",
								ArgsUsage: "[currency id, payment channel addr]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									chAddr := c.Args().Get(1)
									if chAddr == "" {
										return fmt.Errorf("got empty channel addr")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestPayNetInboundSettle(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, chAddr)
									if err != nil {
										return err
									}
									fmt.Println("Settle succeed")
									return nil
								},
							},
							{
								Name:      "collect",
								Usage:     "Collect an inbound payment channel",
								ArgsUsage: "[currency id, payment channel addr]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									chAddr := c.Args().Get(1)
									if chAddr == "" {
										return fmt.Errorf("got empty channel addr")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestPayNetInboundCollect(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, chAddr)
									if err != nil {
										return err
									}
									fmt.Println("Collect succeed")
									return nil
								},
							},
						},
					},
					{
						Name:      "serving",
						Usage:     "Access paych serving manager functions",
						ArgsUsage: " ",
						Subcommands: []*cli.Command{
							{
								Name:      "serve",
								Usage:     "Start serving a paych to the network",
								ArgsUsage: "[currency id, recipient addr, ppp (price per period), period]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									toAddr := c.Args().Get(1)
									if toAddr == "" {
										return fmt.Errorf("got empty recipient addr")
									}
									ppp := c.Args().Get(2)
									if ppp == "" {
										return fmt.Errorf("got empty ppp")
									}
									period := c.Args().Get(3)
									if period == "" {
										return fmt.Errorf("got empty period")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestPayNetServingServe(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, toAddr, ppp, period)
									if err != nil {
										return err
									}
									fmt.Printf("Served %v at %v every %v with currency id %v\n", toAddr, ppp, period, currencyID)
									return nil
								},
							},
							{
								Name:      "list",
								Usage:     "List all paych servings",
								ArgsUsage: " ",
								Action: func(c *cli.Context) error {
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									res, err := api.RequestPayNetServingList(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort))
									if err != nil {
										return err
									}
									fmt.Println("Currently serving:")
									for currencyID, ids := range res {
										fmt.Printf("\t Currency ID: %v\n", currencyID)
										for _, id := range ids {
											fmt.Printf("\t\t %v\n", id)
										}
									}
									return nil
								},
							},
							{
								Name:      "inspect",
								Usage:     "Inspect the details of a paych serving",
								ArgsUsage: "[currency id, recipient addr]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									toAddr := c.Args().Get(1)
									if toAddr == "" {
										return fmt.Errorf("got empty recipient addr")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									ppp, period, expiration, err := api.RequestPayNetServingInspect(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, toAddr)
									if err != nil {
										return err
									}
									fmt.Printf("Serving %v has currency id %v price per byte of %v period of %v and expiration of %v\n", toAddr, currencyID, ppp.String(), period.String(), expiration)
									return nil
								},
							},
							{
								Name:      "retire",
								Usage:     "Stop serving a paych to the network",
								ArgsUsage: "[currency id, recipient addr]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									toAddr := c.Args().Get(1)
									if toAddr == "" {
										return fmt.Errorf("got empty recipient addr")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestPayNetServingRetire(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, toAddr)
									if err != nil {
										return err
									}
									fmt.Printf("Retired %v for currency id %v\n", toAddr, currencyID)
									return nil
								},
							},
							{
								Name:      "force-publish",
								ArgsUsage: " ",
								Usage:     "Force publish of all paych servings to the network immediately",
								Action: func(c *cli.Context) error {
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestPayNetServingForcePublish(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort))
									if err != nil {
										return err
									}
									fmt.Println("Force publish servings done")
									return nil
								},
							},
						},
					},
					{
						Name:      "peer",
						Usage:     "Access payment peer manager functions",
						ArgsUsage: " ",
						Subcommands: []*cli.Command{
							{
								Name:      "list",
								Usage:     "List all payment peers",
								ArgsUsage: " ",
								Action: func(c *cli.Context) error {
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									res, err := api.RequestPayNetPeerList(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort))
									if err != nil {
										return err
									}
									fmt.Println("Current peers:")
									for currencyID, ids := range res {
										fmt.Printf("\t Currency ID: %v\n", currencyID)
										for _, id := range ids {
											fmt.Printf("\t\t %v\n", id)
										}
									}
									return nil
								},
							},
							{
								Name:      "inspect",
								Usage:     "Inspect the details of a payment peer",
								ArgsUsage: "[currency id, recipient addr]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									toAddr := c.Args().Get(1)
									if toAddr == "" {
										return fmt.Errorf("got empty recipient addr")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									peerID, blocked, success, fail, err := api.RequestPayNetPeerInspect(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, toAddr)
									if err != nil {
										return err
									}
									fmt.Printf("Peer %v with currency id %v has a peer id: %v, blocked: %v, success count: %v, failed count: %v\n", toAddr, currencyID, peerID, blocked, success, fail)
									return nil
								},
							},
							{
								Name:      "block",
								Usage:     "Block a payment peer",
								ArgsUsage: "[currency id, recipient addr]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									toAddr := c.Args().Get(1)
									if toAddr == "" {
										return fmt.Errorf("got empty recipient addr")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestPayNetPeerBlock(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, toAddr)
									if err != nil {
										return err
									}
									fmt.Printf("Blocked peer %v with currency id %v\n", toAddr, currencyID)
									return nil
								},
							},
							{
								Name:      "unblock",
								Usage:     "Unblock a payment peer",
								ArgsUsage: "[currency id, recipient addr]",
								Action: func(c *cli.Context) error {
									currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
									if err != nil {
										return fmt.Errorf("error parsing currency id: %v", err.Error())
									}
									toAddr := c.Args().Get(1)
									if toAddr == "" {
										return fmt.Errorf("got empty recipient addr")
									}
									conf, err := config.ReadFromPath(path)
									if err != nil {
										return err
									}
									err = api.RequestPayNetPeerUnblock(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, toAddr)
									if err != nil {
										return err
									}
									fmt.Printf("Unblocked peer %v with currency id %v\n", toAddr, currencyID)
									return nil
								},
							},
						},
					},
				},
			},
			{
				Name:      "system",
				Usage:     "Access system functions",
				ArgsUsage: " ",
				Subcommands: []*cli.Command{
					{
						Name:      "addr",
						Usage:     "Get the network addr of this node",
						ArgsUsage: " ",
						Action: func(c *cli.Context) error {
							conf, err := config.ReadFromPath(path)
							if err != nil {
								return err
							}
							addrs, err := api.RequestSystemAddr(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort))
							if err != nil {
								return err
							}
							fmt.Println("System addrs are:")
							for _, addr := range addrs {
								fmt.Printf("\t %v\n", addr)
							}
							return nil
						},
					},
					{
						Name:      "bootstrap",
						Usage:     "Connect to a bootstrap node",
						ArgsUsage: "[peer p2p addr]",
						Action: func(c *cli.Context) error {
							p2pAddrs := strings.Split(c.Args().Get(0), "/p2p/")
							if len(p2pAddrs) != 2 {
								return fmt.Errorf("error parsing p2p addrs, should contain net addr and peer id")
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
							conf, err := config.ReadFromPath(path)
							if err != nil {
								return err
							}
							err = api.RequestSystemBootstrap(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), pi)
							if err != nil {
								return err
							}
							fmt.Println("Bootstrap succeed")
							return nil
						},
					},
					{
						Name:      "cache-size",
						Usage:     "Get the currenct retrieval cache DB size",
						ArgsUsage: " ",
						Action: func(c *cli.Context) error {
							conf, err := config.ReadFromPath(path)
							if err != nil {
								return err
							}
							size, err := api.RequestSystemCacheSize(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort))
							if err != nil {
								return err
							}
							fmt.Printf("System cache size is: %v\n", size)
							return nil
						},
					},
					{
						Name:      "cache-prune",
						Usage:     "Clean the currenct retrieval cache DB",
						ArgsUsage: " ",
						Action: func(c *cli.Context) error {
							conf, err := config.ReadFromPath(path)
							if err != nil {
								return err
							}
							err = api.RequestSystemCachePrune(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort))
							if err != nil {
								return err
							}
							fmt.Println("System cache prune succeed")
							return nil
						},
					},
				},
			},
			{
				Name:      "retrieve",
				Usage:     "Retrieve a cid from the network",
				ArgsUsage: "[currency id, cid, outpath]",
				Action: func(c *cli.Context) error {
					currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
					if err != nil {
						return fmt.Errorf("error parsing currency id: %v", err.Error())
					}
					id := c.Args().Get(1)
					if id == "" {
						return fmt.Errorf("got empty cid")
					}
					outPath := c.Args().Get(2)
					if outPath == "" {
						return fmt.Errorf("got empty outpath")
					}
					outPathAbs, err := filepath.Abs(outPath)
					if err != nil {
						return fmt.Errorf("error getting path: %v", err.Error())
					}
					conf, err := config.ReadFromPath(path)
					if err != nil {
						return err
					}
					// First try to retrieve from cache
					found, err := api.RequestRetrievalCache(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), id, outPath)
					if err != nil {
						return err
					}
					if found {
						fmt.Println("Retrieval done through cache")
						return nil
					}
					fmt.Println("Search offers from connected peers")
					offers, err := api.RequestCIDNetQueryConnectedOffer(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, id)
					if err != nil {
						return err
					}
					reader := bufio.NewReader(os.Stdin)
					if len(offers) > 0 {
						fmt.Printf("Found offers for currency id %v and cid %v\n", currencyID, id)
						for i, offer := range offers {
							fmt.Printf("Offer %v:\n", i)
							fmt.Printf("\t peer addr: %v\n", offer.PeerAddr().String())
							fmt.Printf("\t root: %v\n", offer.Root().String())
							fmt.Printf("\t currency id: %v\n", offer.CurrencyID())
							fmt.Printf("\t price per byte: %v\b", offer.PPB().String())
							fmt.Printf("\t size: %v\n", offer.Size())
							fmt.Printf("\t recipient addr: %v\n", offer.ToAddr())
							fmt.Printf("\t linked miner addr: %v\n", offer.LinkedMiner())
							fmt.Printf("\t expiration: %v\n", offer.Expiration())
						}
						fmt.Println("Which offer do you want to use or skip? (offer.no, skip)")
						input, _ := reader.ReadString('\n')
						input = input[:len(input)-1]
						if input != "skip" {
							// Parse int
							index, err := strconv.ParseInt(input, 10, 64)
							if err != nil {
								return err
							}
							if int(index) >= len(offers) || index < 0 {
								return fmt.Errorf("invalid index %v", index)
							}
							cidOffer := &offers[index]
							// Start retrieving directly
							fmt.Println("Start retrieving")
							err = api.RequestRetrieval(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), cidOffer, nil, outPathAbs)
							if err != nil {
								return err
							}
							fmt.Println("Retrieval done")
							return nil
						}
					}
					fmt.Println("Search offers from DHT network")
					offers, err = api.RequestCIDNetQueryOffer(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, id, 5) // Just retrieve 5 offers for now
					if err != nil {
						return err
					}
					fmt.Printf("Found %v offers, start searching corresponding pay offers\n", len(offers))
					payOffers := make([][]payoffer.PayOffer, 0)
					for _, offer := range offers {
						pOffers, err := api.RequestPayNetQueryOffer(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, offer.ToAddr(), big.NewInt(0).Mul(offer.PPB(), big.NewInt(offer.Size())).String())
						if err != nil {
							return err
						}
						payOffers = append(payOffers, pOffers)
					}
					fmt.Printf("Found offers for currency id %v and cid %v\n", currencyID, id)
					for i, offer := range offers {
						fmt.Printf("Offer %v:\n", i)
						fmt.Printf("\t peer addr: %v\n", offer.PeerAddr().String())
						fmt.Printf("\t root: %v\n", offer.Root().String())
						fmt.Printf("\t currency id: %v\n", offer.CurrencyID())
						fmt.Printf("\t price per byte: %v\b", offer.PPB().String())
						fmt.Printf("\t size: %v\n", offer.Size())
						fmt.Printf("\t recipient addr: %v\n", offer.ToAddr())
						fmt.Printf("\t linked miner addr: %v\n", offer.LinkedMiner())
						fmt.Printf("\t expiration: %v\n", offer.Expiration())
						fmt.Printf("\t corresponding pay offers:\n")
						if len(payOffers[i]) > 0 {
							for j, pOffer := range payOffers[i] {
								fmt.Printf("\t\t Payoffer %v:\n", j)
								fmt.Printf("\t\t\t from %v\n", pOffer.From())
								fmt.Printf("\t\t\t ppp %v\n", pOffer.PPP())
								fmt.Printf("\t\t\t period %v\n", pOffer.Period())
								fmt.Printf("\t\t\t expiration %v\n", pOffer.Expiration())
							}
						} else {
							fmt.Printf("\t\t Not found, need to create a payment channel\n")
						}
					}
					fmt.Println("Which offer do you want to use or skip? (offer.no, skip)")
					input, _ := reader.ReadString('\n')
					input = input[:len(input)-1]
					if input != "skip" {
						// Parse int
						index, err := strconv.ParseInt(input, 10, 64)
						if err != nil {
							return err
						}
						if int(index) >= len(offers) || index < 0 {
							return fmt.Errorf("invalid index %v", index)
						}
						cidOffer := &offers[index]
						pOffers := payOffers[index]
						// Ask for pay offers
						if len(pOffers) > 0 {
							fmt.Println("Which pay offer do you want to use for this cid offer? (offer.no)")
							input, _ := reader.ReadString('\n')
							input = input[:len(input)-1]
							// Parse int
							indexP, err := strconv.ParseInt(input, 10, 64)
							if err != nil {
								return err
							}
							if int(indexP) >= len(pOffers) || indexP < 0 {
								return fmt.Errorf("invalid index %v", indexP)
							}
							pOffer := pOffers[indexP]
							// Start retrieving with proxy payment
							fmt.Println("Start retrieving")
							err = api.RequestRetrieval(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), cidOffer, &pOffer, outPathAbs)
							if err != nil {
								return err
							}
							fmt.Println("Retrieval done")
						} else {
							fmt.Println("No payoffer found, do you want to create a payment channel for this offer to retrieve? (yes/no)")
							input, _ := reader.ReadString('\n')
							input = input[:len(input)-1]
							if input == "yes" {
								// Query paych offer
								paychOffer, err := api.RequestPayNetOutboundQueryPaych(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), cidOffer.CurrencyID(), cidOffer.ToAddr(), cidOffer.PeerAddr())
								if err != nil {
									return err
								}
								fmt.Println("Received paych offer:")
								fmt.Printf("\t Currency ID: %v\n", paychOffer.CurrencyID())
								fmt.Printf("\t Recipient Addr: %v\n", paychOffer.Addr())
								fmt.Printf("\t Settlement: %v\n", paychOffer.Settlement())
								fmt.Printf("\t Expiration: %v\n", paychOffer.Expiration())
								fmt.Println("Do you want to use this offer to create a payment channel? (yes/no):")
								input, _ := reader.ReadString('\n')
								input = input[:len(input)-1]
								if input == "yes" {
									min := big.NewInt(0).Mul(cidOffer.PPB(), big.NewInt(cidOffer.Size()))
									fmt.Printf("Please provide the amount used to create the channel, minimum: %v:\n", min.String())
									input, _ := reader.ReadString('\n')
									input = input[:len(input)-1]
									err = api.RequestPayNetOutboundCreate(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), paychOffer, input, cidOffer.PeerAddr())
									if err != nil {
										return err
									}
									fmt.Println("Payment channel created")
									fmt.Println("Start retrieving")
									err = api.RequestRetrieval(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), cidOffer, nil, outPathAbs)
									if err != nil {
										return err
									}
									fmt.Println("Retrieval done")
								}
							}
						}
						return nil
					}
					return nil
				},
			},
			{
				Name:      "fast-retrieve",
				Usage:     "Fast retrieve a cid from the network using a pre-set strategy",
				ArgsUsage: "[currency id, cid, outpath, max amount]",
				Action: func(c *cli.Context) error {
					currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 64)
					if err != nil {
						return fmt.Errorf("error parsing currency id: %v", err.Error())
					}
					id := c.Args().Get(1)
					if id == "" {
						return fmt.Errorf("got empty cid")
					}
					outPath := c.Args().Get(2)
					if outPath == "" {
						return fmt.Errorf("got empty outpath")
					}
					outPathAbs, err := filepath.Abs(outPath)
					if err != nil {
						return fmt.Errorf("error getting path: %v", err.Error())
					}
					maxStr := c.Args().Get(3)
					if maxStr == "" {
						return fmt.Errorf("got empty max amount")
					}
					max, ok := big.NewInt(0).SetString(maxStr, 10)
					if !ok {
						return fmt.Errorf("error parsing max amount")
					}
					conf, err := config.ReadFromPath(path)
					if err != nil {
						return err
					}
					// First try to retrieve from cache
					found, err := api.RequestRetrievalCache(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), id, outPath)
					if err != nil {
						return err
					}
					if found {
						fmt.Println("Retrieval done through cache")
						return nil
					}
					fmt.Println("Search offers from connected peers")
					offers, err := api.RequestCIDNetQueryConnectedOffer(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, id)
					if err != nil {
						return err
					}
					if len(offers) > 0 {
						var cidOffer *cidoffer.CIDOffer
						var curPPB *big.Int
						for _, offer := range offers {
							if curPPB == nil {
								curPPB = offer.PPB()
								cidOffer = &offer
							} else {
								if offer.PPB().Cmp(curPPB) < 0 {
									curPPB = offer.PPB()
									cidOffer = &offer
								}
							}
						}
						// Check if passed max amount
						if max.Cmp(big.NewInt(0)) <= 0 || max.Cmp(big.NewInt(0).Mul(cidOffer.PPB(), big.NewInt(cidOffer.Size()))) >= 0 {
							fmt.Println("Start retrieving using offer:")
							fmt.Printf("\t peer addr: %v\n", cidOffer.PeerAddr().String())
							fmt.Printf("\t root: %v\n", cidOffer.Root().String())
							fmt.Printf("\t currency id: %v\n", cidOffer.CurrencyID())
							fmt.Printf("\t price per byte: %v\b", cidOffer.PPB().String())
							fmt.Printf("\t size: %v\n", cidOffer.Size())
							fmt.Printf("\t recipient addr: %v\n", cidOffer.ToAddr())
							fmt.Printf("\t linked miner addr: %v\n", cidOffer.LinkedMiner())
							fmt.Printf("\t expiration: %v\n", cidOffer.Expiration())
							err = api.RequestRetrieval(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), cidOffer, nil, outPathAbs)
							if err != nil {
								return err
							}
							fmt.Println("Retrieval done")
							return nil
						}
					}
					fmt.Println("Search offers from DHT network")
					offers, err = api.RequestCIDNetQueryOffer(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, id, 5) // Just retrieve 5 offers for now
					if err != nil {
						return err
					}
					fmt.Printf("Found %v offers, start searching corresponding pay offers\n", len(offers))
					payOffers := make([][]payoffer.PayOffer, 0)
					for _, offer := range offers {
						pOffers, err := api.RequestPayNetQueryOffer(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), currencyID, offer.ToAddr(), big.NewInt(0).Mul(offer.PPB(), big.NewInt(offer.Size())).String())
						if err != nil {
							return err
						}
						payOffers = append(payOffers, pOffers)
					}
					var cidOffer *cidoffer.CIDOffer
					var payOffer *payoffer.PayOffer
					var curPrice *big.Int
					for i, offer := range offers {
						pOffers := payOffers[i]
						total := big.NewInt(0).Mul(offer.PPB(), big.NewInt(offer.Size()))
						for _, pOffer := range pOffers {
							temp := big.NewInt(0).Sub(total, big.NewInt(1))
							temp = big.NewInt(0).Div(temp, pOffer.Period())
							temp = big.NewInt(0).Add(temp, big.NewInt(1))
							required := big.NewInt(0).Add(total, big.NewInt(0).Mul(temp, pOffer.PPP()))
							if curPrice == nil {
								cidOffer = &offer
								payOffer = &pOffer
								curPrice = required
							} else {
								if required.Cmp(curPrice) < 0 {
									cidOffer = &offer
									payOffer = &pOffer
									curPrice = required
								}
							}
						}
					}
					if curPrice != nil && (max.Cmp(big.NewInt(0)) <= 0 || max.Cmp(curPrice) >= 0) {
						fmt.Println("Start retrieving using offer:")
						fmt.Printf("\t peer addr: %v\n", cidOffer.PeerAddr().String())
						fmt.Printf("\t root: %v\n", cidOffer.Root().String())
						fmt.Printf("\t currency id: %v\n", cidOffer.CurrencyID())
						fmt.Printf("\t price per byte: %v\b", cidOffer.PPB().String())
						fmt.Printf("\t size: %v\n", cidOffer.Size())
						fmt.Printf("\t recipient addr: %v\n", cidOffer.ToAddr())
						fmt.Printf("\t linked miner addr: %v\n", cidOffer.LinkedMiner())
						fmt.Printf("\t expiration: %v\n", cidOffer.Expiration())
						fmt.Printf("\t with payment proxy offer:\n")
						fmt.Printf("\t\t from %v\n", payOffer.From())
						fmt.Printf("\t\t ppp %v\n", payOffer.PPP())
						fmt.Printf("\t\t period %v\n", payOffer.Period())
						fmt.Printf("\t\t expiration %v\n", payOffer.Expiration())
						err = api.RequestRetrieval(c.Context, fmt.Sprintf("localhost:%v", conf.APIPort), cidOffer, payOffer, outPathAbs)
						if err != nil {
							return err
						}
						fmt.Println("Retrieval done")
						return nil
					}
					return fmt.Errorf("Fast-retrieval not succeed for cid %v", id)
				},
			},
		},
	}
	err = app.Run(os.Args)
	if err != nil {
		log.Errorf("Error running cli: %v", err.Error())
	}
}
