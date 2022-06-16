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
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/fcr/api"
)

// CID network command.
var CIDNetCMD = &cli.Command{
	Name:        "cidnet",
	Usage:       "access cid network functions",
	Description: "This command outputs a list of cid network functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		CIDNetSearchOfferCMD,
		CIDNetQueryOfferCMD,
		PieceCMD,
		CServingCMD,
	},
}

var CIDNetSearchOfferCMD = &cli.Command{
	Name:        "search",
	Usage:       "search piece offer",
	Description: "Search the cid network for a piece offer with given currency",
	ArgsUsage:   "[cid, currency id, max]",
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
		max, err := strconv.ParseUint(c.Args().Get(2), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse max: %v", err.Error()))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		offerChan := client.COfferProtoFindOffersAsync(c.Context, byte(currencyID), id, int(max))
		i := 0
		for offer := range offerChan {
			fmt.Printf("Piece offer %v:\n", i)
			offerData, err := offer.Encode()
			if err != nil {
				fmt.Printf("\tFail to encode: %v\n", err.Error())
				continue
			}
			fmt.Printf("\tsize %v\n", offer.Size)
			fmt.Printf("\tprice per byte %v\n", offer.PPB)
			fmt.Printf("\tsender %v\n", offer.RecipientAddr)
			fmt.Printf("\tlinked miner %v (Type %v)\n", offer.LinkedMinerAddr, offer.LinkedMinerKeyType)
			fmt.Printf("\texpiration %v\n", offer.Expiration)
			fmt.Printf("\tinactivity %v\n", offer.Inactivity)
			fmt.Printf("\toffer data: %v\n", base64.StdEncoding.EncodeToString(offerData))
			i++
		}
		fmt.Println("Done")
		return nil
	},
}

var CIDNetQueryOfferCMD = &cli.Command{
	Name:        "query",
	Usage:       "query piece offer from given peer",
	Description: "Query a given peer for a piece offer with given currency",
	ArgsUsage:   "[cid, currency id, peer addr]",
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
		peerAddr := c.Args().Get(2)
		if peerAddr == "" {
			return usageError(c, fmt.Errorf("received empty peer addr"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		offer, err := client.COfferProtoQueryOffer(c.Context, byte(currencyID), peerAddr, id)
		if err != nil {
			return err
		}
		fmt.Printf("Piece offer:\n")
		offerData, err := offer.Encode()
		if err != nil {
			fmt.Printf("\tFail to encode: %v\n", err.Error())
			return err
		}
		fmt.Printf("\tsize %v\n", offer.Size)
		fmt.Printf("\tprice per byte %v\n", offer.PPB)
		fmt.Printf("\tsender %v\n", offer.RecipientAddr)
		fmt.Printf("\tlinked miner %v (Type %v)\n", offer.LinkedMinerAddr, offer.LinkedMinerKeyType)
		fmt.Printf("\texpiration %v\n", offer.Expiration)
		fmt.Printf("\tinactivity %v\n", offer.Inactivity)
		fmt.Printf("\toffer data: %v\n", base64.StdEncoding.EncodeToString(offerData))
		return nil
	},
}
