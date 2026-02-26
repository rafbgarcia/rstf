package actionsreturnnull

import rstf "github.com/rafbgarcia/rstf"

// Final API shape (v1):
//
//	func POST(ctx *rstf.Context) error {
//		return nil // framework writes 204 No Content when nothing was written
//	}
func POST(ctx *rstf.Context) error {
	_ = ctx
	return nil
}
