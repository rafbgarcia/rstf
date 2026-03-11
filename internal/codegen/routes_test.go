package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRouteDefs(t *testing.T) {
	files := []RouteFile{
		{
			Dir: "routes/index",
			Funcs: []RouteFunc{
				{Name: "ListPosts", Kind: RouteFuncKindQuery, ReturnType: "ListPostsResult"},
			},
		},
		{
			Dir: "routes/users.$id",
			Funcs: []RouteFunc{
				{Name: "SendMessage", Kind: RouteFuncKindMutation, InputType: "SendMessageInput"},
			},
		},
	}
	deps := map[string][]string{
		"routes/no-server": nil,
		"shared/ui/card":   nil,
	}

	got := BuildRouteDefs(files, deps)

	assert.Equal(t, []RouteDef{
		{
			Dir:      "routes/index",
			Name:     "index",
			Pattern:  "/",
			RPCFuncs: []RPCFuncDef{{Name: "ListPosts", Kind: RouteFuncKindQuery, ReturnType: "ListPostsResult"}},
		},
		{
			Dir:     "routes/no-server",
			Name:    "no-server",
			Pattern: "/no-server",
		},
		{
			Dir:     "routes/users.$id",
			Name:    "users.$id",
			Pattern: "/users/{id}",
			Params: []RouteParamDef{
				{Name: "id", GoField: "Id"},
			},
			RPCFuncs: []RPCFuncDef{{Name: "SendMessage", Kind: RouteFuncKindMutation, InputType: "SendMessageInput"}},
		},
	}, got)
}

func TestGenerateRoutesTS(t *testing.T) {
	got := GenerateRoutesTS([]RouteDef{
		{
			Name:    "index",
			Pattern: "/",
		},
		{
			Name:    "users.$id",
			Pattern: "/users/{id}",
			Params: []RouteParamDef{
				{Name: "id", GoField: "Id"},
			},
			Dir: "routes/users.$id",
			RPCFuncs: []RPCFuncDef{
				{Name: "GetMessages", Kind: RouteFuncKindQuery, ReturnType: "GetMessagesResult"},
				{Name: "SendMessage", Kind: RouteFuncKindMutation, InputType: "SendMessageInput"},
			},
		},
	})

	for _, expected := range []string{
		`import { defineAction, defineMutation, defineQuery, useAction, useMutation, useQuery } from "./client";`,
		`export const routes = {`,
		`"index": {`,
		`pattern: "/",`,
		`url(): string {`,
		`return "/";`,
		`"users.$id": {`,
		`pattern: "/users/{id}",`,
		`url(params: { id: string }): string {`,
		`return "/users/" + encodeURIComponent(params.id);`,
		`GetMessages: defineQuery<{ id: string }, RoutesUsersId.GetMessagesResult>("users.$id", "GetMessages"),`,
		`SendMessage: defineMutation<{ id: string }, RoutesUsersId.SendMessageInput, void>("users.$id", "SendMessage"),`,
		`export { useAction, useMutation, useQuery };`,
		`export type RouteName = keyof typeof routes;`,
	} {
		assert.Contains(t, got, expected, "missing %q\n\n%s", expected, got)
	}
}

func TestGenerateRoutesGo(t *testing.T) {
	got := GenerateRoutesGo([]RouteDef{
		{
			Name:    "index",
			Pattern: "/",
		},
		{
			Name:    "users.$id",
			Pattern: "/users/{id}",
			Params: []RouteParamDef{
				{Name: "id", GoField: "Id"},
			},
			RPCFuncs: []RPCFuncDef{
				{Name: "GetMessages", Kind: RouteFuncKindQuery},
			},
		},
	})

	for _, expected := range []string{
		"package routes",
		"type Location string",
		"type IndexParams struct {",
		"type IndexRoute struct{}",
		"func (IndexRoute) URL() Location {",
		"var Index IndexRoute",
		"type UsersDotIdParams struct {",
		"\tId string",
		"type UsersDotIdRoute struct{}",
		`func (UsersDotIdRoute) Name() string { return "users.$id" }`,
		`func (UsersDotIdRoute) Pattern() string { return "/users/{id}" }`,
		`return Location("/users/" + url.PathEscape(params.Id))`,
		"func (UsersDotIdRoute) URL(params UsersDotIdParams) Location {",
		"var UsersDotId UsersDotIdRoute",
		`type QueryKey[P any] struct {`,
		`func (q QueryKey[P]) Invalidate(ctx *rstf.MutationContext, params P) {`,
		`return rstf.NewSubscriptionKey("users.$id", "GetMessages", map[string]string{"id": params.Id})`,
		"var UsersDotIdGetMessages = QueryKey[UsersDotIdParams]{",
	} {
		assert.Contains(t, got, expected, "missing %q\n\n%s", expected, got)
	}
}
