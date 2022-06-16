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
	"time"

	crypto2 "github.com/filecoin-project/go-crypto"
	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/fcr/api"
	"github.com/wcgcyx/fcr/crypto"
)

// Wallet command.
var WalletCMD = &cli.Command{
	Name:        "wallet",
	Usage:       "access wallet functions",
	Description: "This command outputs a list of wallet functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		WalletGenerateCMD,
		WalletSetKeyCMD,
		WalletGetAddrCMD,
		WalletList,
		WalletRetireKey,
		WalletStopRetire,
	},
}

var WalletGenerateCMD = &cli.Command{
	Name:        "generate",
	Usage:       "generate a private key",
	Description: "Generate a private key for given currency id",
	ArgsUsage:   "[currency id]",
	Hidden:      true,
	Action: func(c *cli.Context) error {
		// Parse arguments.
		currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		if byte(currencyID) == crypto.FIL {
			prv, err := crypto2.GenerateKey()
			if err != nil {
				return err
			}
			fmt.Println(base64.StdEncoding.EncodeToString(prv))
			return nil
		}
		return fmt.Errorf("Unsupported currency id %v", currencyID)
	},
}

var WalletSetKeyCMD = &cli.Command{
	Name:        "set",
	Usage:       "set the wallet key",
	Description: "Set a private key for given currency id",
	ArgsUsage:   "[currency id, key type, base64 key]",
	Action: func(c *cli.Context) error {
		// Parse arguments.
		currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		keyType, err := strconv.ParseUint(c.Args().Get(1), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse key type: %v", err.Error()))
		}
		key, err := base64.StdEncoding.DecodeString(c.Args().Get(2))
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse key: %v", err.Error()))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.SignerSetKey(c.Context, byte(currencyID), byte(keyType), key)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}

var WalletGetAddrCMD = &cli.Command{
	Name:        "get",
	Usage:       "get the wallet address",
	Description: "Get the address of the stored key for given currency id",
	ArgsUsage:   "[currency id]",
	Action: func(c *cli.Context) error {
		// Parse arguments.
		currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		res, err := client.SignerGetAddr(c.Context, byte(currencyID))
		if err != nil {
			return err
		}
		bal, err := client.TransactorGetBalance(c.Context, byte(currencyID), res.Addr)
		if err != nil {
			return err
		}
		fmt.Printf("%v (Type %v): %v\n", res.Addr, res.KeyType, bal)
		return nil
	},
}

var WalletList = &cli.Command{
	Name:        "list",
	Aliases:     []string{"ls"},
	Usage:       "list all addresses",
	Description: "List the address of stored key for all configured currency ids",
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
		curChan := client.SignerListCurrencyIDs(c.Context)
		for currencyID := range curChan {
			if c.IsSet("long") {
				res, err := client.SignerGetAddr(c.Context, byte(currencyID))
				if err != nil {
					return err
				}
				bal, err := client.TransactorGetBalance(c.Context, byte(currencyID), res.Addr)
				if err != nil {
					return err
				}
				fmt.Printf("Currency %v: %v (Type %v) with balance %v\n", currencyID, res.Addr, res.KeyType, bal)
			} else {
				fmt.Printf("Currency %v\n", currencyID)
			}
		}
		fmt.Println("Done")
		return nil
	},
}

var WalletRetireKey = &cli.Command{
	Name:        "retire",
	Usage:       "retire key",
	Description: "Retire a key for given currency id",
	ArgsUsage:   "[currency id]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "confirm",
			Value: false,
			Usage: "confirm by setting true",
		},
		&cli.DurationFlag{
			Name:   "timeout",
			Value:  time.Hour,
			Usage:  "specify the timeout",
			Hidden: true,
		},
	},
	Hidden: true,
	Action: func(c *cli.Context) error {
		// Parse arguments.
		currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		if !c.IsSet("confirm") {
			return fmt.Errorf("you must pass confirm flag to confirm this")
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		errStr := <-client.SignerRetireKey(c.Context, byte(currencyID), c.Duration("timeout"))
		if errStr != "" {
			return fmt.Errorf(errStr)
		}
		fmt.Println("Succeed")
		return nil
	},
}

var WalletStopRetire = &cli.Command{
	Name:        "stop-retire",
	Usage:       "stop retiring a key",
	Description: "Stop retiring a key for given currency id",
	ArgsUsage:   "[currency id]",
	Hidden:      true,
	Action: func(c *cli.Context) error {
		// Parse arguments.
		currencyID, err := strconv.ParseUint(c.Args().Get(0), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.SignerStopRetire(c.Context, byte(currencyID))
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}
