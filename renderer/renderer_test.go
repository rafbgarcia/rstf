package renderer

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

func startRenderer(t *testing.T) *Renderer {
	t.Helper()
	r := New()
	if err := r.Start(testdataDir()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { r.Stop() })
	return r
}

func TestStartStop(t *testing.T) {
	r := New()
	if err := r.Start(testdataDir()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if r.port == 0 {
		t.Fatal("expected non-zero port")
	}
	if err := r.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestRenderWithServerData(t *testing.T) {
	r := startRenderer(t)

	html, err := r.Render(RenderRequest{
		Component: "hello/hello",
		Layout:    "layout/layout",
		ServerData: map[string]map[string]any{
			"hello/hello": {
				"name":  "World",
				"count": 42,
			},
			"layout/layout": {
				"title": "Test App",
			},
		},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	// React 19 may insert <!-- --> comment nodes between text and interpolated values.
	if !strings.Contains(html, "Hello") || !strings.Contains(html, "World") {
		t.Errorf("expected HTML to contain 'Hello' and 'World', got: %s", html)
	}
	if !strings.Contains(html, "Count:") || !strings.Contains(html, "42") {
		t.Errorf("expected HTML to contain 'Count:' and '42', got: %s", html)
	}
}

func TestRenderWithLayout(t *testing.T) {
	r := startRenderer(t)

	html, err := r.Render(RenderRequest{
		Component: "hello/hello",
		Layout:    "layout/layout",
		ServerData: map[string]map[string]any{
			"hello/hello": {
				"name":  "World",
				"count": 1,
			},
			"layout/layout": {
				"title": "My App",
			},
		},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !strings.Contains(html, "My App") {
		t.Errorf("expected HTML to contain layout title 'My App', got: %s", html)
	}
	if !strings.Contains(html, "<html") {
		t.Errorf("expected HTML to contain <html from layout, got: %s", html)
	}
	if !strings.Contains(html, "<main") {
		t.Errorf("expected HTML to contain <main from layout, got: %s", html)
	}
	// Route content should be nested inside layout
	if !strings.Contains(html, "Hello") || !strings.Contains(html, "World") {
		t.Errorf("expected HTML to contain route content, got: %s", html)
	}
}

func TestRenderMissingComponent(t *testing.T) {
	r := startRenderer(t)

	_, err := r.Render(RenderRequest{
		Component: "nonexistent/component",
		Layout:    "layout/layout",
		ServerData: map[string]map[string]any{
			"layout/layout": {"title": "Test"},
		},
	})
	if err == nil {
		t.Fatal("expected error for missing component")
	}
	if !strings.Contains(err.Error(), "Component not found") {
		t.Errorf("expected 'Component not found' error, got: %v", err)
	}
}

func TestRenderNoViewExport(t *testing.T) {
	r := startRenderer(t)

	_, err := r.Render(RenderRequest{
		Component: "broken/broken",
		Layout:    "layout/layout",
		ServerData: map[string]map[string]any{
			"layout/layout": {"title": "Test"},
		},
	})
	if err == nil {
		t.Fatal("expected error for component without View export")
	}
	if !strings.Contains(err.Error(), "does not export View") {
		t.Errorf("expected 'does not export View' error, got: %v", err)
	}
}

func TestRenderNoServerData(t *testing.T) {
	r := startRenderer(t)

	html, err := r.Render(RenderRequest{
		Component: "hello/hello",
		Layout:    "layout/layout",
	})
	if err != nil {
		t.Fatalf("Render with no server data: %v", err)
	}
	// Should render without error; server data values will be defaults (empty strings, 0)
	if !strings.Contains(html, "Hello") {
		t.Errorf("expected HTML to contain 'Hello', got: %s", html)
	}
}

func TestStopWithoutStart(t *testing.T) {
	r := New()
	if err := r.Stop(); err != nil {
		t.Fatalf("Stop without Start should not error, got: %v", err)
	}
}
