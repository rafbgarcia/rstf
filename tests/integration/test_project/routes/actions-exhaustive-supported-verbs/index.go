package actionsexhaustivesupportedverbs

import rstf "github.com/rafbgarcia/rstf"

type Response struct {
	Method string `json:"method"`
	Route  string `json:"route"`
}

func GET(ctx *rstf.Context) error {
	return ctx.JSON(200, Response{Method: "GET", Route: "/actions-exhaustive-supported-verbs"})
}

func POST(ctx *rstf.Context) error {
	return ctx.JSON(200, Response{Method: "POST", Route: "/actions-exhaustive-supported-verbs"})
}

func PUT(ctx *rstf.Context) error {
	return ctx.JSON(200, Response{Method: "PUT", Route: "/actions-exhaustive-supported-verbs"})
}

func PATCH(ctx *rstf.Context) error {
	return ctx.JSON(200, Response{Method: "PATCH", Route: "/actions-exhaustive-supported-verbs"})
}

func DELETE(ctx *rstf.Context) error {
	return ctx.JSON(200, Response{Method: "DELETE", Route: "/actions-exhaustive-supported-verbs"})
}
