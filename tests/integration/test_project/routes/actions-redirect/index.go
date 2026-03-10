package actionsredirect

import (
	rstf "github.com/rafbgarcia/rstf"
	"github.com/rafbgarcia/rstf/tests/integration/test_project/.rstf/routes"
)

func POST(ctx *rstf.Context) error {
	return ctx.RedirectTo(303, routes.URL(routes.UsersParamId, routes.UsersParamIdParams{
		Id: "123",
	}))
}
