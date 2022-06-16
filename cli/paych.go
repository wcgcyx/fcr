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
	"math/big"
	"strconv"

	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/fcr/api"
	"github.com/wcgcyx/fcr/fcroffer"
	"github.com/wcgcyx/fcr/paychstate"
)

// Paych command.
var PaychCMD = &cli.Command{
	Name:        "paych",
	Usage:       "access paych functions",
	Description: "This command outputs a list of paych functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		PaychQueryOfferCMD,
		PaychCreateCMD,
		PaychRenewCMD,
		PaychTopupCMD,
		PaychUpdateCMD,
		PaychSettleCMD,
		PaychCollectCMD,
		PaychActiveOutCMD,
		PaychActiveInCMD,
		PaychInactiveOutCMD,
		PaychInactiveInCMD,
	},
}

var PaychQueryOfferCMD = &cli.Command{
	Name:        "query",
	Usage:       "query for paych offer",
	Description: "Query given peer the minimum settlement time for a new payment channel",
	ArgsUsage:   "[currency id, peer addr]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "cidoffer",
			Aliases: []string{"co"},
			Usage:   "override arguments and query from cidoffer",
		},
	},
	Action: func(c *cli.Context) error {
		var currencyID byte
		var peerAddr string
		if c.IsSet("cidoffer") {
			offerData, err := base64.StdEncoding.DecodeString(c.String("cidoffer"))
			if err != nil {
				return err
			}
			offer := fcroffer.PieceOffer{}
			err = offer.Decode(offerData)
			if err != nil {
				return err
			}
			currencyID = offer.CurrencyID
			peerAddr = offer.RecipientAddr
		} else {
			currency, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
			if err != nil {
				return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
			}
			currencyID = byte(currency)
			peerAddr = c.Args().Get(1)
			if peerAddr == "" {
				return usageError(c, fmt.Errorf("received empty peer addr"))
			}
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		paychOffer, err := client.PaychProtoQueryAdd(c.Context, currencyID, peerAddr)
		if err != nil {
			return err
		}
		fmt.Printf("Paych offer:\n")
		offerData, err := paychOffer.Encode()
		if err != nil {
			fmt.Printf("\tFail to encode: %v\n", err.Error())
			return err
		}
		fmt.Printf("\tcurrency %v\n", paychOffer.CurrencyID)
		fmt.Printf("\trecipient %v\n", paychOffer.ToAddr)
		fmt.Printf("\tsettlement %v\n", paychOffer.Settlement)
		fmt.Printf("\texpiration %v\n", paychOffer.Expiration)
		fmt.Printf("\toffer data: %v\n", base64.StdEncoding.EncodeToString(offerData))
		return nil
	},
}

