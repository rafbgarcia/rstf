package codegen

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}

func TestParseDir(t *testing.T) {
	routes, err := ParseDir(testdataDir())
	require.NoError(t, err)

	// Sort for deterministic order.
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Dir < routes[j].Dir
	})

	require.Len(t, routes, 2)

	t.Run("dashboard route", func(t *testing.T) {
		rf := routes[0]
		assert.Equal(t, "dashboard", rf.Dir)
		assert.Equal(t, "dashboard", rf.Package)
		require.Len(t, rf.Funcs, 1)

		fn := rf.Funcs[0]
		assert.Equal(t, "SSR", fn.Name)
		assert.True(t, fn.HasContext, "expected HasContext=true")
		assert.Equal(t, "ServerData", fn.ReturnType)

		// Structs: ServerData, Post, and Author (transitive)
		assert.Len(t, rf.Structs, 3)
	})

	t.Run("settings route", func(t *testing.T) {
		rf := routes[1]
		assert.Equal(t, "settings", rf.Dir)
		require.Len(t, rf.Funcs, 1)

		fn := rf.Funcs[0]
		assert.True(t, fn.HasContext, "expected HasContext=true")
		assert.Equal(t, "ServerData", fn.ReturnType)

		// Structs: ServerData and Config (transitive)
		assert.Len(t, rf.Structs, 2)
	})
}

func TestGoTypeToTS(t *testing.T) {
	tests := []struct {
		goType  string
		isSlice bool
		want    string
	}{
		{"string", false, "string"},
		{"int", false, "number"},
		{"int64", false, "number"},
		{"float64", false, "number"},
		{"bool", false, "boolean"},
		{"Post", false, "Post"},
		{"string", true, "string[]"},
		{"Post", true, "Post[]"},
		{"uint32", false, "number"},
	}

	for _, tt := range tests {
		got := goTypeToTS(tt.goType, tt.isSlice)
		assert.Equal(t, tt.want, got, "goTypeToTS(%q, %v)", tt.goType, tt.isSlice)
	}
}

func TestJsonTagName(t *testing.T) {
	routes, err := ParseDir(testdataDir())
	require.NoError(t, err)

	var dashboard *RouteFile
	for i := range routes {
		if routes[i].Dir == "dashboard" {
			dashboard = &routes[i]
			break
		}
	}
	require.NotNil(t, dashboard, "dashboard route not found")

	var post *StructDef
	for i := range dashboard.Structs {
		if dashboard.Structs[i].Name == "Post" {
			post = &dashboard.Structs[i]
			break
		}
	}
	require.NotNil(t, post, "Post struct not found")

	require.Len(t, post.Fields, 2)
	assert.Equal(t, "title", post.Fields[0].JSONName)
	assert.Equal(t, "published", post.Fields[1].JSONName)
}

func TestParseDirSkipsNonRouteFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "helpers", "helpers.go"), `
package helpers

func DoSomething() string {
	return "hi"
}
`)

	routes, err := ParseDir(dir)
	require.NoError(t, err)
	assert.Len(t, routes, 0)
}

func TestParseDirRejectsNestedRouteDirectories(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "routes", "admin", "users", "index.go"), `
package users

type ServerData struct{}

func SSR() ServerData {
	return ServerData{}
}
`)

	_, err := ParseDir(dir)
	require.Error(t, err)
	require.ErrorContains(t, err, `invalid route directory "routes/admin/users"`)
	require.ErrorContains(t, err, "use dotted names like routes/admin.users")
}

func TestParseDirNoContext(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "page", "page.go"), `
package page

type Item struct {
	Name string `+"`json:\"name\"`"+`
}

type ServerData struct {
	Items []Item `+"`json:\"items\"`"+`
}

func SSR() ServerData {
	return ServerData{}
}
`)

	routes, err := ParseDir(dir)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	require.NotEmpty(t, routes[0].Funcs)
	fn := routes[0].Funcs[0]
	assert.False(t, fn.HasContext, "expected HasContext=false for function without context param")
	assert.Equal(t, "ServerData", fn.ReturnType)
}

func TestParseDirDetectsOnServerStart(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "myapp", "main.go"), `
package myapp

import rstf "github.com/rafbgarcia/rstf"

type Session struct {
	UserName string
}

func OnServerStart(app *rstf.App) {
}

func SSR(ctx *rstf.Context) Session {
	return Session{}
}
`)

	routes, err := ParseDir(dir)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	assert.True(t, routes[0].HasOnServerStart, "expected HasOnServerStart=true when OnServerStart(*rstf.App) is exported")
}

func TestParseDirDetectsRPCFunctions(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "routes", "chat._id", "index.go"), `
package chat

import rstf "github.com/rafbgarcia/rstf"

type Message struct {
	Body string `+"`json:\"body\"`"+`
}

type GetMessagesResult struct {
	Messages []Message `+"`json:\"messages\"`"+`
}

type SendMessageInput struct {
	Body string `+"`json:\"body\"`"+`
}

func GetMessages(ctx *rstf.QueryContext) (GetMessagesResult, error) {
	return GetMessagesResult{}, nil
}

func SendMessage(ctx *rstf.MutationContext, input SendMessageInput) error {
	return nil
}

func NotifySlack(ctx *rstf.ActionContext, input string) (string, error) {
	return input, nil
}
`)

	routes, err := ParseDir(dir)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	require.Len(t, routes[0].Funcs, 3)

	assert.Contains(t, routes[0].Funcs, RouteFunc{
		Name:         "GetMessages",
		Kind:         RouteFuncKindQuery,
		ReturnType:   "GetMessagesResult",
		ReturnsError: true,
		HasContext:   true,
	})
	assert.Contains(t, routes[0].Funcs, RouteFunc{
		Name:         "SendMessage",
		Kind:         RouteFuncKindMutation,
		InputType:    "SendMessageInput",
		ReturnsError: true,
		HasContext:   true,
	})
	assert.Contains(t, routes[0].Funcs, RouteFunc{
		Name:         "NotifySlack",
		Kind:         RouteFuncKindAction,
		InputType:    "string",
		ReturnType:   "string",
		ReturnsError: true,
		HasContext:   true,
	})
	assert.Len(t, routes[0].Structs, 3)
}

