package actionsredirect

import rstf "github.com/rafbgarcia/rstf"

// Final API shape (v1):
//
//	func POST(ctx *rstf.Context) error {
//		return ctx.Redirect(303, "/dashboard")
//	}
func POST(ctx *rstf.Context) error {
	return nil
}
