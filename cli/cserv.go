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

	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/fcr/api"
)

// CID serving command.
var CServingCMD = &cli.Command{
	Name:        "serving",
	Usage:       "Access serving functions",
	Description: "This command outputs a list of piece serving functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		CServingServeCMD,
		CServingListCMD,
		CServingInspectCMD,
		CServingStopCMD,
	},
}

var CServingServeCMD = &cli.Command{
	Name:        "serve",
	Usage:       "serve piece to network",
	Description: "Start serving a piece with given price",
	ArgsUsage:   "[cid, currency id, ppb]",
	Action: func(c *cli.Context) error {
		// Parse arguments.
		id, err := cid.Parse(c.Args().Get(0))
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse cid: %v", err.Error()))
		}
		currencyID, err := strconv.ParseUint(c.Args().Get(1), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		ppb, ok := big.NewInt(0).SetString(c.Args().Get(2), 10)
		if !ok {
			return usageError(c, fmt.Errorf("fail to parse ppb"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.CServMgrServe(c.Context, id, byte(currencyID), ppb)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var CServingListCMD = &cli.Command{
	Name:        "list",
	Aliases:     []string{"ls"},
	Usage:       "list served pieces",
	Description: "List pieces that are currently being served",
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
		idChan := client.CServMgrListPieceIDs(c.Context)
		for id := range idChan {
			curChan := client.CServMgrListCurrencyIDs(c.Context, id)
			for currencyID := range curChan {
				if c.IsSet("long") {
					res, err := client.CServMgrInspect(c.Context, id, byte(currencyID))
					if err != nil {
						return err
					}
					if res.Served {
						fmt.Printf("Piece %v (Currency %v): PPB %v\n", id.String(), currencyID, res.PPB)
					}
				} else {
					fmt.Printf("Piece %v (Currency %v)\n", id.String(), currencyID)
				}
			}
		}
		fmt.Println("Done")
		return nil
	},
}

var CServingInspectCMD = &cli.Command{
	Name:        "inspect",
	Usage:       "inspect a served piece",
	Description: "Inspect the details of a given piece",
	ArgsUsage:   "[cid, currency id]",
	Action: func(c *cli.Context) error {
		// Parse arguments.
		id, err := cid.Parse(c.Args().Get(0))
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse cid: %v", err.Error()))
		}
		currencyID, err := strconv.ParseUint(c.Args().Get(1), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		res, err := client.CServMgrInspect(c.Context, id, byte(currencyID))
		if err != nil {
			return err
		}
		if res.Served {
			fmt.Printf("Piece %v (Currency %v):\n", id.String(), currencyID)
			fmt.Printf("\tPrice per byte: %v\n", res.PPB)
		} else {
			fmt.Printf("Piece %v does not exist\n", id.String())
		}
		return nil
	},
}

var CServingStopCMD = &cli.Command{
	Name:        "stop",
	Usage:       "stop a served piece",
	Description: "Stop serving a given piece and currency id pair",
	ArgsUsage:   "[cid, currency id]",
	Action: func(c *cli.Context) error {
		// Parse arguments.
		id, err := cid.Parse(c.Args().Get(0))
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse cid: %v", err.Error()))
		}
		currencyID, err := strconv.ParseUint(c.Args().Get(1), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.CServMgrStop(c.Context, id, byte(currencyID))
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}