func TestParseDirOnServerStartWithAlias(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "myapp", "main.go"), `
package myapp

import fw "github.com/rafbgarcia/rstf"

type Session struct {
	UserName string
}

func OnServerStart(app *fw.App) {
}

func SSR(ctx *fw.Context) Session {
	return Session{}
}
`)

	routes, err := ParseDir(dir)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	assert.True(t, routes[0].HasOnServerStart, "expected HasOnServerStart=true with aliased import")
}

func TestParseDirOnServerStartWrongSignature(t *testing.T) {
	dir := t.TempDir()
	// OnServerStart with wrong signature should not be detected.
	writeFile(t, filepath.Join(dir, "myapp", "main.go"), `
package myapp

type Session struct {
	UserName string
}

// Wrong: OnServerStart takes no args.
func OnServerStart() {
}

func SSR() Session {
	return Session{}
}
`)

	routes, err := ParseDir(dir)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	assert.False(t, routes[0].HasOnServerStart, "expected HasOnServerStart=false for OnServerStart() with wrong signature")
}

func TestParseDirNoOnServerStart(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "myapp", "main.go"), `
package myapp

import rstf "github.com/rafbgarcia/rstf"

type Session struct {
	UserName string
}

func SSR(ctx *rstf.Context) Session {
	return Session{}
}
`)

	routes, err := ParseDir(dir)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	assert.False(t, routes[0].HasOnServerStart, "expected HasOnServerStart=false when no OnServerStart function exists")
}

func TestParseDirOnServerStartOnlyNoSSR(t *testing.T) {
	// A package with only OnServerStart() and no SSR should still be parsed
	// (the layout might configure the app without returning server data).
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "myapp", "main.go"), `
package myapp

import rstf "github.com/rafbgarcia/rstf"

func OnServerStart(app *rstf.App) {
}
`)

	routes, err := ParseDir(dir)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	assert.True(t, routes[0].HasOnServerStart, "expected HasOnServerStart=true")
	assert.Len(t, routes[0].Funcs, 0)
}

func TestParseDirDetectsAroundRequest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "myapp", "main.go"), `
package myapp

import rstf "github.com/rafbgarcia/rstf"

type Session struct {
	UserName string
}

func AroundRequest() []rstf.Middleware {
	return nil
}

func SSR(ctx *rstf.Context) Session {
	return Session{}
}
`)

	routes, err := ParseDir(dir)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	assert.True(t, routes[0].HasAroundRequest, "expected HasAroundRequest=true when AroundRequest() []rstf.Middleware is exported")
}

func TestParseDirAroundRequestWithAlias(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "myapp", "main.go"), `
package myapp

import fw "github.com/rafbgarcia/rstf"

type Session struct {
	UserName string
}

func AroundRequest() []fw.Middleware {
	return nil
}

func SSR(ctx *fw.Context) Session {
	return Session{}
}
`)

	routes, err := ParseDir(dir)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	assert.True(t, routes[0].HasAroundRequest, "expected HasAroundRequest=true with aliased import")
}

func TestParseDirAroundRequestWrongSignature(t *testing.T) {
	dir := t.TempDir()
	// AroundRequest with params should not be detected.
	writeFile(t, filepath.Join(dir, "myapp", "main.go"), `
package myapp

import "net/http"

type Session struct {
	UserName string
}

// Wrong: AroundRequest takes a param and returns single middleware.
func AroundRequest(next http.Handler) http.Handler {
	return next
}

func SSR() Session {
	return Session{}
}
`)

	routes, err := ParseDir(dir)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	assert.False(t, routes[0].HasAroundRequest, "expected HasAroundRequest=false for AroundRequest with wrong signature")
}

func TestParseDirAroundRequestOnlyNoSSR(t *testing.T) {
	// A package with only AroundRequest() and no SSR should still be parsed.
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "myapp", "main.go"), `
package myapp

import rstf "github.com/rafbgarcia/rstf"

func AroundRequest() []rstf.Middleware {
	return nil
}
`)

	routes, err := ParseDir(dir)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	assert.True(t, routes[0].HasAroundRequest, "expected HasAroundRequest=true")
	assert.Len(t, routes[0].Funcs, 0)
}

func TestParseDirBothConventions(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "myapp", "main.go"), `
package myapp

import rstf "github.com/rafbgarcia/rstf"

type Session struct {
	UserName string
}

func OnServerStart(app *rstf.App) {
}

func AroundRequest() []rstf.Middleware {
	return nil
}

func SSR(ctx *rstf.Context) Session {
	return Session{}
}
`)

	routes, err := ParseDir(dir)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	assert.True(t, routes[0].HasOnServerStart, "expected HasOnServerStart=true")
	assert.True(t, routes[0].HasAroundRequest, "expected HasAroundRequest=true")
	assert.Len(t, routes[0].Funcs, 1)
}

func TestParseDirSkipsNonStructReturns(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "api", "api.go"), `
package api

func SSR() string {
	return "test"
}
`)

	routes, err := ParseDir(dir)
	require.NoError(t, err)
	// SSR() returns a primitive — should be skipped.
	assert.Len(t, routes, 0)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, err, "mkdir %s", filepath.Dir(path))
	err = os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err, "writing %s", path)
}
