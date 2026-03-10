package useravatar

import rstf "github.com/rafbgarcia/rstf"

type ServerData struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func SSR(ctx *rstf.Context) ServerData {
	return ServerData{
		Name:   "Ada Lovelace",
		Status: "staff",
	}
}
