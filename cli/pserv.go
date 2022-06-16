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

	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/fcr/api"
)

// Paych serving command.
var PservingCMD = &cli.Command{
	Name:        "serving",
	Usage:       "Access serving functions",
	Description: "This command outputs a list of paych serving functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		PServingServeCMD,
		PServingListCMD,
		PServingInspectCMD,
		PservingStopCMD,
	},
}

var PServingServeCMD = &cli.Command{
	Name:        "serve",
	Usage:       "serve paych to network",
	Description: "Start serving a paych with given price",
	ArgsUsage:   "[currency id, peer addr, paych addr, ppp, period]",
	Action: func(c *cli.Context) error {
		currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		peerAddr := c.Args().Get(1)
		if peerAddr == "" {
			return usageError(c, fmt.Errorf("received empty peer addr"))
		}
		chAddr := c.Args().Get(2)
		if chAddr == "" {
			return usageError(c, fmt.Errorf("received empty paych addr"))
		}
		ppp, ok := big.NewInt(0).SetString(c.Args().Get(3), 10)
		if !ok {
			return usageError(c, fmt.Errorf("fail to parse ppp"))
		}
		period, ok := big.NewInt(0).SetString(c.Args().Get(4), 10)
		if !ok {
			return usageError(c, fmt.Errorf("fail to parse period"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.PservMgrServe(c.Context, byte(currencyID), peerAddr, chAddr, ppp, period)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var PServingListCMD = &cli.Command{
	Name:        "list",
	Aliases:     []string{"ls"},
	Usage:       "list served paychs",
	Description: "List paychs that are currently being served",
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
		curChan := client.PservMgrListCurrencyIDs(c.Context)
		for currencyID := range curChan {
			peerChan := client.PservMgrListRecipients(c.Context, currencyID)
			for peer := range peerChan {
				paychChan := client.PservMgrListServings(c.Context, currencyID, peer)
				for paych := range paychChan {
					if c.IsSet("long") {
						res, err := client.PservMgrInspect(c.Context, currencyID, peer, paych)
						if err != nil {
							return err
						}
						if res.Served {
							fmt.Printf("Paych %v (to %v, currency %v): PPP %v, Period %v\n", paych, peer, currencyID, res.PPP, res.Period)
						}
					} else {
						fmt.Printf("Paych %v (to %v, currency %v)\n", paych, peer, currencyID)
					}
				}
			}
		}
		fmt.Println("Done")
		return nil
	},
}

var PServingInspectCMD = &cli.Command{
	Name:        "inspect",
	Usage:       "inspect a served paych",
	Description: "Inspect the details of a given paych",
	ArgsUsage:   "[currency id, peer addr, paych addr]",
	Action: func(c *cli.Context) error {
		currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		peerAddr := c.Args().Get(1)
		if peerAddr == "" {
			return usageError(c, fmt.Errorf("received empty peer addr"))
		}
		chAddr := c.Args().Get(2)
		if chAddr == "" {
			return usageError(c, fmt.Errorf("received empty paych addr"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		res, err := client.PservMgrInspect(c.Context, byte(currencyID), peerAddr, chAddr)
		if err != nil {
			return err
		}
		if res.Served {
			fmt.Printf("Paych %v (to %v, currency %v):\n", chAddr, peerAddr, currencyID)
			fmt.Printf("\tPPP %v\n", res.PPP)
			fmt.Printf("\tPeriod %v\n", res.Period)
		} else {
			fmt.Printf("Paych %v-%v-%v does not exist\n", currencyID, peerAddr, chAddr)
		}
		return nil
	},
}

var PservingStopCMD = &cli.Command{
	Name:        "stop",
	Usage:       "stop a served paych",
	Description: "Stop serving a given paych",
	ArgsUsage:   "[currency id, peer addr, paych addr]",
	Action: func(c *cli.Context) error {
		currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		peerAddr := c.Args().Get(1)
		if peerAddr == "" {
			return usageError(c, fmt.Errorf("received empty peer addr"))
		}
		chAddr := c.Args().Get(2)
		if chAddr == "" {
			return usageError(c, fmt.Errorf("received empty paych addr"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.PservMgrStop(c.Context, byte(currencyID), peerAddr, chAddr)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}