var PaychCreateCMD = &cli.Command{
	Name:        "create",
	Usage:       "create a payment channel",
	Description: "create a payment channel for given paych offer",
	ArgsUsage:   "[paych offer, amt]",
	Action: func(c *cli.Context) error {
		offerData, err := base64.StdEncoding.DecodeString(c.Args().Get(0))
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse offer data"))
		}
		amt, ok := big.NewInt(0).SetString(c.Args().Get(1), 10)
		if !ok {
			return usageError(c, fmt.Errorf("fail to parse amount"))
		}
		if amt.Cmp(big.NewInt(0)) <= 0 {
			return fmt.Errorf("expect positive amount, got %v", amt)
		}
		offer := fcroffer.PaychOffer{}
		err = offer.Decode(offerData)
		if err != nil {
			return err
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		// Create channel
		fmt.Println("Create payment channel...")
		chAddr, err := client.TransactorCreate(c.Context, offer.CurrencyID, offer.ToAddr, amt)
		if err != nil {
			return err
		}
		fmt.Println(chAddr)
		fmt.Println("Add payment channel...")
		err = client.PaychProtoAdd(c.Context, chAddr, offer)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var PaychRenewCMD = &cli.Command{
	Name:        "renew",
	Usage:       "renew an active out paych",
	Description: "Attempt to renew a outbound payment channel that is currently active",
	ArgsUsage:   "[currency id, peer addr, paych addr]",
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
		chAddr := c.Args().Get(2)
		if chAddr == "" {
			return usageError(c, fmt.Errorf("received empty paych addr"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		_, err = client.ActiveOutRead(c.Context, currencyID, peerAddr, chAddr)
		if err != nil {
			return err
		}
		err = client.PaychProtoRenew(c.Context, currencyID, peerAddr, chAddr)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var PaychTopupCMD = &cli.Command{
	Name:        "topup",
	Usage:       "topup an active out paych",
	Description: "Topup a outbound payment channel that is currently active",
	ArgsUsage:   "[currency id, peer addr, paych addr, amt]",
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
		chAddr := c.Args().Get(2)
		if chAddr == "" {
			return usageError(c, fmt.Errorf("received empty paych addr"))
		}
		amt, ok := big.NewInt(0).SetString(c.Args().Get(3), 10)
		if !ok {
			return usageError(c, fmt.Errorf("fail to parse amount"))
		}
		if amt.Cmp(big.NewInt(0)) <= 0 {
			return fmt.Errorf("expect positive amount, got %v", amt)
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		_, err = client.ActiveOutRead(c.Context, currencyID, peerAddr, chAddr)
		if err != nil {
			return err
		}
		fmt.Println("Topup payment channel...")
		err = client.TransactorTopup(c.Context, currencyID, chAddr, amt)
		if err != nil {
			return err
		}
		fmt.Println("Updating payment manager channel balance...")
		err = client.PayMgrUpdateOutboundChannelBalance(c.Context, currencyID, peerAddr, chAddr)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var PaychUpdateCMD = &cli.Command{
	Name:        "update",
	Usage:       "update an active out paych",
	Description: "Update a outbound payment channel",
	ArgsUsage:   "[currency id, peer addr, paych addr]",
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
		chAddr := c.Args().Get(2)
		if chAddr == "" {
			return usageError(c, fmt.Errorf("received empty paych addr"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		var state paychstate.State
		// Try to load state
		state, err = client.ActiveOutRead(c.Context, currencyID, peerAddr, chAddr)
		if err != nil {
			state, err = client.ActiveInRead(c.Context, currencyID, peerAddr, chAddr)
			if err != nil {
				state, err = client.InactiveOutRead(c.Context, currencyID, peerAddr, chAddr)
				if err != nil {
					state, err = client.InactiveInRead(c.Context, currencyID, peerAddr, chAddr)
					if err != nil {
						return err
					}
				}
			}
		}
		if state.Voucher == "" {
			return fmt.Errorf("voucher stored is empty, cannot update")
		}
		fmt.Println("Update payment channel...")
		err = client.TransactorUpdate(c.Context, currencyID, chAddr, state.Voucher)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var PaychSettleCMD = &cli.Command{
	Name:        "settle",
	Usage:       "settle an inactive paych",
	Description: "Settle an inactive payment channel",
	ArgsUsage:   "[currency id, peer addr, paych addr]",
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
		chAddr := c.Args().Get(2)
		if chAddr == "" {
			return usageError(c, fmt.Errorf("received empty paych addr"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		// Try to load state
		_, err = client.InactiveOutRead(c.Context, currencyID, peerAddr, chAddr)
		if err != nil {
			_, err = client.InactiveInRead(c.Context, currencyID, peerAddr, chAddr)
			if err != nil {
				return err
			}
		}
		fmt.Println("Settle payment channel...")
		err = client.TransactorSettle(c.Context, currencyID, chAddr)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var PaychCollectCMD = &cli.Command{
	Name:        "collect",
	Usage:       "collect a settled paych",
	Description: "Collect a settled payment channel",
	ArgsUsage:   "[currency id, peer addr, paych addr]",
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
		chAddr := c.Args().Get(2)
		if chAddr == "" {
			return usageError(c, fmt.Errorf("received empty paych addr"))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		// Try to load state
		inbound := false
		_, err = client.InactiveOutRead(c.Context, currencyID, peerAddr, chAddr)
		if err != nil {
			_, err = client.InactiveInRead(c.Context, currencyID, peerAddr, chAddr)
			if err != nil {
				return err
			}
			inbound = true
		}
		fmt.Println("Collect payment channel...")
		err = client.TransactorCollect(c.Context, currencyID, chAddr)
		if err != nil {
			return err
		}
		if inbound {
			client.InactiveInRemove(currencyID, peerAddr, chAddr)
		} else {
			client.InactiveOutRemove(currencyID, peerAddr, chAddr)
		}
		fmt.Println("Succeed")
		return nil
	},
}

var PaychActiveOutCMD = &cli.Command{
	Name:        "active-out",
	Usage:       "access active out paych functions",
	Description: "This command outputs a list of active out paych functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		PaychActiveOutGetCMD,
		PaychActiveOutListCMD,
		PaychActiveOutBearNetworkLossCMD,
	},
}

var PaychActiveOutListCMD = &cli.Command{
	Name:        "list",
	Aliases:     []string{"ls"},
	Usage:       "list all paychs",
	Description: "List all payment channels stored",
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
		curChan := client.ActiveOutListCurrencyIDs(c.Context)
		for currencyID := range curChan {
			fmt.Printf("Currency %v\n", currencyID)
			peerChan := client.ActiveOutListPeers(c.Context, currencyID)
			for peer := range peerChan {
				fmt.Printf("\tPeer %v\n", peer)
				paychChan := client.ActiveOutListPaychsByPeer(c.Context, currencyID, peer)
				for paych := range paychChan {
					if c.IsSet("long") {
						state, err := client.ActiveOutRead(c.Context, currencyID, peer, paych)
						if err != nil {
							return err
						}
						res, err := client.PaychMonitorCheck(c.Context, true, currencyID, state.ChAddr)
						if err != nil {
							return err
						}
						// Calculate network loss amount
						loss := big.NewInt(0)
						if state.NetworkLossVoucher != "" {
							res, err := client.TransactorVerifyVoucher(currencyID, state.NetworkLossVoucher)
							if err != nil {
								return err
							}
							loss = big.NewInt(0).Sub(res.Redeemed, state.Redeemed)
						}
						fmt.Printf("\t\t%v:\n", paych)
						fmt.Printf("\t\t\tBalance %v, Redeemed %v, Nonce %v, Voucher %v, Network loss voucher %v (%v), Settlement %v, Last updated %v\n", state.Balance, state.Redeemed, state.Nonce, state.Voucher, state.NetworkLossVoucher, loss, res.Settlement, res.Updated)
					} else {
						fmt.Printf("\t\t%v\n", paych)
					}
				}
			}
		}
		fmt.Println("Done")
		return nil
	},
}

var PaychActiveOutGetCMD = &cli.Command{
	Name:        "get",
	Usage:       "get paych state",
	Description: "Get the state of a given paych",
	ArgsUsage:   "[currency id, peer addr, paych addr]",
	Action: func(c *cli.Context) error {
		// Parse arguments.
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
		state, err := client.ActiveOutRead(c.Context, byte(currencyID), peerAddr, paychAddr)
		if err != nil {
			return err
		}
		res, err := client.PaychMonitorCheck(c.Context, true, state.CurrencyID, state.ChAddr)
		if err != nil {
			return err
		}
		loss := big.NewInt(0)
		if state.NetworkLossVoucher != "" {
			res, err := client.TransactorVerifyVoucher(state.CurrencyID, state.NetworkLossVoucher)
			if err != nil {
				return err
			}
			loss = big.NewInt(0).Sub(res.Redeemed, state.Redeemed)
		}
		fmt.Printf("Channel: %v (Currency %v):\n", paychAddr, currencyID)
		fmt.Printf("\tBalance: %v\n", state.Balance)
		fmt.Printf("\tRedeemed: %v\n", state.Redeemed)
		fmt.Printf("\tNonce: %v\n", state.Nonce)
		fmt.Printf("\tVoucher: %v\n", state.Voucher)
		fmt.Printf("\tNetwork loss voucher: %v (%v)\n", state.NetworkLossVoucher, loss)
		fmt.Printf("\tSettlement: %v\n", res.Settlement)
		fmt.Printf("\tLast updated: %v\n", res.Updated)
		return nil
	},
}

var PaychActiveOutBearNetworkLossCMD = &cli.Command{
	Name:        "bear",
	Usage:       "bear the network loss",
	Description: "Bear network loss and continue the use of active out paych",
	ArgsUsage:   "[currency id, peer addr, paych addr]",
	Action: func(c *cli.Context) error {
		// Parse arguments.
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
		// Check first
		state, err := client.ActiveOutRead(c.Context, byte(currencyID), peerAddr, paychAddr)
		if err != nil {
			return err
		}
		if state.NetworkLossVoucher == "" {
			return fmt.Errorf("No network loss found")
		}
		err = client.PayMgrBearNetworkLoss(c.Context, state.CurrencyID, peerAddr, paychAddr)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var PaychInactiveOutCMD = &cli.Command{
	Name:        "inactive-out",
	Usage:       "access inactive out paych functions",
	Description: "This command outputs a list of inactive out paych functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		PaychInactiveOutGetCMD,
		PaychInactiveOutListCMD,
	},
}

var PaychInactiveOutListCMD = &cli.Command{
	Name:        "list",
	Aliases:     []string{"ls"},
	Usage:       "list all paychs",
	Description: "List all payment channels stored",
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
		curChan := client.InactiveOutListCurrencyIDs(c.Context)
		for currencyID := range curChan {
			fmt.Printf("Currency %v\n", currencyID)
			peerChan := client.InactiveOutListPeers(c.Context, currencyID)
			for peer := range peerChan {
				fmt.Printf("\tPeer %v\n", peer)
				paychChan := client.InactiveOutListPaychsByPeer(c.Context, currencyID, peer)
				for paych := range paychChan {
					if c.IsSet("long") {
						state, err := client.InactiveOutRead(c.Context, currencyID, peer, paych)
						if err != nil {
							return err
						}
						res, err := client.TransactorCheck(c.Context, currencyID, paych)
						if err != nil {
							return err
						}
						loss := big.NewInt(0)
						if state.NetworkLossVoucher != "" {
							res, err := client.TransactorVerifyVoucher(currencyID, state.NetworkLossVoucher)
							if err != nil {
								return err
							}
							loss = big.NewInt(0).Sub(res.Redeemed, state.Redeemed)
						}
						fmt.Printf("\t\t%v:\n", paych)
						fmt.Printf("\t\t\tBalance %v, Redeemed %v, Nonce %v, Voucher, %v, Network loss voucher %v (%v), Settling at %v, Height checked %v", state.Balance, state.Redeemed, state.Nonce, state.Voucher, state.NetworkLossVoucher, loss, res.SettlingAt, res.CurrentHeight)
					} else {
						fmt.Printf("\t\t%v\n", paych)
					}
				}
			}
		}
		fmt.Println("Done")
		return nil
	},
}

var PaychInactiveOutGetCMD = &cli.Command{
	Name:        "get",
	Usage:       "get paych state",
	Description: "Get the state of a given paych",
	ArgsUsage:   "[currency id, peer addr, paych addr]",
	Action: func(c *cli.Context) error {
		// Parse arguments.
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
		state, err := client.InactiveOutRead(c.Context, byte(currencyID), peerAddr, paychAddr)
		if err != nil {
			return err
		}
		res, err := client.TransactorCheck(c.Context, state.CurrencyID, paychAddr)
		if err != nil {
			return err
		}
		loss := big.NewInt(0)
		if state.NetworkLossVoucher != "" {
			res, err := client.TransactorVerifyVoucher(state.CurrencyID, state.NetworkLossVoucher)
			if err != nil {
				return err
			}
			loss = big.NewInt(0).Sub(res.Redeemed, state.Redeemed)
		}
		fmt.Printf("Channel: %v (Currency %v):\n", paychAddr, currencyID)
		fmt.Printf("\tBalance: %v\n", state.Balance)
		fmt.Printf("\tRedeemed: %v\n", state.Redeemed)
		fmt.Printf("\tNonce: %v\n", state.Nonce)
		fmt.Printf("\tVoucher: %v\n", state.Voucher)
		fmt.Printf("\tNetwork loss voucher: %v (%v)\n", state.NetworkLossVoucher, loss)
		fmt.Printf("\tSettling at: %v\n", res.SettlingAt)
		fmt.Printf("\tCurrent height: %v\n", res.CurrentHeight)
		return nil
	},
}

var PaychActiveInCMD = &cli.Command{
	Name:        "active-in",
	Usage:       "access active in paych functions",
	Description: "This command outputs a list of active in paych functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		PaychActiveInGetCMD,
		PaychActiveInListCMD,
	},
}

var PaychActiveInListCMD = &cli.Command{
	Name:        "list",
	Aliases:     []string{"ls"},
	Usage:       "list all paychs",
	Description: "List all payment channels stored",
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
		curChan := client.ActiveInListCurrencyIDs(c.Context)
		for currencyID := range curChan {
			fmt.Printf("Currency %v\n", currencyID)
			peerChan := client.ActiveInListPeers(c.Context, currencyID)
			for peer := range peerChan {
				fmt.Printf("\tPeer %v\n", peer)
				paychChan := client.ActiveInListPaychsByPeer(c.Context, currencyID, peer)
				for paych := range paychChan {
					if c.IsSet("long") {
						state, err := client.ActiveInRead(c.Context, currencyID, peer, paych)
						if err != nil {
							return err
						}
						res, err := client.PaychMonitorCheck(c.Context, false, currencyID, state.ChAddr)
						if err != nil {
							return err
						}
						fmt.Printf("\t\t%v:\n", paych)
						fmt.Printf("\t\t\tBalance %v, Redeemed %v, Nonce %v, Voucher: %v, Settlement %v, Last updated %v\n", state.Balance, state.Redeemed, state.Nonce, state.Voucher, res.Settlement, res.Updated)
					} else {
						fmt.Printf("\t\t%v\n", paych)
					}
				}
			}
		}
		fmt.Println("Done")
		return nil
	},
}

var PaychActiveInGetCMD = &cli.Command{
	Name:        "get",
	Usage:       "get paych state",
	Description: "Get the state of a given paych",
	ArgsUsage:   "[currency id, peer addr, paych addr]",
	Action: func(c *cli.Context) error {
		// Parse arguments.
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
		state, err := client.ActiveInRead(c.Context, byte(currencyID), peerAddr, paychAddr)
		if err != nil {
			return err
		}
		res, err := client.PaychMonitorCheck(c.Context, false, state.CurrencyID, state.ChAddr)
		if err != nil {
			return err
		}
		fmt.Printf("Channel: %v (Currency %v):\n", paychAddr, currencyID)
		fmt.Printf("\tBalance: %v\n", state.Balance)
		fmt.Printf("\tRedeemed: %v\n", state.Redeemed)
		fmt.Printf("\tNonce: %v\n", state.Nonce)
		fmt.Printf("\tVoucher: %v\n", state.Voucher)
		fmt.Printf("\tSettlement: %v\n", res.Settlement)
		fmt.Printf("\tLast updated: %v\n", res.Updated)
		return nil
	},
}

var PaychInactiveInCMD = &cli.Command{
	Name:        "inactive-in",
	Usage:       "access active in paych functions",
	Description: "This command outputs a list of inactive in paych functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		PaychInactiveInGetCMD,
		PaychInactiveInListCMD,
	},
}

var PaychInactiveInListCMD = &cli.Command{
	Name:        "list",
	Aliases:     []string{"ls"},
	Usage:       "list all paychs",
	Description: "List all payment channels stored",
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
		curChan := client.InactiveInListCurrencyIDs(c.Context)
		for currencyID := range curChan {
			fmt.Printf("Currency %v\n", currencyID)
			peerChan := client.InactiveInListPeers(c.Context, currencyID)
			for peer := range peerChan {
				fmt.Printf("\tPeer %v\n", peer)
				paychChan := client.InactiveInListPaychsByPeer(c.Context, currencyID, peer)
				for paych := range paychChan {
					if c.IsSet("long") {
						state, err := client.ActiveInRead(c.Context, currencyID, peer, paych)
						if err != nil {
							return err
						}
						res, err := client.TransactorCheck(c.Context, currencyID, paych)
						if err != nil {
							return err
						}
						fmt.Printf("\t\t%v:\n", paych)
						fmt.Printf("\t\t\tBalance %v, Redeemed %v, Nonce %v, Voucher %v, Settling at %v, Height checked %v", state.Balance, state.Redeemed, state.Nonce, state.Voucher, res.SettlingAt, res.CurrentHeight)
					} else {
						fmt.Printf("\t\t%v\n", paych)
					}
				}
			}
		}
		fmt.Println("Done")
		return nil
	},
}

var PaychInactiveInGetCMD = &cli.Command{
	Name:        "get",
	Usage:       "get paych state",
	Description: "Get the state of a given paych",
	ArgsUsage:   "[currency id, peer addr, paych addr]",
	Action: func(c *cli.Context) error {
		// Parse arguments.
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
		state, err := client.InactiveInRead(c.Context, byte(currencyID), peerAddr, paychAddr)
		if err != nil {
			return err
		}
		res, err := client.TransactorCheck(c.Context, state.CurrencyID, paychAddr)
		if err != nil {
			return err
		}
		fmt.Printf("Channel: %v (Currency %v):\n", paychAddr, currencyID)
		fmt.Printf("\tBalance: %v\n", state.Balance)
		fmt.Printf("\tRedeemed: %v\n", state.Redeemed)
		fmt.Printf("\tNonce: %v\n", state.Nonce)
		fmt.Printf("\tVoucher: %v\n", state.Voucher)
		fmt.Printf("\tSettling at: %v\n", res.SettlingAt)
		fmt.Printf("\tCurrent height: %v\n", res.CurrentHeight)
		return nil
	},
}
