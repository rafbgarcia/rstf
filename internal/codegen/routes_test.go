package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRouteDefs(t *testing.T) {
	files := []RouteFile{
		{Dir: "routes/index"},
		{Dir: "routes/users.$id"},
	}
	deps := map[string][]string{
		"routes/no-server": nil,
		"shared/ui/card":   nil,
	}

	got := BuildRouteDefs(files, deps)

	assert.Equal(t, []RouteDef{
		{
			Dir:     "routes/index",
			Name:    "index",
			Pattern: "/",
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
		},
	})

	for _, expected := range []string{
		`export const routes = {`,
		`"index": {`,
		`name: "index",`,
		`pattern: "/",`,
		`url(): string {`,
		`return "/";`,
		`"users.$id": {`,
		`pattern: "/users/{id}",`,
		`url(params: { id: string }): string {`,
		`return "/users/" + encodeURIComponent(params.id);`,
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
	} {
		assert.Contains(t, got, expected, "missing %q\n\n%s", expected, got)
	}
}
