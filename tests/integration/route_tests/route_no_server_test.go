package route_tests

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRouteNoServer(t *testing.T) {
	baseURL := ensureRouteContractServerRunning(t)

	req, err := http.NewRequest(http.MethodGet, baseURL+"/no-server", nil)
	require.NoErrorf(t, err, "new request (GET html)")
	req.Header.Set("Accept", "text/html")
	resp, err := http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "GET /no-server (html)")
	htmlPayload, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "GET /no-server (html)")
	require.Contains(t, resp.Header.Get("Content-Type"), "text/html", "GET /no-server (html)")
	require.Contains(t, string(htmlPayload), "Static route without index.go", "GET /no-server (html)")

	req, err = http.NewRequest(http.MethodGet, baseURL+"/no-server", nil)
	require.NoErrorf(t, err, "new request (GET json)")
	req.Header.Set("Accept", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "GET /no-server (json)")
	_ = resp.Body.Close()
	require.Equal(t, http.StatusNotAcceptable, resp.StatusCode, "GET /no-server (json)")

	req, err = http.NewRequest(http.MethodHead, baseURL+"/no-server", nil)
	require.NoErrorf(t, err, "new request (HEAD html)")
	req.Header.Set("Accept", "text/html")
	resp, err = http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "HEAD /no-server (html)")
	headBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "HEAD /no-server (html)")
	require.Len(t, headBody, 0, "HEAD /no-server (html)")
}
