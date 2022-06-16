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

	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/fcr/version"
)

// NewCLI creates a CLI app.
func NewCLI() *cli.App {
	app := &cli.App{
		Name:      "fcr",
		HelpName:  "fcr",
		Usage:     "A Filecoin secondary retrieval client",
		UsageText: "fcr [global options] command [arguments...]",
		Version:   version.Version,
		Description: "\n\t This is a Filecoin secondary retrieval client featured with\n" +
			"\t the ability to participate in an ipld retrieval network and\n" +
			"\t a payment proxy network.\n\n" +
			"\t You can use it as a retrieval client to retrieve files using\n" +
			"\t Filecoin (FIL).\n\n" +
			"\t -OR-\n\n" +
			"\t You can earn FIL by importing a regular file or a .car file\n" +
			"\t or a lotus unsealed sector and serving the ipld graph to the\n" +
			"\t network for paid retrieval.\n\n" +
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
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Value:   9424,
				Usage:   "specify fcr local api port",
			},
			&cli.PathFlag{
				Name:    "auth",
				Aliases: []string{"a"},
				Value:   "",
				Usage:   "specify fcr api token file",
			},
		},
	}
	app.Commands = []*cli.Command{
		DaemonCMD,
		WalletCMD,
		CIDNetCMD,
		PayNetCMD,
		PeerCMD,
		RetrieveCMD,
		RetrieveCacheCMD,
		FastRetrieveCMD,
		SystemCMD,
		{
			Name:        "version",
			Usage:       "get fcr version",
			Description: "Get the fcr version",
			ArgsUsage:   " ",
			Action: func(c *cli.Context) error {
				fmt.Println("fcr version: ", version.Version)
				return nil
			},
		},
	}
	return app
}

// usageError is used to generate the usage error.
//
// @input - cli context, error.
//
// @output - error.
func usageError(c *cli.Context, err error) error {
	fmt.Println("Usage:", c.App.Name, c.Command.Name, c.Command.ArgsUsage)
	return fmt.Errorf("Incorrect usage: %v", err.Error())
}
