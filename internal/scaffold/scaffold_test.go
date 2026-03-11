package scaffold

import (
	"path/filepath"
	"testing"

	"github.com/rafbgarcia/rstf/internal/release"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveConfigUsesReleaseDefaults(t *testing.T) {
	cfg, err := DeriveConfig("sunroom", "")
	require.NoError(t, err)

	assert.Equal(t, "sunroom", cfg.Name)
	assert.Equal(t, "sunroom", cfg.Module)
	assert.Equal(t, release.FrameworkModule, cfg.FrameworkModule)
	assert.Equal(t, release.ModuleVersion, cfg.FrameworkVersion)
	assert.Equal(t, release.CLIPackage, cfg.CLIPackage)
	assert.Equal(t, release.Version, cfg.CLIRef)
	assert.Empty(t, cfg.FrameworkReplace)
}

func TestDeriveConfigHonorsLocalOverrides(t *testing.T) {
	t.Setenv("RSTF_SCAFFOLD_FRAMEWORK_REPLACE", "/tmp/rstf")
	t.Setenv("RSTF_SCAFFOLD_CLI_PACKAGE", "file:/tmp/rstf/packages/cli")

	cfg, err := DeriveConfig("sunroom", "example.com/sunroom")
	require.NoError(t, err)

	assert.Equal(t, "example.com/sunroom", cfg.Module)
	assert.Equal(t, filepath.Clean("/tmp/rstf"), filepath.Clean(cfg.FrameworkReplace))
	assert.Equal(t, "file:/tmp/rstf/packages/cli", cfg.CLIRef)
}
