package integration_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/rafbgarcia/rstf/internal/codegen"
)

func startRouteContractServer(t *testing.T) string {
	t.Helper()

	root := testProjectRoot()

	_, err := codegen.Generate(root)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Join(root, ".rstf")) })

	build := exec.Command("go", "build", "-o", filepath.Join(root, ".rstf", "server"), "./.rstf/server_gen.go")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compiling server: %v\n%s", err, out)
	}

	port := freePort(t)
	server := exec.Command(filepath.Join(root, ".rstf", "server"), "--port", port)
	server.Dir = root
	server.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := server.Start(); err != nil {
		t.Fatalf("starting server: %v", err)
	}
	t.Cleanup(func() {
		stopProcessGroup(t, server, 1*time.Second)
	})

	baseURL := fmt.Sprintf("http://localhost:%s", port)
	waitForServerJSON(t, baseURL+"/actions-exhaustive-supported-verbs", 5*time.Second)
	return baseURL
}

func waitForServerJSON(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("new request for readiness: %v", err)
		}
		req.Header.Set("Accept", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server at %s not ready after %s", url, timeout)
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
