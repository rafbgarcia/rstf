package settings

import rstf "github.com/rafbgarcia/rstf"

type Config struct {
	Theme    string  `json:"theme"`
	FontSize int     `json:"fontSize"`
	Beta     bool    `json:"beta"`
	Score    float64 `json:"score"`
}

type ServerData struct {
	Config Config `json:"config"`
	Title  string `json:"title"`
}

func SSR(ctx *rstf.Context) ServerData {
	return ServerData{}
}
