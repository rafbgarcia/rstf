package integration_test

import (
	"bytes"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/rafbgarcia/rstf/internal/release"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	cliBinaryOnce sync.Once
	cliBinaryPath string
	cliBinaryErr  error
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func rstfBinary(t *testing.T) string {
	t.Helper()

	cliBinaryOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "rstf-cli-*")
		if err != nil {
			cliBinaryErr = err
			return
		}

		outputPath := filepath.Join(tmpDir, "rstf")
		cmd := exec.Command("go", "build", "-o", outputPath, "./cmd/rstf")
		cmd.Dir = repoRoot(t)
		out, err := cmd.CombinedOutput()
		if err != nil {
			cliBinaryErr = assert.AnError
			cliBinaryPath = string(out)
			return
		}
		cliBinaryPath = outputPath
	})

	require.NoError(t, cliBinaryErr, cliBinaryPath)
	return cliBinaryPath
}

func TestCLIInitScaffoldsApp(t *testing.T) {
	appDir := filepath.Join(t.TempDir(), "sunroom")

	cmd := exec.Command(rstfBinary(t), "init", appDir, "--module", "example.com/sunroom", "--skip-install")
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "rstf init failed:\n%s", out)

	for _, path := range []string{
		"go.mod",
		"package.json",
		"rstf/routes/routes_gen.go",
		"rstf/generated/routes.ts",
		"main.go",
		"main.tsx",
		"main.css",
		"postcss.config.mjs",
		"routes/index/index.go",
		"routes/index/index.tsx",
		"routes/live-chat._id/index.go",
		"routes/live-chat._id/index.tsx",
		"routes/users._id/index.go",
		"routes/users._id/index.tsx",
		"shared/ui/app-badge/index.go",
		"shared/ui/app-badge/index.tsx",
	} {
		_, err := os.Stat(filepath.Join(appDir, path))
		require.NoErrorf(t, err, "expected %s to exist", path)
	}

	goMod, err := os.ReadFile(filepath.Join(appDir, "go.mod"))
	require.NoError(t, err)
	assert.Contains(t, string(goMod), "module example.com/sunroom")
	assert.Contains(t, string(goMod), "require github.com/rafbgarcia/rstf "+release.ModuleVersion)
	assert.NotContains(t, string(goMod), "replace github.com/rafbgarcia/rstf =>")

	packageJSON, err := os.ReadFile(filepath.Join(appDir, "package.json"))
	require.NoError(t, err)
	assert.Contains(t, string(packageJSON), "\"tailwindcss\"")
	assert.Contains(t, string(packageJSON), "\"@rstf/cli\": \""+release.Version+"\"")
	assert.Contains(t, string(packageJSON), "\"build\": \"rstf build\"")
}

func TestCLIBuildProducesRunnableDist(t *testing.T) {
	appDir := filepath.Join(t.TempDir(), "sunroom")

	initCmd := exec.Command(rstfBinary(t), "init", appDir, "--module", "example.com/sunroom", "--skip-install")
	initCmd.Dir = repoRoot(t)
	initCmd.Env = scaffoldLocalReleaseEnv(t)
	initOut, err := initCmd.CombinedOutput()
	require.NoErrorf(t, err, "rstf init failed:\n%s", initOut)

	npmInstall := exec.Command("npm", "install")
	npmInstall.Dir = appDir
	npmInstall.Env = append(os.Environ(), "NO_UPDATE_NOTIFIER=1", "npm_config_fund=false", "npm_config_audit=false")
	npmInstall.Stdout = os.Stdout
	npmInstall.Stderr = os.Stderr
	require.NoError(t, npmInstall.Run())

	goModTidy := exec.Command("go", "mod", "tidy")
	goModTidy.Dir = appDir
	goModTidy.Stdout = os.Stdout
	goModTidy.Stderr = os.Stderr
	require.NoError(t, goModTidy.Run())

	buildCmd := exec.Command("npm", "run", "build")
	buildCmd.Dir = appDir
	buildCmd.Env = append(os.Environ(), "RSTF_CLI_LOCAL_BINARY="+rstfBinary(t))
	buildOut, err := buildCmd.CombinedOutput()
	require.NoErrorf(t, err, "rstf build failed:\n%s", buildOut)

	distDir := filepath.Join(appDir, "dist")
	for _, path := range []string{
		"sunroom",
		"rstf/generated/routes.ts",
		"rstf/ssr/index.js",
		"rstf/static/main.css",
	} {
		_, err := os.Stat(filepath.Join(distDir, path))
		require.NoErrorf(t, err, "expected dist asset %s to exist", path)
	}

	port := freePort(t)
	serverCmd := exec.Command(filepath.Join(distDir, "sunroom"), "--port", port)
	serverCmd.Dir = distDir
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr
	serverCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	require.NoError(t, serverCmd.Start())
	t.Cleanup(func() {
		stopProcessGroup(t, serverCmd, 1*time.Second)
	})

	baseURL := "http://localhost:" + port
	waitForServer(t, baseURL+"/", 15*time.Second)

	resp, err := http.Get(baseURL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	body := new(bytes.Buffer)
	_, err = body.ReadFrom(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body.String(), "Sunroom")
	assert.Contains(t, strings.ToLower(body.String()), "live queries")

	liveResp, err := http.Get(baseURL + "/live-chat/studio")
	require.NoError(t, err)
	defer liveResp.Body.Close()

	liveBody := new(bytes.Buffer)
	_, err = liveBody.ReadFrom(liveResp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, liveResp.StatusCode)
	assert.Contains(t, liveBody.String(), "Studio room")
	assert.Contains(t, strings.ToLower(liveBody.String()), "live query")
}
