package main

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
	"os"

	"github.com/wcgcyx/fcr/cli"
)

// This is the main program of fcr.
func main() {
	app := cli.NewCLI()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err.Error())
	}
}
