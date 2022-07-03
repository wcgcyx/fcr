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
	"time"

	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/fcr/api"
	"github.com/wcgcyx/fcr/renewmgr"
	"github.com/wcgcyx/fcr/reservmgr"
	"github.com/wcgcyx/fcr/settlemgr"
)

// Payment network policy command.
var PayNetPolicyCMD = &cli.Command{
	Name:        "policy",
	Usage:       "access policy functions",
	Description: "This command outputs a list of policy functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		SettleCMD,
		RenewCMD,
		ReserveCMD,
	},
}

var SettleCMD = &cli.Command{
	Name:        "settle",
	Usage:       "access settlement policy functions",
	Description: "This command outputs a list of settlement policy functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		SettleSetCMD,
		SettleGetCMD,
		SettleRemoveCMD,
		SettleListCMD,
	},
}

var SettleSetCMD = &cli.Command{
	Name:        "set",
	Usage:       "set settlement policy",
	Description: "This command sets the settlement policy",
	ArgsUsage:   fmt.Sprintf("[currency id, peer addr (or %v), duration]", settlemgr.DefaultPolicyID),
	Action: func(c *cli.Context) error {
		currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		peerAddr := c.Args().Get(1)
		if peerAddr == "" {
			return usageError(c, fmt.Errorf("received empty peer addr"))
		}
		duration, err := time.ParseDuration(c.Args().Get(2))
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse duration: %v", err.Error()))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		if peerAddr == settlemgr.DefaultPolicyID {
			err = client.SettleMgrSetDefaultPolicy(c.Context, byte(currencyID), duration)
		} else {
			err = client.SettleMgrSetSenderPolicy(c.Context, byte(currencyID), peerAddr, duration)
		}
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var SettleGetCMD = &cli.Command{
	Name:        "get",
	Usage:       "get settlement policy",
	Description: "This command gets the settlement policy",
	ArgsUsage:   fmt.Sprintf("[currency id, peer addr (or %v)]", settlemgr.DefaultPolicyID),
	Action: func(c *cli.Context) error {
		currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		peerAddr := c.Args().Get(1)
		if peerAddr == "" {
			return usageError(c, fmt.Errorf("received empty peer addr"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		var duration time.Duration
		if peerAddr == settlemgr.DefaultPolicyID {
			duration, err = client.SettleMgrGetDefaultPolicy(c.Context, byte(currencyID))
		} else {
			duration, err = client.SettleMgrGetSenderPolicy(c.Context, byte(currencyID), peerAddr)
		}
		if err != nil {
			return err
		}
		fmt.Println(duration)
		return nil
	},
}

var SettleRemoveCMD = &cli.Command{
	Name:        "remove",
	Usage:       "remove settlement policy",
	Description: "This command removes the settlement policy",
	ArgsUsage:   fmt.Sprintf("[currency id, peer addr (or %v)]", settlemgr.DefaultPolicyID),
	Action: func(c *cli.Context) error {
		currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		peerAddr := c.Args().Get(1)
		if peerAddr == "" {
			return usageError(c, fmt.Errorf("received empty peer addr"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		if peerAddr == settlemgr.DefaultPolicyID {
			err = client.SettleMgrRemoveDefaultPolicy(c.Context, byte(currencyID))
		} else {
			err = client.SettleMgrRemoveSenderPolicy(c.Context, byte(currencyID), peerAddr)
		}
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var SettleListCMD = &cli.Command{
	Name:        "list",
	Aliases:     []string{"ls"},
	Usage:       "list all policies",
	Description: "List all existing and active policies for settlement",
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
		curChan := client.SettleMgrListCurrencyIDs(c.Context)
		for currencyID := range curChan {
			fmt.Printf("Currency %v:\n", currencyID)
			peerChan := client.SettleMgrListSenders(c.Context, currencyID)
			for peer := range peerChan {
				if c.IsSet("long") {
					var policy time.Duration
					if peer == settlemgr.DefaultPolicyID {
						policy, err = client.SettleMgrGetDefaultPolicy(c.Context, currencyID)
					} else {
						policy, err = client.SettleMgrGetSenderPolicy(c.Context, currencyID, peer)
					}
					if err != nil {
						return err
					}
					fmt.Printf("\t%v: %v\n", peer, policy)
				} else {
					fmt.Printf("\t%v\n", peer)
				}
			}
		}
		fmt.Println("Done")
		return nil
	},
}

var RenewCMD = &cli.Command{
	Name:        "renew",
	Usage:       "access renew policy functions",
	Description: "This command outputs a list of renew policy functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		RenewSetCMD,
		RenewGetCMD,
		RenewRemoveCMD,
		RenewListCMD,
	},
}

var RenewSetCMD = &cli.Command{
	Name:        "set",
	Usage:       "set renew policy",
	Description: "This command sets the renew policy",
	ArgsUsage:   fmt.Sprintf("[currency id, peer addr (or %v), duration] or [currency id, peer addr, paych addr, duration]", renewmgr.DefaultPolicyID),
	Action: func(c *cli.Context) error {
		if c.Args().Len() == 3 {
			currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
			if err != nil {
				return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
			}
			peerAddr := c.Args().Get(1)
			if peerAddr == "" {
				return usageError(c, fmt.Errorf("received empty peer addr"))
			}
			duration, err := time.ParseDuration(c.Args().Get(2))
			if err != nil {
				return usageError(c, fmt.Errorf("fail to parse duration: %v", err.Error()))
			}
			client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
			if err != nil {
				return err
			}
			defer closer()
			if peerAddr == renewmgr.DefaultPolicyID {
				err = client.RenewMgrSetDefaultPolicy(c.Context, byte(currencyID), duration)
			} else {
				err = client.RenewMgrSetSenderPolicy(c.Context, byte(currencyID), peerAddr, duration)
			}
			if err != nil {
				return err
			}
		} else if c.Args().Len() == 4 {
			currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
			if err != nil {
				return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
			}
			peerAddr := c.Args().Get(1)
			if peerAddr == "" {
				return usageError(c, fmt.Errorf("received empty peer addr"))
			}
			paychAddr := c.Args().Get(2)
			if paychAddr == "" {
				return usageError(c, fmt.Errorf("received empty paych addr"))
			}
			duration, err := time.ParseDuration(c.Args().Get(3))
			if err != nil {
				return usageError(c, fmt.Errorf("fail to parse duration: %v", err.Error()))
			}
			client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
			if err != nil {
				return err
			}
			defer closer()
			err = client.RenewMgrSetPaychPolicy(c.Context, byte(currencyID), peerAddr, paychAddr, duration)
			if err != nil {
				return err
			}
		} else {
			return usageError(c, fmt.Errorf("incorrect number of arguments, expect 3 or 4, got %v", c.Args().Len()))
		}
		fmt.Println("Succeed")
		return nil
	},
}

var RenewGetCMD = &cli.Command{
	Name:        "get",
	Usage:       "get renew policy",
	Description: "This command gets the renew policy",
	ArgsUsage:   fmt.Sprintf("[currency id, peer addr (or %v)] or [currency id, peer addr, paych addr]", renewmgr.DefaultPolicyID),
	Action: func(c *cli.Context) error {
		var duration time.Duration
		if c.Args().Len() == 2 {
			currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
			if err != nil {
				return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
			}
			peerAddr := c.Args().Get(1)
			if peerAddr == "" {
				return usageError(c, fmt.Errorf("received empty peer addr"))
			}
			client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
			if err != nil {
				return err
			}
			defer closer()
			if peerAddr == renewmgr.DefaultPolicyID {
				duration, err = client.RenewMgrGetDefaultPolicy(c.Context, byte(currencyID))
			} else {
				duration, err = client.RenewMgrGetSenderPolicy(c.Context, byte(currencyID), peerAddr)
			}
			if err != nil {
				return err
			}
		} else {
			currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
			if err != nil {
				return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
			}
			peerAddr := c.Args().Get(1)
			if peerAddr == "" {
				return usageError(c, fmt.Errorf("received empty peer addr"))
			}
			paychAddr := c.Args().Get(2)
			if paychAddr == "" {
				return usageError(c, fmt.Errorf("received empty paych addr"))
			}
			client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
			if err != nil {
				return err
			}
			defer closer()
			duration, err = client.RenewMgrGetPaychPolicy(c.Context, byte(currencyID), peerAddr, paychAddr)
			if err != nil {
				return err
			}
		}
		fmt.Println(duration)
		return nil
	},
}

var RenewRemoveCMD = &cli.Command{
	Name:        "remove",
	Usage:       "remove renew policy",
	Description: "This command removes the renew policy",
	ArgsUsage:   fmt.Sprintf("[currency id, peer addr (or %v)] or [currency id, peer addr, paych addr]", renewmgr.DefaultPolicyID),
	Action: func(c *cli.Context) error {
		if c.Args().Len() == 2 {
			currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
			if err != nil {
				return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
			}
			peerAddr := c.Args().Get(1)
			if peerAddr == "" {
				return usageError(c, fmt.Errorf("received empty peer addr"))
			}
			client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
			if err != nil {
				return err
			}
			defer closer()
			if peerAddr == renewmgr.DefaultPolicyID {
				err = client.RenewMgrRemoveDefaultPolicy(c.Context, byte(currencyID))
			} else {
				err = client.RenewMgrRemoveSenderPolicy(c.Context, byte(currencyID), peerAddr)
			}
			if err != nil {
				return err
			}
		} else {
			currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
			if err != nil {
				return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
			}
			peerAddr := c.Args().Get(1)
			if peerAddr == "" {
				return usageError(c, fmt.Errorf("received empty peer addr"))
			}
			paychAddr := c.Args().Get(2)
			if paychAddr == "" {
				return usageError(c, fmt.Errorf("received empty paych addr"))
			}
			client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
			if err != nil {
				return err
			}
			defer closer()
			err = client.RenewMgrRemovePaychPolicy(c.Context, byte(currencyID), peerAddr, paychAddr)
			if err != nil {
				return err
			}
		}
		fmt.Println("Succeed")
		return nil
	},
}

var RenewListCMD = &cli.Command{
	Name:        "list",
	Aliases:     []string{"ls"},
	Usage:       "list all policies",
	Description: "List all existing and active policies for renew",
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
		curChan := client.RenewMgrListCurrencyIDs(c.Context)
		for currencyID := range curChan {
			fmt.Printf("Currency %v:\n", currencyID)
			peerChan := client.RenewMgrListSenders(c.Context, currencyID)
			for peer := range peerChan {
				if peer == renewmgr.DefaultPolicyID {
					if c.IsSet("long") {
						policy, err := client.RenewMgrGetDefaultPolicy(c.Context, currencyID)
						if err != nil {
							return err
						}
						fmt.Printf("\t%v: %v\n", peer, policy)
					} else {
						fmt.Printf("\t%v\n", peer)
					}
				} else {
					fmt.Printf("\tPeer %v:\n", peer)
					paychChan := client.RenewMgrListPaychs(c.Context, currencyID, peer)
					for paych := range paychChan {
						if c.IsSet("long") {
							var policy time.Duration
							if paych == renewmgr.DefaultPolicyID {
								policy, err = client.RenewMgrGetSenderPolicy(c.Context, currencyID, peer)
								if err != nil {
									return err
								}
							} else {
								policy, err = client.RenewMgrGetPaychPolicy(c.Context, currencyID, peer, paych)
								if err != nil {
									return err
								}
							}
							fmt.Printf("\t\t%v: %v\n", paych, policy)
						} else {
							fmt.Printf("\t\t%v\n", paych)
						}
					}
				}
			}
		}
		fmt.Println("Done")
		return nil
	},
}

var ReserveCMD = &cli.Command{
	Name:        "reserve",
	Usage:       "access reservation policy functions",
	Description: "This command outputs a list of reservation policy functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		ReserveSetCMD,
		ReserveGetCMD,
		ReserveRemoveCMD,
		ReserveListCMD,
	},
}

var ReserveSetCMD = &cli.Command{
	Name:        "set",
	Usage:       "set reservation policy",
	Description: "This command sets the reservation policy",
	ArgsUsage:   fmt.Sprintf("[currency id, paych addr (or %v), max (-1 if unlimited)] or [currency id, paych addr, peer addr, max (-1 if unlimited)]", reservmgr.DefaultPolicyID),
	Action: func(c *cli.Context) error {
		if c.Args().Len() == 3 {
			currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
			if err != nil {
				return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
			}
			paychAddr := c.Args().Get(1)
			if paychAddr == "" {
				return usageError(c, fmt.Errorf("received empty paych addr"))
			}
			max, ok := big.NewInt(0).SetString(c.Args().Get(2), 10)
			if !ok {
				return usageError(c, fmt.Errorf("fail to parse max %v", c.Args().Get(2)))
			}
			client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
			if err != nil {
				return err
			}
			defer closer()
			if paychAddr == reservmgr.DefaultPolicyID {
				if max.Cmp(big.NewInt(0)) < 0 {
					err = client.ReservMgrSetDefaultPolicy(c.Context, byte(currencyID), true, nil)
				} else {
					err = client.ReservMgrSetDefaultPolicy(c.Context, byte(currencyID), false, max)
				}
			} else {
				if max.Cmp(big.NewInt(0)) < 0 {
					err = client.ReservMgrSetPaychPolicy(c.Context, byte(currencyID), paychAddr, true, nil)
				} else {
					err = client.ReservMgrSetPaychPolicy(c.Context, byte(currencyID), paychAddr, false, max)
				}
			}
			if err != nil {
				return err
			}
		} else if c.Args().Len() == 4 {
			currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
			if err != nil {
				return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
			}
			paychAddr := c.Args().Get(1)
			if paychAddr == "" {
				return usageError(c, fmt.Errorf("received empty paych addr"))
			}
			peerAddr := c.Args().Get(2)
			if peerAddr == "" {
				return usageError(c, fmt.Errorf("received empty peer addr"))
			}
			max, ok := big.NewInt(0).SetString(c.Args().Get(3), 10)
			if !ok {
				return usageError(c, fmt.Errorf("fail to parse max %v", c.Args().Get(2)))
			}
			client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
			if err != nil {
				return err
			}
			defer closer()
			if max.Cmp(big.NewInt(0)) < 0 {
				err = client.ReservMgrSetPeerPolicy(c.Context, byte(currencyID), paychAddr, peerAddr, true, nil)
			} else {
				err = client.ReservMgrSetPeerPolicy(c.Context, byte(currencyID), paychAddr, peerAddr, false, max)
			}
			if err != nil {
				return err
			}
		} else {
			return usageError(c, fmt.Errorf("incorrect number of arguments, expect 3 or 4, got %v", c.Args().Len()))
		}
		fmt.Println("Succeed")
		return nil
	},
}

var ReserveGetCMD = &cli.Command{
	Name:        "get",
	Usage:       "get reservation policy",
	Description: "This command gets the reservation policy",
	ArgsUsage:   fmt.Sprintf("[currency id, paych addr (or %v)] or [currency id, paych addr, peer addr]", reservmgr.DefaultPolicyID),
	Action: func(c *cli.Context) error {
		var res api.ReservMgrPolicyRes
		if c.Args().Len() == 2 {
			currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
			if err != nil {
				return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
			}
			paychAddr := c.Args().Get(1)
			if paychAddr == "" {
				return usageError(c, fmt.Errorf("received empty paych addr"))
			}
			client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
			if err != nil {
				return err
			}
			defer closer()
			if paychAddr == reservmgr.DefaultPolicyID {
				res, err = client.ReservMgrGetDefaultPolicy(c.Context, byte(currencyID))
			} else {
				res, err = client.ReservMgrGetPaychPolicy(c.Context, byte(currencyID), paychAddr)
			}
			if err != nil {
				return nil
			}
		} else {
			currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
			if err != nil {
				return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
			}
			paychAddr := c.Args().Get(1)
			if paychAddr == "" {
				return usageError(c, fmt.Errorf("received empty paych addr"))
			}
			peerAddr := c.Args().Get(2)
			if peerAddr == "" {
				return usageError(c, fmt.Errorf("received empty peer addr"))
			}
			client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
			if err != nil {
				return err
			}
			defer closer()
			res, err = client.ReservMgrGetPeerPolicy(c.Context, byte(currencyID), paychAddr, peerAddr)
			if err != nil {
				return err
			}
		}
		if res.Unlimited {
			fmt.Println("Unlimited")
		} else {
			fmt.Println(res.Max)
		}
		return nil
	},
}

var ReserveRemoveCMD = &cli.Command{
	Name:        "remove",
	Usage:       "remove reservation policy",
	Description: "This command removes the reservation policy",
	ArgsUsage:   fmt.Sprintf("[currency id, paych addr (or %v)] or [currency id, paych addr, peer addr]", reservmgr.DefaultPolicyID),
	Action: func(c *cli.Context) error {
		if c.Args().Len() == 2 {
			currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
			if err != nil {
				return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
			}
			paychAddr := c.Args().Get(1)
			if paychAddr == "" {
				return usageError(c, fmt.Errorf("received empty paych addr"))
			}
			client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
			if err != nil {
				return err
			}
			defer closer()
			if paychAddr == reservmgr.DefaultPolicyID {
				err = client.ReservMgrRemoveDefaultPolicy(c.Context, byte(currencyID))
			} else {
				err = client.ReservMgrRemovePaychPolicy(c.Context, byte(currencyID), paychAddr)
			}
			if err != nil {
				return nil
			}
		} else {
			currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
			if err != nil {
				return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
			}
			paychAddr := c.Args().Get(1)
			if paychAddr == "" {
				return usageError(c, fmt.Errorf("received empty paych addr"))
			}
			peerAddr := c.Args().Get(2)
			if peerAddr == "" {
				return usageError(c, fmt.Errorf("received empty peer addr"))
			}
			client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
			if err != nil {
				return err
			}
			defer closer()
			err = client.ReservMgrRemovePeerPolicy(c.Context, byte(currencyID), paychAddr, peerAddr)
			if err != nil {
				return err
			}
		}
		fmt.Println("Succeed")
		return nil
	},
}

var ReserveListCMD = &cli.Command{
	Name:        "list",
	Aliases:     []string{"ls"},
	Usage:       "list all policies",
	Description: "List all existing and active policies for reserve",
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
		curChan := client.ReservMgrListCurrencyIDs(c.Context)
		for currencyID := range curChan {
			fmt.Printf("Currency %v:\n", currencyID)
			paychChan := client.ReservMgrListPaychs(c.Context, currencyID)
			for paych := range paychChan {
				if paych == reservmgr.DefaultPolicyID {
					if c.IsSet("long") {
						policy, err := client.ReservMgrGetDefaultPolicy(c.Context, currencyID)
						if err != nil {
							return err
						}
						if policy.Unlimited {
							fmt.Printf("\t%v: Unlimited\n", paych)
						} else {
							fmt.Printf("\t%v: %v\n", paych, policy.Max)
						}
					}
				} else {
					fmt.Printf("\tPaych: %v:\n", paych)
					peerChan := client.ReservMgrListPeers(c.Context, currencyID, paych)
					for peer := range peerChan {
						if c.IsSet("long") {
							var policy api.ReservMgrPolicyRes
							if peer == reservmgr.DefaultPolicyID {
								policy, err = client.ReservMgrGetPaychPolicy(c.Context, currencyID, paych)
								if err != nil {
									return err
								}
							} else {
								policy, err = client.ReservMgrGetPeerPolicy(c.Context, currencyID, paych, peer)
								if err != nil {
									return err
								}
							}
							if policy.Unlimited {
								fmt.Printf("\t\t%v: Unlimited\n", peer)
							} else {
								fmt.Printf("\t\t%v: %v\n", peer, policy.Max)
							}
						} else {
							fmt.Printf("\t\t%v\n", peer)
						}
					}
				}
			}
		}
		fmt.Println("Done")
		return nil
	},
}
