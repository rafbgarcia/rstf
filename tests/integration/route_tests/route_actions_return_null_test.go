package route_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRouteActionsReturnNull(t *testing.T) {
	baseURL := ensureRouteContractServerRunning(t)

	req, err := http.NewRequest(http.MethodPost, baseURL+"/actions-return-null", nil)
	require.NoErrorf(t, err, "new request (POST)")
	resp, err := http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "POST /actions-return-null")
	_ = resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode, "POST /actions-return-null")

	req, err = http.NewRequest(http.MethodGet, baseURL+"/actions-return-null", nil)
	require.NoErrorf(t, err, "new request (GET json)")
	req.Header.Set("Accept", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "GET /actions-return-null (json)")
	_ = resp.Body.Close()
	require.Equal(t, http.StatusNotAcceptable, resp.StatusCode, "GET /actions-return-null (json)")
}
