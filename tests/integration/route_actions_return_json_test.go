package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	rstf "github.com/rafbgarcia/rstf"
)

func TestRouteActionsReturnJSON(t *testing.T) {
	baseURL := startRouteContractServer(t)

	reqBody := strings.NewReader(`{"title":"hello"}`)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/actions-return-json", reqBody)
	if err != nil {
		t.Fatalf("new request (POST success): %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /actions-return-json: %v", err)
	}
	payload, _ := io.ReadAll(resp.Body)
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
	if err := json.Unmarshal(payload, &created); err != nil {
		t.Fatalf("POST /actions-return-json decode: %v\nbody=%s", err, string(payload))
	}
	if created.ID != "post_123" || created.Status != "created" {
		t.Fatalf("POST /actions-return-json unexpected payload: %+v", created)
	}

	assertErrorEnvelope(
		t,
		baseURL,
		http.MethodPost,
		"/actions-return-json",
		strings.NewReader("title=plain"),
		"text/plain",
		http.StatusUnsupportedMediaType,
		string(rstf.ErrorCodeUnsupportedContentType),
	)

	assertErrorEnvelope(
		t,
		baseURL,
		http.MethodPost,
		"/actions-return-json",
		strings.NewReader(`{"title":`),
		"application/json",
		http.StatusBadRequest,
		string(rstf.ErrorCodeInvalidPayload),
	)

	huge := `{"title":"` + strings.Repeat("a", 2048) + `"}`
	details := assertErrorEnvelope(
		t,
		baseURL,
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

	assertErrorEnvelope(
		t,
		baseURL,
		http.MethodPost,
		"/actions-return-json",
		strings.NewReader(`{"title":""}`),
		"application/json",
		http.StatusUnprocessableEntity,
		string(rstf.ErrorCodeValidationFailed),
	)
}
