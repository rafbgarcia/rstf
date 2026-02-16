package testproject

import rstf "github.com/rafbgarcia/rstf"

type ServerData struct {
	AppName string `json:"appName"`
}

func SSR(ctx *rstf.Context) ServerData {
	return ServerData{
		AppName: "Basic Example",
	}
}
