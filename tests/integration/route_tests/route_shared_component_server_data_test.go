package route_tests

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRouteSharedComponentServerData(t *testing.T) {
	baseURL := ensureRouteContractServerRunning(t)

	req, err := http.NewRequest(http.MethodGet, baseURL+"/get-vs-ssr", nil)
	require.NoErrorf(t, err, "new request (GET html)")
	req.Header.Set("Accept", "text/html")

	resp, err := http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "GET /get-vs-ssr (html)")
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "GET /get-vs-ssr (html)")
	require.Contains(t, string(body), `data-testid="user-avatar"`, "shared avatar should render")
	require.Contains(t, string(body), "Ada Lovelace", "shared avatar should receive server data")
	require.Contains(t, string(body), "staff", "shared avatar should receive full server data payload")
}
