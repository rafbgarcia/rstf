package actionsredirect

import rstf "github.com/rafbgarcia/rstf"

func POST(ctx *rstf.Context) error {
	return ctx.Redirect(303, "/get-vs-ssr")
}
