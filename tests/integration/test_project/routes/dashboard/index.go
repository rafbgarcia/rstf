package dashboard

import rstf "github.com/rafbgarcia/rstf"

type Post struct {
	Title     string `json:"title"`
	Published bool   `json:"published"`
}

type ServerData struct {
	Posts   []Post `json:"posts"`
	Message string `json:"message"`
}

func SSR(ctx *rstf.Context) ServerData {
	return ServerData{
		Message: "Welcome to the dashboard!",
		Posts: []Post{
			{Title: "First Post", Published: true},
			{Title: "Draft Post", Published: false},
		},
	}
}
