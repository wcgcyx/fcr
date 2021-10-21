package piecemgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
)

// PieceManager is an interface for a piece manager.
// It is capable of importing a file, a car file and an unsealed sector.
type PieceManager interface {
	// Import imports a file.
	// It takes a context and a path as arguments.
	// It returns the cid imported and error.
	Import(ctx context.Context, path string) (cid.Cid, error)

	// ImportCar imports a car file.
	// It takes a context and a path as arguments.
	// It returns the cid imported and error.
	ImportCar(ctx context.Context, path string) (cid.Cid, error)

	// ImportSector imports an filecoin lotus unsealed sector copy.
	// It takes a context and a path, a boolean indicating whether to keep a copy as arguments.
	// It returns the a list of cids imported and error.
	ImportSector(ctx context.Context, path string, copy bool) ([]cid.Cid, error)

	// ListImported lists all imported pieces.
	// It takes a context as the argument.
	// It returns a list of cids imported and error.
	ListImported(ctx context.Context) ([]cid.Cid, error)

	// Inspect inspects a piece.
	// It takes a cid as the argument.
	// It returns the path, the index, the size, a boolean indicating whether a copy is kept and error.
	Inspect(id cid.Cid) (string, int, int64, bool, error)

	// Remove removes a piece.
	// It takes a cid as the argument.
	// It returns error.
	Remove(id cid.Cid) error

	// LinkSystem returns the linksystem.
	// It returns the linksystem.
	LinkSystem() ipld.LinkSystem
}
