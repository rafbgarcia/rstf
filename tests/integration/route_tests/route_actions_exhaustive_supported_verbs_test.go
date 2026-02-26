package route_tests

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouteActionsExhaustiveSupportedVerbs(t *testing.T) {
	baseURL := ensureRouteContractServerRunning(t)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		req, err := http.NewRequest(method, baseURL+"/actions-exhaustive-supported-verbs", nil)
		require.NoErrorf(t, err, "new request (%s)", method)
		resp, err := http.DefaultClient.Do(req)
		require.NoErrorf(t, err, "%s /actions-exhaustive-supported-verbs", method)
		payload, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		require.Equalf(t, http.StatusOK, resp.StatusCode, "%s /actions-exhaustive-supported-verbs", method)
		var body struct {
			Method string `json:"method"`
		}
		require.NoErrorf(t, json.Unmarshal(payload, &body), "%s /actions-exhaustive-supported-verbs decode", method)
		require.Equalf(t, method, body.Method, "%s /actions-exhaustive-supported-verbs method", method)
	}

	req, err := http.NewRequest(http.MethodGet, baseURL+"/actions-exhaustive-supported-verbs", nil)
	require.NoErrorf(t, err, "new request (GET json)")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "GET /actions-exhaustive-supported-verbs (json)")
	payload, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "GET /actions-exhaustive-supported-verbs (json)")
	var getBody struct {
		Method string `json:"method"`
	}
	require.NoErrorf(t, json.Unmarshal(payload, &getBody), "GET /actions-exhaustive-supported-verbs decode")
	require.Equal(t, http.MethodGet, getBody.Method, "GET /actions-exhaustive-supported-verbs method")

	req, err = http.NewRequest(http.MethodGet, baseURL+"/actions-exhaustive-supported-verbs", nil)
	require.NoErrorf(t, err, "new request (GET html)")
	req.Header.Set("Accept", "text/html")
	resp, err = http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "GET /actions-exhaustive-supported-verbs (html)")
	_ = resp.Body.Close()
	require.Equal(t, http.StatusNotAcceptable, resp.StatusCode, "GET /actions-exhaustive-supported-verbs (html)")

	req, err = http.NewRequest(http.MethodOptions, baseURL+"/actions-exhaustive-supported-verbs", nil)
	require.NoErrorf(t, err, "new request (OPTIONS)")
	resp, err = http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "OPTIONS /actions-exhaustive-supported-verbs")
	_ = resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode, "OPTIONS /actions-exhaustive-supported-verbs")
	allow := resp.Header.Get("Allow")
	for _, method := range []string{"OPTIONS", "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"} {
		assert.Containsf(t, allow, method, "OPTIONS /actions-exhaustive-supported-verbs: Allow header missing %q", method)
	}
}
