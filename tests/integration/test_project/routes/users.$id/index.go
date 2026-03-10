package usersid

import rstf "github.com/rafbgarcia/rstf"

func GET(ctx *rstf.Context) error {
	id := ctx.Request.PathValue("id")
	return ctx.JSON(200, map[string]any{
		"id":    id,
		"route": "/users/" + id,
	})
}
