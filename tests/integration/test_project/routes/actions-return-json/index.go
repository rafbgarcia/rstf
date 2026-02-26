package actionsreturnjson

import rstf "github.com/rafbgarcia/rstf"

type CreatePostInput struct {
	Title string `json:"title"`
}

type Response struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func POST(ctx *rstf.Context) error {
	var payload CreatePostInput
	if err := ctx.BindJSON(&payload); err != nil {
		return err
	}
	if payload.Title == "" {
		return rstf.ValidationError("title is required", map[string]any{
			"field": "title",
		})
	}

	return ctx.JSON(201, Response{
		ID:     "post_123",
		Status: "created",
	})
}
