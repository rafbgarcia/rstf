package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rafbgarcia/rstf/internal/codegen"
	"github.com/stretchr/testify/require"
)

func TestTypeSafeRoutesTypeScript(t *testing.T) {
	root := testProjectRoot()

	_, err := codegen.Generate(root)
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(filepath.Join(root, "rstf")) })

	tscPath, ok := lookupTSC()
	if !ok {
		t.Skip("tsc not available")
	}

	configPath := filepath.Join(root, "tsconfig.routes.json")
	err = os.WriteFile(configPath, []byte(`{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "node",
    "jsx": "react-jsx",
    "strict": true,
    "noEmit": true,
    "baseUrl": ".",
    "types": [],
    "paths": {
      "@rstf/*": ["./rstf/generated/*"]
    }
  },
  "include": [
    "routes/typesafe-routes.ts",
    "routes/typesafe-rpc.ts",
    "rstf/generated/client.ts",
    "rstf/generated/routes.ts",
    "rstf/types/**/*.d.ts"
  ]
}
`), 0644)
	require.NoError(t, err)

	cmd := exec.Command(tscPath, "-p", configPath)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "tsc failed:\n%s", out)
}

func lookupTSC() (string, bool) {
	_, filename, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	candidates := []string{
		filepath.Join(repoRoot, "node_modules", ".bin", "tsc"),
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
	}

	if path, err := exec.LookPath("tsc"); err == nil {
		return path, true
	}
	return "", false
}
