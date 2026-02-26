package admissionslow

import (
	"time"

	rstf "github.com/rafbgarcia/rstf"
)

type Response struct {
	OK bool `json:"ok"`
}

func GET(ctx *rstf.Context) error {
	time.Sleep(250 * time.Millisecond)
	return ctx.JSON(200, Response{OK: true})
}
