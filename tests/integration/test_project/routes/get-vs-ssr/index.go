package dashboard

import rstf "github.com/rafbgarcia/rstf"

// Final API shape (v1):
//
//	func SSR(ctx *rstf.Context) SSRData {
//		return SSRData{Message: "dashboard html"}
//	}
//
//	func GET(ctx *rstf.Context) error {
//		return ctx.JSON(200, APIResponse{
//			Source: "get",
//			Route:  "/get-vs-ssr",
//		})
//	}
//
// Runtime dispatch contract:
// - GET + Accept preferring text/html -> SSR
// - GET + non-HTML Accept -> GET

type Post struct {
	Title     string `json:"title"`
	Published bool   `json:"published"`
}

type ServerData struct {
	Posts   []Post `json:"posts"`
	Message string `json:"message"`
}

type APIResponse struct {
	Source string `json:"source"`
	Route  string `json:"route"`
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

func GET(ctx *rstf.Context) error {
	_ = ctx
	return nil
}
