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
	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/fcr/daemon"
)

// Daemon command.
var DaemonCMD = &cli.Command{
	Name:        "daemon",
	Usage:       "start daemon",
	Description: "This command starts a running FCR daemon",
	ArgsUsage:   " ",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Value:   "",
			Usage:   "specify config file",
		},
	},
	Action: func(c *cli.Context) error {
		return daemon.Daemon(c.Context, c.String("config"))
	},
}
