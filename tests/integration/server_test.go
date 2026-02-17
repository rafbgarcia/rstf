package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rafbgarcia/rstf/internal/codegen"
	"github.com/rafbgarcia/rstf/renderer"
)

func TestEndToEnd(t *testing.T) {
	root := testProjectRoot()

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

	// Step 4: Verify full HTML output.
	// Strip React's <!-- --> comment nodes (internal text boundary markers)
	// so the expected string is clean and not coupled to React internals.
	got := strings.ReplaceAll(html, "<!-- -->", "")

	want := `<html><head><title>Basic Example</title><title>Dashboard - Welcome to the dashboard!</title></head><body><header><h1>Basic Example</h1><nav><a href="/dashboard">Dashboard</a></nav></header><main><div><h2 class="text-blue-500">Welcome to the dashboard!</h2><button data-testid="counter">Count: 0</button><ul><li>First Post (published)</li><li>Draft Post (draft)</li></ul></div></main></body></html>`

	if got != want {
		t.Errorf("HTML mismatch.\n\nGot:\n%s\n\nWant:\n%s", got, want)
	}
}
