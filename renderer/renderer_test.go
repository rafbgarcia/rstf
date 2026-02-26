package renderer

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

func startRenderer(t *testing.T) *Renderer {
	t.Helper()
	r := New()
	require.NoError(t, r.Start(testdataDir()))
	t.Cleanup(func() { r.Stop() })
	return r
}

func TestStartStop(t *testing.T) {
	r := New()
	require.NoError(t, r.Start(testdataDir()))
	require.NotZero(t, r.port)
	require.NoError(t, r.Stop())
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
	require.NoError(t, err)

	// React 19 may insert <!-- --> comment nodes between text and interpolated values.
	assert.Contains(t, html, "Hello")
	assert.Contains(t, html, "World")
	assert.Contains(t, html, "Count:")
	assert.Contains(t, html, "42")
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
	require.NoError(t, err)

	assert.Contains(t, html, "My App")
	assert.Contains(t, html, "<html")
	assert.Contains(t, html, "<main")
	// Route content should be nested inside layout
	assert.Contains(t, html, "Hello")
	assert.Contains(t, html, "World")
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Component not found")
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not export View")
}

func TestRenderNoServerData(t *testing.T) {
	r := startRenderer(t)

	html, err := r.Render(RenderRequest{
		Component: "hello/hello",
		Layout:    "layout/layout",
	})
	require.NoError(t, err)
	// Should render without error; server data values will be defaults (empty strings, 0)
	assert.Contains(t, html, "Hello")
}

func TestStopWithoutStart(t *testing.T) {
	r := New()
	require.NoError(t, r.Stop())
}
