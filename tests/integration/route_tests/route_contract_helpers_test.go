package route_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/rafbgarcia/rstf/internal/codegen"
	"github.com/stretchr/testify/require"
)

var (
	routeServerOnce sync.Once
	routeServerErr  error
	routeServerURL  string
	routeServerCmd  *exec.Cmd

	testProjectRootOnce sync.Once
	testProjectRootDir  string
	testProjectRootErr  error
)

func ensureRouteContractServerRunning(t *testing.T) string {
	t.Helper()

	routeServerOnce.Do(func() {
		root := testProjectRoot()

		_, routeServerErr = codegen.Generate(root)
		if routeServerErr != nil {
			return
		}

		build := exec.Command("go", "build", "-o", filepath.Join(root, "rstf", "server"), "./rstf/server_gen.go")
		build.Dir = root
		out, err := build.CombinedOutput()
		if err != nil {
			routeServerErr = fmt.Errorf("compiling server: %w\n%s", err, out)
			return
		}

		port := freePort(t)
		routeServerCmd = exec.Command(filepath.Join(root, "rstf", "server"), "--port", port)
		routeServerCmd.Dir = root
		routeServerCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := routeServerCmd.Start(); err != nil {
			routeServerErr = fmt.Errorf("starting server: %w", err)
			return
		}

		routeServerURL = fmt.Sprintf("http://localhost:%s", port)
		routeServerErr = waitForServerJSON(routeServerURL+"/actions-exhaustive-supported-verbs", 10*time.Second)
	})

	if routeServerErr != nil {
		require.NoErrorf(t, routeServerErr, "route test server setup failed")
	}
	return routeServerURL
}

func TestMain(m *testing.M) {
	code := m.Run()
	stopRouteContractServer()
	os.Exit(code)
}

func stopRouteContractServer() {
	if routeServerCmd != nil {
		stopProcessGroup(routeServerCmd, 1*time.Second)
	}
	if testProjectRootDir != "" {
		_ = os.RemoveAll(testProjectRootDir)
	}
}

func waitForServerJSON(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("new request for readiness: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("server at %s not ready after %s", url, timeout)
}

func testProjectRoot() string {
	testProjectRootOnce.Do(func() {
		testProjectRootDir, testProjectRootErr = cloneTestProject(testProjectFixtureRoot())
	})
	if testProjectRootErr != nil {
		panic(fmt.Sprintf("cloning test project fixture: %v", testProjectRootErr))
	}
	return testProjectRootDir
}

func testProjectFixtureRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "test_project")
}

func cloneTestProject(src string) (string, error) {
	dst, err := os.MkdirTemp(filepath.Dir(src), ".tmp-rstf-route-tests-*")
	if err != nil {
		return "", err
	}
	if err := copyDir(src, dst); err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}
	if err := installTestProjectDependencies(dst); err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}
	return dst, nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		base := filepath.Base(rel)
		if base == "node_modules" || base == ".rstf" || base == "rstf" || base == "dist" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if d.Type()&fs.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(linkTarget, target)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func installTestProjectDependencies(dir string) error {
	cmd := exec.Command("npm", "install")
	cmd.Dir = dir
	cmd.Env = append(
		os.Environ(),
		"NO_UPDATE_NOTIFIER=1",
		"npm_config_fund=false",
		"npm_config_audit=false",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoErrorf(t, err, "finding free port")
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return fmt.Sprintf("%d", port)
}

func stopProcessGroup(cmd *exec.Cmd, grace time.Duration) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	pid := cmd.Process.Pid
	_ = syscall.Kill(-pid, syscall.SIGINT)

	waitCh := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		return
	case <-time.After(grace):
	}

	_ = syscall.Kill(-pid, syscall.SIGKILL)
	select {
	case <-waitCh:
	case <-time.After(1 * time.Second):
	}
}

func assertErrorEnvelope(
	t *testing.T,
	baseURL string,
	method string,
	path string,
	body io.Reader,
	contentType string,
	wantStatus int,
	wantCode string,
) map[string]any {
	t.Helper()
	req, err := http.NewRequest(method, baseURL+path, body)
	require.NoErrorf(t, err, "new request (%s %s)", method, path)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "%s %s", method, path)
	payload, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equalf(t, wantStatus, resp.StatusCode, "%s %s: body=%s", method, path, string(payload))
	var env struct {
		Error struct {
			Code    string         `json:"code"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	require.NoErrorf(t, json.Unmarshal(payload, &env), "%s %s decode envelope: body=%s", method, path, string(payload))
	require.Equalf(t, wantCode, env.Error.Code, "%s %s", method, path)
	return env.Error.Details
}
