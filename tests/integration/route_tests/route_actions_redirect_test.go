package route_tests

import (
	"net/http"
	"testing"
)

func TestRouteActionsRedirect(t *testing.T) {
	baseURL := ensureRouteContractServerRunning(t)

	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/actions-redirect", nil)
	if err != nil {
		t.Fatalf("new request (POST): %v", err)
	}
	resp, err := noRedirectClient.Do(req)
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
}
