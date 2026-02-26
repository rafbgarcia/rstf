package route_tests

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRouteGetVsSSR(t *testing.T) {
	baseURL := ensureRouteContractServerRunning(t)

	req, err := http.NewRequest(http.MethodGet, baseURL+"/get-vs-ssr", nil)
	require.NoErrorf(t, err, "new request (GET json)")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "GET /get-vs-ssr (json)")
	payload, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "GET /get-vs-ssr (json)")
	var getResp struct {
		Source string `json:"source"`
		Route  string `json:"route"`
	}
	require.NoErrorf(t, json.Unmarshal(payload, &getResp), "GET /get-vs-ssr decode: body=%s", string(payload))
	require.Equal(t, "get", getResp.Source, "GET /get-vs-ssr source")
	require.Equal(t, "/get-vs-ssr", getResp.Route, "GET /get-vs-ssr route")

	req, err = http.NewRequest(http.MethodGet, baseURL+"/get-vs-ssr", nil)
	require.NoErrorf(t, err, "new request (GET html)")
	req.Header.Set("Accept", "text/html")
	resp, err = http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "GET /get-vs-ssr (html)")
	htmlPayload, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "GET /get-vs-ssr (html)")
	require.Contains(t, resp.Header.Get("Content-Type"), "text/html", "GET /get-vs-ssr (html)")
	require.Contains(t, string(htmlPayload), "<!DOCTYPE html>", "GET /get-vs-ssr (html)")

	req, err = http.NewRequest(http.MethodHead, baseURL+"/get-vs-ssr", nil)
	require.NoErrorf(t, err, "new request (HEAD json)")
	req.Header.Set("Accept", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "HEAD /get-vs-ssr")
	headBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "HEAD /get-vs-ssr")
	require.Len(t, headBody, 0, "HEAD /get-vs-ssr body")
}
