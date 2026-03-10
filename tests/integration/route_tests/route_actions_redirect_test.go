package route_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRouteActionsRedirect(t *testing.T) {
	baseURL := ensureRouteContractServerRunning(t)

	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/actions-redirect", nil)
	require.NoErrorf(t, err, "new request (POST)")
	resp, err := noRedirectClient.Do(req)
	require.NoErrorf(t, err, "POST /actions-redirect")
	_ = resp.Body.Close()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode, "POST /actions-redirect")
	require.Equal(t, "/users/123", resp.Header.Get("Location"), "POST /actions-redirect location")
}
