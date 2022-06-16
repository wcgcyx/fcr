package piecemgr

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

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
	"github.com/ipld/go-ipld-prime"
)

// Logger
var log = logging.Logger("piecemgr")

// PieceManager is an interface for a piece manager that is capable of
// importing a file, a car file and an unsealed sector.
type PieceManager interface {
	// Import imports a file.
	//
	// @input - context, file path.
	//
	// @output - cid imported, error.
	Import(ctx context.Context, path string) (cid.Cid, error)

	// ImportCar imports a car file.
	//
	// @input - context, file path.
	//
	// @output - cid imported, error.
	ImportCar(ctx context.Context, path string) (cid.Cid, error)

	// ImportSector imports an unsealed sector copy.
	//
	// @input - context, file path, whether to keep a copy in datastore.
	//
	// @output - list of cid imported, error.
	ImportSector(ctx context.Context, path string, copy bool) ([]cid.Cid, error)

	// ListImported lists all imported pieces.
	//
	// @input - context.
	//
	// @output - cid chan out, error chan out.
	ListImported(ctx context.Context) (<-chan cid.Cid, <-chan error)

	// Inspect inspects a piece.
	//
	// @input - context, cid.
	//
	// @output - if exists, path, index (applicable in sector), the size, whether a copy is kept, error.
	Inspect(ctx context.Context, id cid.Cid) (bool, string, int, uint64, bool, error)

	// Remove removes a piece.
	//
	// @input - context, cid.
	//
	// @output - error.
	Remove(ctx context.Context, id cid.Cid) error

	// LinkSystem returns the linksystem for retrieval.
	//
	// @output - linksystem.
	LinkSystem() ipld.LinkSystem
}
