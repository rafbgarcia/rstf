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

	// Step 4: Verify HTML output.
	checks := []string{
		"<html",                      // Layout rendered
		"Basic Example",              // Layout server data
		"Welcome to the dashboard!", // Route server data
		"First Post",                 // Post title
		"Draft Post",                 // Second post
		"(published)",                // Published indicator
		"(draft)",                    // Draft indicator
	}
	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("HTML missing %q\n\nFull HTML:\n%s", check, html)
		}
	}
}
