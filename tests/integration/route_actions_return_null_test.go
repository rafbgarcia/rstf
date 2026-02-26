package integration_test

import (
	"net/http"
	"testing"
)

func TestRouteActionsReturnNull(t *testing.T) {
	baseURL := startRouteContractServer(t)

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

	req, err = http.NewRequest(http.MethodGet, baseURL+"/actions-return-null", nil)
	if err != nil {
		t.Fatalf("new request (GET json): %v", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /actions-return-null (json): %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotAcceptable {
		t.Fatalf("GET /actions-return-null (json): got %d, want 406", resp.StatusCode)
	}
}
