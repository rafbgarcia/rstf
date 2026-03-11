package renderer

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/rafbgarcia/rstf/internal/bundler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

var (
	rendererDepsOnce sync.Once
	rendererDepsErr  error
)

func ensureLocalNodeModules(t *testing.T) {
	t.Helper()

	nodeModulesDir := filepath.Join(testdataDir(), "node_modules")
	require.NoError(t, os.MkdirAll(nodeModulesDir, 0755))

	rendererDepsOnce.Do(func() {
		reactPath := filepath.Join(nodeModulesDir, "react")
		if _, err := os.Stat(reactPath); os.IsNotExist(err) {
			cmd := exec.Command("npm", "install")
			cmd.Dir = testdataDir()
			cmd.Env = append(
				os.Environ(),
				"NO_UPDATE_NOTIFIER=1",
				"npm_config_fund=false",
				"npm_config_audit=false",
			)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			rendererDepsErr = cmd.Run()
			if rendererDepsErr != nil {
				return
			}
		} else if err != nil {
			rendererDepsErr = err
			return
		}

		rstfLinkPath := filepath.Join(nodeModulesDir, "@rstf")
		if err := os.Remove(rstfLinkPath); err != nil && !os.IsNotExist(err) {
			rendererDepsErr = err
			return
		}
		rendererDepsErr = os.Symlink(filepath.Join("..", ".rstf", "generated"), rstfLinkPath)
	})

	require.NoError(t, rendererDepsErr)
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
