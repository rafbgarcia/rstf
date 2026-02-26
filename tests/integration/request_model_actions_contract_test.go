package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	rstf "github.com/rafbgarcia/rstf"
	"github.com/rafbgarcia/rstf/internal/codegen"
)

func TestRequestModelActionsContract(t *testing.T) {
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
		_ = server.Process.Signal(syscall.SIGINT)
		_ = server.Wait()
	})

	baseURL := fmt.Sprintf("http://localhost:%s", port)
	waitForServer(t, baseURL+"/get-vs-ssr", 30*time.Second)

	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/actions-return-null", nil)
	if err != nil {
		t.Fatalf("new request (POST): %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /actions-return-null: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("POST /actions-return-null: got %d, want 204", resp.StatusCode)
	}

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		req, err := http.NewRequest(method, baseURL+"/actions-exhaustive-supported-verbs", nil)
		if err != nil {
			t.Fatalf("new request (%s): %v", method, err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s /actions-exhaustive-supported-verbs: %v", method, err)
		}
		payload, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s /actions-exhaustive-supported-verbs: got %d, want 200", method, resp.StatusCode)
		}
		var body struct {
			Method string `json:"method"`
		}
		if err := json.Unmarshal(payload, &body); err != nil {
			t.Fatalf("%s /actions-exhaustive-supported-verbs decode: %v", method, err)
		}
		if body.Method != method {
			t.Fatalf("%s /actions-exhaustive-supported-verbs: got method=%q", method, body.Method)
		}
	}

	req, err = http.NewRequest(http.MethodGet, baseURL+"/actions-exhaustive-supported-verbs", nil)
	if err != nil {
		t.Fatalf("new request (GET /actions-exhaustive-supported-verbs): %v", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /actions-exhaustive-supported-verbs: %v", err)
	}
	payload, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /actions-exhaustive-supported-verbs: got %d, want 200", resp.StatusCode)
	}
	var getBody struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal(payload, &getBody); err != nil {
		t.Fatalf("GET /actions-exhaustive-supported-verbs decode: %v", err)
	}
	if getBody.Method != http.MethodGet {
		t.Fatalf("GET /actions-exhaustive-supported-verbs: got method=%q", getBody.Method)
	}

	reqBody := strings.NewReader(`{"title":"hello"}`)
	req, err = http.NewRequest(http.MethodPost, baseURL+"/actions-return-json", reqBody)
	if err != nil {
		t.Fatalf("new request (POST /actions-return-json): %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /actions-return-json: %v", err)
	}
	jsonPayload, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /actions-return-json: got %d, want 201", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		t.Fatalf("POST /actions-return-json: expected application/json content-type, got %q", resp.Header.Get("Content-Type"))
	}
	var created struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(jsonPayload, &created); err != nil {
		t.Fatalf("POST /actions-return-json decode: %v\nbody=%s", err, string(jsonPayload))
	}
	if created.ID != "post_123" || created.Status != "created" {
		t.Fatalf("POST /actions-return-json unexpected payload: %+v", created)
	}

	req, err = http.NewRequest(http.MethodPost, baseURL+"/actions-redirect", nil)
	if err != nil {
		t.Fatalf("new request (POST /actions-redirect): %v", err)
	}
	resp, err = noRedirectClient.Do(req)
	if err != nil {
		t.Fatalf("POST /actions-redirect: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST /actions-redirect: got %d, want 303", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/get-vs-ssr" {
		t.Fatalf("POST /actions-redirect: got Location=%q, want /get-vs-ssr", got)
	}

	req, err = http.NewRequest(http.MethodGet, baseURL+"/get-vs-ssr", nil)
	if err != nil {
		t.Fatalf("new request (GET /get-vs-ssr): %v", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /get-vs-ssr (json): %v", err)
	}
	jsonPayload, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /get-vs-ssr (json): got %d, want 200", resp.StatusCode)
	}
	var getResp struct {
		Source string `json:"source"`
		Route  string `json:"route"`
	}
	if err := json.Unmarshal(jsonPayload, &getResp); err != nil {
		t.Fatalf("GET /get-vs-ssr decode: %v\nbody=%s", err, string(jsonPayload))
	}
	if getResp.Source != "get" || getResp.Route != "/get-vs-ssr" {
		t.Fatalf("GET /get-vs-ssr unexpected payload: %+v", getResp)
	}

	assertEnvelope := func(t *testing.T, method, path string, body io.Reader, contentType string, wantStatus int, wantCode string) map[string]any {
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

	assertEnvelope(t,
		http.MethodPost,
		"/actions-return-json",
		strings.NewReader("title=plain"),
		"text/plain",
		http.StatusUnsupportedMediaType,
		string(rstf.ErrorCodeUnsupportedContentType),
	)

	assertEnvelope(t,
		http.MethodPost,
		"/actions-return-json",
		strings.NewReader(`{"title":`),
		"application/json",
		http.StatusBadRequest,
		string(rstf.ErrorCodeInvalidPayload),
	)

	huge := `{"title":"` + strings.Repeat("a", 2048) + `"}`
	details := assertEnvelope(t,
		http.MethodPost,
		"/actions-return-json",
		bytes.NewBufferString(huge),
		"application/json",
		http.StatusRequestEntityTooLarge,
		string(rstf.ErrorCodePayloadTooLarge),
	)
	if got := details["limitBytes"]; got != float64(1024) {
		t.Fatalf("POST /actions-return-json payload_too_large: got limitBytes=%v, want 1024", got)
	}

	req, err = http.NewRequest(http.MethodHead, baseURL+"/get-vs-ssr", nil)
	if err != nil {
		t.Fatalf("new request (HEAD /get-vs-ssr): %v", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HEAD /get-vs-ssr: %v", err)
	}
	headBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("HEAD /get-vs-ssr: got %d, want 200", resp.StatusCode)
	}
	if len(headBody) != 0 {
		t.Fatalf("HEAD /get-vs-ssr: expected empty body, got %q", string(headBody))
	}

	req, err = http.NewRequest(http.MethodOptions, baseURL+"/actions-exhaustive-supported-verbs", nil)
	if err != nil {
		t.Fatalf("new request (OPTIONS /actions-exhaustive-supported-verbs): %v", err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /actions-exhaustive-supported-verbs: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("OPTIONS /actions-exhaustive-supported-verbs: got %d, want 204", resp.StatusCode)
	}
	allow := resp.Header.Get("Allow")
	for _, method := range []string{"OPTIONS", "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"} {
		if !strings.Contains(allow, method) {
			t.Fatalf("OPTIONS /actions-exhaustive-supported-verbs: Allow header missing %q (got %q)", method, allow)
		}
	}
}
