package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverTSXRouteDirs(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "routes", "index", "index.tsx"), `export function View() { return <div />; }`)
	writeFile(t, filepath.Join(root, "routes", "admin.users", "index.tsx"), `export function View() { return <div />; }`)

	got, err := discoverTSXRouteDirs(root)
	require.NoError(t, err)
	assert.Equal(t, []string{"routes/admin.users", "routes/index"}, got)
}

func TestDiscoverTSXRouteDirsRejectsNestedRouteDirectories(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "routes", "admin", "users", "index.tsx"), `export function View() { return <div />; }`)

	_, err := discoverTSXRouteDirs(root)
	require.Error(t, err)
	require.ErrorContains(t, err, `invalid route directory "routes/admin/users"`)
	require.ErrorContains(t, err, "use dotted names like routes/admin.users")
}
