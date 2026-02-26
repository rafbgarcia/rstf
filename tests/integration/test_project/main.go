package testproject

import (
	"time"

	rstf "github.com/rafbgarcia/rstf"
)

type ServerData struct {
	AppName string `json:"appName"`
}

func SSR(ctx *rstf.Context) ServerData {
	return ServerData{
		AppName: "Basic Example",
	}
}

func OnServerStart(app *rstf.App) {
	if err := app.SetRequestBodyLimitBytes(1024); err != nil {
		panic(err)
	}
	if err := app.SetMaxConcurrentRequests(1); err != nil {
		panic(err)
	}
	if err := app.SetMaxQueuedRequests(1); err != nil {
		panic(err)
	}
	if err := app.SetQueueTimeout(100 * time.Millisecond); err != nil {
		panic(err)
	}
}
