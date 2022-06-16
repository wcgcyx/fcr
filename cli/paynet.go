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
	"context"
	"encoding/base64"
	"fmt"
	"math/big"

	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/fcr/api"
	"github.com/wcgcyx/fcr/fcroffer"
)

// Payment network command.
var PayNetCMD = &cli.Command{
	Name:        "paynet",
	Usage:       "access payment network functions",
	Description: "This command outputs a list of payment network functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		PayNetQueryOfferCMD,
		PayNetReserveCMD,
		PaychCMD,
		PservingCMD,
		PayNetPolicyCMD,
	},
}

var PayNetReserveCMD = &cli.Command{
	Name:        "reserve",
	Usage:       "reserve payment for given offer",
	Description: "Reserve channel balance for given piece offer or pay offer",
	ArgsUsage:   "[piece offer/pay offer]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "payoffer",
			Aliases: []string{"po"},
			Usage:   "indicate whether pay offer is supplied",
		},
	},
	Action: func(c *cli.Context) error {
		offerData, err := base64.StdEncoding.DecodeString(c.Args().Get(0))
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse offer data"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		if c.IsSet("payoffer") {
			offer := fcroffer.PayOffer{}
			err = offer.Decode(offerData)
			if err != nil {
				return err
			}
			res, err := client.PayMgrReserveForSelfWithOffer(c.Context, offer)
			if err != nil {
				return err
			}
			fmt.Printf("Reservation: %v-%v\n", res.ResCh, res.ResID)
		} else {
			offer := fcroffer.PieceOffer{}
			err = offer.Decode(offerData)
			if err != nil {
				return err
			}
			required := big.NewInt(0).Mul(offer.PPB, big.NewInt(int64(offer.Size)))
			res, err := client.PayMgrReserveForSelf(c.Context, offer.CurrencyID, offer.RecipientAddr, required, offer.Expiration, offer.Inactivity)
			if err != nil {
				return err
			}
			fmt.Printf("Reservation: %v-%v\n", res.ResCh, res.ResID)
		}
		return nil
	},
}

var PayNetQueryOfferCMD = &cli.Command{
	Name:        "query",
	Usage:       "query pay offer for given piece offer",
	Description: "Query network for a pay offer based on given piece offer",
	ArgsUsage:   "[piece offer]",
	Action: func(c *cli.Context) error {
		offerData, err := base64.StdEncoding.DecodeString(c.Args().Get(0))
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse offer data"))
		}
		offer := fcroffer.PieceOffer{}
		err = offer.Decode(offerData)
		if err != nil {
			return err
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		// Check if active out contains given peer
		subCtx, cancel := context.WithCancel(c.Context)
		defer cancel()
		paychChan := client.ActiveOutListPaychsByPeer(subCtx, offer.CurrencyID, offer.RecipientAddr)
		exists := false
		for range paychChan {
			exists = true
			cancel()
		}
		if exists {
			return fmt.Errorf("exists active out channel to %v with currency %v, no need to query offer", offer.RecipientAddr, offer.CurrencyID)
		}
		required := big.NewInt(0).Mul(offer.PPB, big.NewInt(int64(offer.Size)))
		routeChan := client.RouteStoreListRoutesTo(c.Context, offer.CurrencyID, offer.RecipientAddr)
		i := 0
		for route := range routeChan {
			payOffer, err := client.POfferProtoQueryOffer(c.Context, offer.CurrencyID, route, required)
			if err != nil {
				fmt.Printf("Fail to query for route %v: %v\n", route, err.Error())
				continue
			}
			fmt.Printf("Pay offer %v:\n", i)
			payOfferData, err := payOffer.Encode()
			if err != nil {
				fmt.Printf("\tFail to encode: %v\n", err.Error())
				continue
			}
			fmt.Printf("\tamount %v\n", payOffer.Amt)
			fmt.Printf("\tprice per period %v\n", payOffer.PPP)
			fmt.Printf("\tperiod %v\n", payOffer.Period)
			fmt.Printf("\texpiration %v\n", payOffer.Expiration)
			fmt.Printf("\tinactivity %v\n", payOffer.Inactivity)
			fmt.Printf("\toffer data %v\n", base64.StdEncoding.EncodeToString(payOfferData))
			i++
		}
		return nil
	},
}
