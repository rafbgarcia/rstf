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
		"export type RouteParamValue = string | number;",
		`export type RouteName =`,
		`"index"`,
		`"users.$id"`,
		`"users.$id": { path: "/users/$id", params: ["id"] }`,
		`export function url(name: "index"): string;`,
		`export function url(name: "users.$id", params: { id: RouteParamValue }): string;`,
		"path = path.replace(`$${paramName}`, encodeURIComponent(String(value)));",
		"export const routes = { url } as const;",
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
		"func URL[P any](route Route[P], params P) Location {",
		"type IndexParams struct {",
		"func IndexURL() Location {",
		"type UsersParamIdParams struct {",
		"\tId string",
		"var UsersParamId = Route[UsersParamIdParams]{",
		`name: "users.$id",`,
		`pattern: "/users/{id}",`,
		`return Location("/users/" + url.PathEscape(params.Id))`,
		"func UsersParamIdURL(params UsersParamIdParams) Location {",
	} {
		assert.Contains(t, got, expected, "missing %q\n\n%s", expected, got)
	}
}
