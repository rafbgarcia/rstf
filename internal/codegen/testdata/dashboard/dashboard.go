package dashboard

import rstf "github.com/rafbgarcia/rstf"

type Post struct {
	Title     string `json:"title"`
	Published bool   `json:"published"`
}

type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type ServerData struct {
	Posts  []Post `json:"posts"`
	Author Author `json:"author"`
}

func SSR(ctx *rstf.Context) ServerData {
	return ServerData{}
}
