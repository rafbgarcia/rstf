package basic_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rafbgarcia/rstf/internal/codegen"
	"github.com/rafbgarcia/rstf/internal/renderer"
)

func projectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

func TestCodegen(t *testing.T) {
	root := projectRoot()

	routeCount, err := codegen.Generate(root)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Join(root, ".rstf")) })

	if routeCount != 1 {
		t.Errorf("expected 1 route, got %d", routeCount)
	}

	// Verify generated files exist.
	expectedFiles := []string{
		".rstf/server_gen.go",
		".rstf/types/main.d.ts",
		".rstf/types/dashboard.d.ts",
		".rstf/generated/main.ts",
		".rstf/generated/routes/dashboard.ts",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(root, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// Verify DTS content.
	mainDTS, _ := os.ReadFile(filepath.Join(root, ".rstf/types/main.d.ts"))
	if !strings.Contains(string(mainDTS), "declare namespace Main") {
		t.Errorf("main.d.ts missing Main namespace:\n%s", mainDTS)
	}
	if !strings.Contains(string(mainDTS), "appName: string") {
		t.Errorf("main.d.ts missing appName field:\n%s", mainDTS)
	}

	dashDTS, _ := os.ReadFile(filepath.Join(root, ".rstf/types/dashboard.d.ts"))
	if !strings.Contains(string(dashDTS), "declare namespace RoutesDashboard") {
		t.Errorf("dashboard.d.ts missing RoutesDashboard namespace:\n%s", dashDTS)
	}
	if !strings.Contains(string(dashDTS), "posts: Post[]") {
		t.Errorf("dashboard.d.ts missing posts field:\n%s", dashDTS)
	}

	// Verify server_gen.go content.
	serverCode, _ := os.ReadFile(filepath.Join(root, ".rstf/server_gen.go"))
	serverStr := string(serverCode)
	for _, expected := range []string{
		"package main",
		`app "github.com/rafbgarcia/rstf/examples/basic"`,
		`dashboard "github.com/rafbgarcia/rstf/examples/basic/routes/dashboard"`,
		`Component: "routes/dashboard"`,
		`Layout:    "main"`,
		"structToMap(app.SSR(ctx))",
		"structToMap(dashboard.SSR(ctx))",
	} {
		if !strings.Contains(serverStr, expected) {
			t.Errorf("server_gen.go missing %q", expected)
		}
	}
}

func TestEndToEnd(t *testing.T) {
	root := projectRoot()

	// Step 1: Run codegen.
	_, err := codegen.Generate(root)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Join(root, ".rstf")) })

	// Step 2: Start the renderer sidecar.
	r := renderer.New()
	if err := r.Start(root); err != nil {
		t.Fatalf("renderer.Start: %v", err)
	}
	t.Cleanup(func() { r.Stop() })

	// Step 3: Render the dashboard route (same request that server_gen.go would make).
	html, err := r.Render(renderer.RenderRequest{
		Component: "routes/dashboard",
		Layout:    "main",
		ServerData: map[string]map[string]any{
			"main": {
				"appName": "Basic Example",
			},
			"routes/dashboard": {
				"message": "Welcome to the dashboard!",
				"posts": []map[string]any{
					{"title": "First Post", "published": true},
					{"title": "Draft Post", "published": false},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	// Step 4: Verify HTML output.
	checks := []string{
		"<html",              // Layout rendered
		"Basic Example",      // Layout server data
		"Welcome to the dashboard!", // Route server data
		"First Post",         // Post title
		"Draft Post",         // Second post
		"(published)",        // Published indicator
		"(draft)",            // Draft indicator
	}
	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("HTML missing %q\n\nFull HTML:\n%s", check, html)
		}
	}
}
