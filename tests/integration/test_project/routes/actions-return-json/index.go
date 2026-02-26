package actionsreturnjson

import rstf "github.com/rafbgarcia/rstf"

type CreatePostInput struct {
	Title string `json:"title"`
}

type Response struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// Final API shape (v1):
//
//	func POST(ctx *rstf.Context) error {
//		var payload CreatePostInput
//		if err := ctx.BindJSON(&payload); err != nil {
//			return err // mapped to 400/413/415 envelope by framework
//		}
//
//		return ctx.JSON(201, Response{
//			ID:     "post_123",
//			Status: "created",
//		})
//	}
func POST(ctx *rstf.Context) error {
	_ = ctx
	return nil
}
