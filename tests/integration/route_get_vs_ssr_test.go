package integration_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRouteGetVsSSR(t *testing.T) {
	baseURL := startRouteContractServer(t)

	req, err := http.NewRequest(http.MethodGet, baseURL+"/get-vs-ssr", nil)
	if err != nil {
		t.Fatalf("new request (GET json): %v", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /get-vs-ssr (json): %v", err)
	}
	payload, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /get-vs-ssr (json): got %d, want 200", resp.StatusCode)
	}
	var getResp struct {
		Source string `json:"source"`
		Route  string `json:"route"`
	}
	if err := json.Unmarshal(payload, &getResp); err != nil {
		t.Fatalf("GET /get-vs-ssr decode: %v\nbody=%s", err, string(payload))
	}
	if getResp.Source != "get" || getResp.Route != "/get-vs-ssr" {
		t.Fatalf("GET /get-vs-ssr unexpected payload: %+v", getResp)
	}

	req, err = http.NewRequest(http.MethodGet, baseURL+"/get-vs-ssr", nil)
	if err != nil {
		t.Fatalf("new request (GET html): %v", err)
	}
	req.Header.Set("Accept", "text/html")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /get-vs-ssr (html): %v", err)
	}
	htmlPayload, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /get-vs-ssr (html): got %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("GET /get-vs-ssr (html): expected text/html content-type, got %q", resp.Header.Get("Content-Type"))
	}
	if !strings.Contains(string(htmlPayload), "<!DOCTYPE html>") {
		t.Fatalf("GET /get-vs-ssr (html): expected HTML document body=%s", string(htmlPayload))
	}

	req, err = http.NewRequest(http.MethodHead, baseURL+"/get-vs-ssr", nil)
	if err != nil {
		t.Fatalf("new request (HEAD json): %v", err)
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
}
