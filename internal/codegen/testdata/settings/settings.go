package settings

import rstf "github.com/rafbgarcia/rstf"

type Config struct {
	Theme    string  `json:"theme"`
	FontSize int     `json:"fontSize"`
	Beta     bool    `json:"beta"`
	Score    float64 `json:"score"`
}

func SSR(ctx *rstf.Context) (config Config, title string) {
	return
}
