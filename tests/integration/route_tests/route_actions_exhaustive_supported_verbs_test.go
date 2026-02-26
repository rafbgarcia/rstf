package route_tests

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRouteActionsExhaustiveSupportedVerbs(t *testing.T) {
	baseURL := ensureRouteContractServerRunning(t)

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

	req, err := http.NewRequest(http.MethodGet, baseURL+"/actions-exhaustive-supported-verbs", nil)
	if err != nil {
		t.Fatalf("new request (GET json): %v", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /actions-exhaustive-supported-verbs (json): %v", err)
	}
	payload, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /actions-exhaustive-supported-verbs (json): got %d, want 200", resp.StatusCode)
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

	req, err = http.NewRequest(http.MethodGet, baseURL+"/actions-exhaustive-supported-verbs", nil)
	if err != nil {
		t.Fatalf("new request (GET html): %v", err)
	}
	req.Header.Set("Accept", "text/html")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /actions-exhaustive-supported-verbs (html): %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotAcceptable {
		t.Fatalf("GET /actions-exhaustive-supported-verbs (html): got %d, want 406", resp.StatusCode)
	}

	req, err = http.NewRequest(http.MethodOptions, baseURL+"/actions-exhaustive-supported-verbs", nil)
	if err != nil {
		t.Fatalf("new request (OPTIONS): %v", err)
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
