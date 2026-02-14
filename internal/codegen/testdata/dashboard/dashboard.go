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

func SSR(ctx *rstf.Context) (posts []Post, author Author) {
	return
}
