package codegen

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}

func TestParseDir(t *testing.T) {
	routes, err := ParseDir(testdataDir())
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	// Sort for deterministic order.
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Dir < routes[j].Dir
	})

	if len(routes) != 2 {
		t.Fatalf("expected 2 route dirs, got %d", len(routes))
	}

	t.Run("dashboard route", func(t *testing.T) {
		rf := routes[0]
		if rf.Dir != "dashboard" {
			t.Errorf("expected dir=dashboard, got %s", rf.Dir)
		}
		if rf.Package != "dashboard" {
			t.Errorf("expected package=dashboard, got %s", rf.Package)
		}
		if len(rf.Funcs) != 1 {
			t.Fatalf("expected 1 func, got %d", len(rf.Funcs))
		}

		fn := rf.Funcs[0]
		if fn.Name != "SSR" {
			t.Errorf("expected func name=SSR, got %s", fn.Name)
		}
		if !fn.HasContext {
			t.Error("expected HasContext=true")
		}
		if fn.ReturnType != "ServerData" {
			t.Errorf("expected ReturnType=ServerData, got %s", fn.ReturnType)
		}

		// Structs: ServerData, Post, and Author (transitive)
		if len(rf.Structs) != 3 {
			t.Fatalf("expected 3 structs, got %d", len(rf.Structs))
		}
	})

	t.Run("settings route", func(t *testing.T) {
		rf := routes[1]
		if rf.Dir != "settings" {
			t.Errorf("expected dir=settings, got %s", rf.Dir)
		}
		if len(rf.Funcs) != 1 {
			t.Fatalf("expected 1 func, got %d", len(rf.Funcs))
		}

		fn := rf.Funcs[0]
		if !fn.HasContext {
			t.Error("expected HasContext=true")
		}
		if fn.ReturnType != "ServerData" {
			t.Errorf("expected ReturnType=ServerData, got %s", fn.ReturnType)
		}

		// Structs: ServerData and Config (transitive)
		if len(rf.Structs) != 2 {
			t.Fatalf("expected 2 structs, got %d", len(rf.Structs))
		}
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
		if got != tt.want {
			t.Errorf("goTypeToTS(%q, %v) = %q, want %q", tt.goType, tt.isSlice, got, tt.want)
		}
	}
}

func TestJsonTagName(t *testing.T) {
	routes, err := ParseDir(testdataDir())
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	var dashboard *RouteFile
	for i := range routes {
		if routes[i].Dir == "dashboard" {
			dashboard = &routes[i]
			break
		}
	}
	if dashboard == nil {
		t.Fatal("dashboard route not found")
	}

	var post *StructDef
	for i := range dashboard.Structs {
		if dashboard.Structs[i].Name == "Post" {
			post = &dashboard.Structs[i]
			break
		}
	}
	if post == nil {
		t.Fatal("Post struct not found")
	}

	if len(post.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(post.Fields))
	}
	if post.Fields[0].JSONName != "title" {
		t.Errorf("field 0 jsonName: got %q, want %q", post.Fields[0].JSONName, "title")
	}
	if post.Fields[1].JSONName != "published" {
		t.Errorf("field 1 jsonName: got %q, want %q", post.Fields[1].JSONName, "published")
	}
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
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if len(routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(routes))
	}
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
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	fn := routes[0].Funcs[0]
	if fn.HasContext {
		t.Error("expected HasContext=false for function without context param")
	}
	if fn.ReturnType != "ServerData" {
		t.Errorf("expected ReturnType=ServerData, got %s", fn.ReturnType)
	}
}

func TestParseDirDetectsAppFunc(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "myapp", "main.go"), `
package myapp

import rstf "github.com/rafbgarcia/rstf"

type Session struct {
	UserName string
}

func App(app *rstf.App) {
}

func SSR(ctx *rstf.Context) Session {
	return Session{}
}
`)

	routes, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if !routes[0].HasApp {
		t.Error("expected HasApp=true when App(*rstf.App) is exported")
	}
}

func TestParseDirAppFuncWithAlias(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "myapp", "main.go"), `
package myapp

import fw "github.com/rafbgarcia/rstf"

type Session struct {
	UserName string
}

func App(app *fw.App) {
}

func SSR(ctx *fw.Context) Session {
	return Session{}
}
`)

	routes, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if !routes[0].HasApp {
		t.Error("expected HasApp=true with aliased import")
	}
}

func TestParseDirAppFuncWrongSignature(t *testing.T) {
	dir := t.TempDir()
	// App with wrong signature should not be detected.
	writeFile(t, filepath.Join(dir, "myapp", "main.go"), `
package myapp

type Session struct {
	UserName string
}

// Wrong: App takes no args.
func App() {
}

func SSR() Session {
	return Session{}
}
`)

	routes, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].HasApp {
		t.Error("expected HasApp=false for App() with wrong signature")
	}
}

func TestParseDirNoAppFunc(t *testing.T) {
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
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].HasApp {
		t.Error("expected HasApp=false when no App function exists")
	}
}

func TestParseDirAppOnlyNoSSR(t *testing.T) {
	// A package with only App() and no SSR should still be parsed
	// (the layout might configure the app without returning server data).
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "myapp", "main.go"), `
package myapp

import rstf "github.com/rafbgarcia/rstf"

func App(app *rstf.App) {
}
`)

	routes, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 route (App-only), got %d", len(routes))
	}
	if !routes[0].HasApp {
		t.Error("expected HasApp=true")
	}
	if len(routes[0].Funcs) != 0 {
		t.Errorf("expected 0 route funcs, got %d", len(routes[0].Funcs))
	}
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
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	// SSR() returns a primitive — should be skipped.
	if len(routes) != 0 {
		t.Errorf("expected 0 routes (primitive return), got %d", len(routes))
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	err = os.WriteFile(path, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
