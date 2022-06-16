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
	"path/filepath"
	"strconv"

	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/fcr/api"
)

// Piece command.
var PieceCMD = &cli.Command{
	Name:        "piece",
	Usage:       "access piece manager functions",
	Description: "This command outputs a list of piece manager functions",
	ArgsUsage:   " ",
	Subcommands: []*cli.Command{
		PieceImportCMD,
		PieceImportCarCMD,
		PieceImportSectorCMD,
		PieceListCMD,
		PieceInspectCMD,
		PieceRemoveCMD,
	},
}

var PieceImportCMD = &cli.Command{
	Name:        "import",
	Usage:       "import a file",
	Description: "Import a file into the piece manager",
	ArgsUsage:   "[filename]",
	Action: func(c *cli.Context) error {
		// Parse arguments.
		filename := c.Args().Get(0)
		if filename == "" {
			return usageError(c, fmt.Errorf("received empty filename"))
		}
		filenameAbs, err := filepath.Abs(filename)
		if err != nil {
			return fmt.Errorf("fail to get abs path: %v", err.Error())
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		id, err := client.PieceMgrImport(c.Context, filenameAbs)
		if err != nil {
			return err
		}
		fmt.Printf("Imported: %v\n", id.String())
		return nil
	},
}

var PieceImportCarCMD = &cli.Command{
	Name:        "import-car",
	Usage:       "import a .car file",
	Description: "Import a .car file into the piece manager",
	ArgsUsage:   "[filename]",
	Action: func(c *cli.Context) error {
		// Parse arguments.
		filename := c.Args().Get(0)
		if filename == "" {
			return usageError(c, fmt.Errorf("received empty filename"))
		}
		filenameAbs, err := filepath.Abs(filename)
		if err != nil {
			return fmt.Errorf("fail to get abs path: %v", err.Error())
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		id, err := client.PieceMgrImportCar(c.Context, filenameAbs)
		if err != nil {
			return err
		}
		fmt.Printf("Imported: %v\n", id.String())
		return nil
	},
}

var PieceImportSectorCMD = &cli.Command{
	Name:        "import-sector",
	Usage:       "import a lotus unsealed sector",
	Description: "Import a lotus unsealed sector into the piece manager",
	ArgsUsage:   "[filename, copy]",
	Action: func(c *cli.Context) error {
		// Parse arguments.
		filename := c.Args().Get(0)
		if filename == "" {
			return usageError(c, fmt.Errorf("received empty filename"))
		}
		copy, err := strconv.ParseBool(c.Args().Get(1))
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse copy: %v", err.Error()))
		}
		filenameAbs, err := filepath.Abs(filename)
		if err != nil {
			return fmt.Errorf("fail to get abs path: %v", err.Error())
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		ids, err := client.PieceMgrImportSector(c.Context, filenameAbs, copy)
		if err != nil {
			return err
		}
		fmt.Println("Imported:")
		for _, id := range ids {
			fmt.Printf("\t%v\n", id.String())
		}
		return nil
	},
}

var PieceListCMD = &cli.Command{
	Name:        "list",
	Aliases:     []string{"ls"},
	Usage:       "list all pieces",
	Description: "List all pieces imported to this piece manager",
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
		idChan := client.PieceMgrListImported(c.Context)
		for id := range idChan {
			if c.IsSet("long") {
				res, err := client.PieceMgrInspect(c.Context, id)
				if err != nil {
					return err
				}
				if res.Exists {
					fmt.Printf("%v: path(%v), index(%v), size(%v), copy (%v)\n", id.String(), res.Path, res.Index, res.Size, res.Copy)
				}
			} else {
				fmt.Println(id.String())
			}
		}
		fmt.Println("Done")
		return nil
	},
}

var PieceInspectCMD = &cli.Command{
	Name:        "inspect",
	Usage:       "inspect given piece",
	Description: "Inspect the details of a given piece",
	ArgsUsage:   "[cid]",
	Action: func(c *cli.Context) error {
		id, err := cid.Parse(c.Args().Get(0))
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse cid: %v", err.Error()))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		res, err := client.PieceMgrInspect(c.Context, id)
		if err != nil {
			return err
		}
		if res.Exists {
			fmt.Printf("Piece %v:\n", id.String())
			fmt.Printf("\tpath - %v\n", res.Path)
			fmt.Printf("\tindex - %v\n", res.Index)
			fmt.Printf("\tsize - %v\n", res.Size)
			fmt.Printf("\tcopy - %v\n", res.Copy)
		} else {
			fmt.Printf("Piece %v does not exist\n", id.String())
		}
		return nil
	},
}

var PieceRemoveCMD = &cli.Command{
	Name:        "remove",
	Usage:       "remove given piece",
	Description: "Remove the given piece from the piece manager",
	ArgsUsage:   "[cid]",
	Action: func(c *cli.Context) error {
		id, err := cid.Parse(c.Args().Get(0))
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse cid: %v", err.Error()))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		err = client.PieceMgrRemove(c.Context, id)
		if err != nil {
			return err
		}
		fmt.Println("Succeed")
		return nil
	},
}
