package renderer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rafbgarcia/rstf/internal/bundler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

func repoRootDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filename))
}

func ensureLocalNodeModules(t *testing.T) {
	t.Helper()

	nodeModulesDir := filepath.Join(testdataDir(), "node_modules")
	require.NoError(t, os.MkdirAll(nodeModulesDir, 0755))

	for _, pkg := range []string{"tsx", "react", "react-dom", "scheduler"} {
		linkPath := filepath.Join(nodeModulesDir, pkg)
		if _, err := os.Lstat(linkPath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			require.NoError(t, err)
		}

		target := filepath.Join(repoRootDir(), "node_modules", pkg)
		require.NoError(t, os.Symlink(target, linkPath))
		t.Cleanup(func() {
			_ = os.Remove(linkPath)
		})
	}
}

func startRenderer(t *testing.T) *Renderer {
	t.Helper()
	ensureLocalNodeModules(t)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(testdataDir(), "rstf", "ssr")) })
	require.NoError(t, bundler.BundleSSREntries(testdataDir(), map[string]string{
		"hello/hello": filepath.Join(testdataDir(), "rstf", "ssr_entries", "hello-hello.ssr.tsx"),
	}))
	r := New()
	require.NoError(t, r.Start(testdataDir()))
	t.Cleanup(func() { r.Stop() })
	return r
}

func TestStartStop(t *testing.T) {
	ensureLocalNodeModules(t)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(testdataDir(), "rstf", "ssr")) })
	require.NoError(t, bundler.BundleSSREntries(testdataDir(), map[string]string{
		"hello/hello": filepath.Join(testdataDir(), "rstf", "ssr_entries", "hello-hello.ssr.tsx"),
	}))
	r := New()
	require.NoError(t, r.Start(testdataDir()))
	require.NotNil(t, r.iso)
	require.NoError(t, r.Stop())
}

func TestRenderWithServerData(t *testing.T) {
	r := startRenderer(t)

	html, err := r.Render(RenderRequest{
		Component: "hello/hello",
		Layout:    "layout/layout",
		SSRProps: map[string]map[string]any{
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
		SSRProps: map[string]map[string]any{
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
		SSRProps: map[string]map[string]any{
			"layout/layout": {"title": "Test"},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing SSR bundle")
}

func TestRenderNoViewExport(t *testing.T) {
	r := startRenderer(t)

	_, err := r.Render(RenderRequest{
		Component: "broken/broken",
		Layout:    "layout/layout",
		SSRProps: map[string]map[string]any{
			"layout/layout": {"title": "Test"},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing SSR bundle")
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
