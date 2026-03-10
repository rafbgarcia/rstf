package route_tests

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRouteDynamicParams(t *testing.T) {
	baseURL := ensureRouteContractServerRunning(t)

	req, err := http.NewRequest(http.MethodGet, baseURL+"/users/123", nil)
	require.NoErrorf(t, err, "new request (GET /users/123)")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "GET /users/123")
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "GET /users/123")

	var payload struct {
		ID    string `json:"id"`
		Route string `json:"route"`
	}
	require.NoErrorf(t, json.Unmarshal(body, &payload), "decode /users/123 body=%s", string(body))
	require.Equal(t, "123", payload.ID, "GET /users/123 id")
	require.Equal(t, "/users/123", payload.Route, "GET /users/123 route")
}
