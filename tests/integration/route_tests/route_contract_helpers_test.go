package route_tests

import (
	"encoding/json"
	"fmt"
	"io"
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
)

var (
	routeServerOnce sync.Once
	routeServerErr  error
	routeServerURL  string
	routeServerCmd  *exec.Cmd
)

func ensureRouteContractServerRunning(t *testing.T) string {
	t.Helper()

	routeServerOnce.Do(func() {
		root := testProjectRoot()

		_, routeServerErr = codegen.Generate(root)
		if routeServerErr != nil {
			return
		}

		build := exec.Command("go", "build", "-o", filepath.Join(root, ".rstf", "server"), "./.rstf/server_gen.go")
		build.Dir = root
		out, err := build.CombinedOutput()
		if err != nil {
			routeServerErr = fmt.Errorf("compiling server: %w\n%s", err, out)
			return
		}

		port := freePort(t)
		routeServerCmd = exec.Command(filepath.Join(root, ".rstf", "server"), "--port", port)
		routeServerCmd.Dir = root
		routeServerCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := routeServerCmd.Start(); err != nil {
			routeServerErr = fmt.Errorf("starting server: %w", err)
			return
		}

		routeServerURL = fmt.Sprintf("http://localhost:%s", port)
		routeServerErr = waitForServerJSON(routeServerURL+"/actions-exhaustive-supported-verbs", 5*time.Second)
	})

	if routeServerErr != nil {
		t.Fatalf("route test server setup failed: %v", routeServerErr)
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
	root := testProjectRoot()
	_ = os.RemoveAll(filepath.Join(root, ".rstf"))
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
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "test_project")
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("finding free port: %v", err)
	}
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
	if err != nil {
		t.Fatalf("new request (%s %s): %v", method, path, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	payload, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s: got %d, want %d (body=%s)", method, path, resp.StatusCode, wantStatus, string(payload))
	}
	var env struct {
		Error struct {
			Code    string         `json:"code"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(payload, &env); err != nil {
		t.Fatalf("%s %s decode envelope: %v\nbody=%s", method, path, err, string(payload))
	}
	if env.Error.Code != wantCode {
		t.Fatalf("%s %s: got code=%q, want %q", method, path, env.Error.Code, wantCode)
	}
	return env.Error.Details
}
