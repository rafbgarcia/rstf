package route_tests

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	rstf "github.com/rafbgarcia/rstf"
	"github.com/stretchr/testify/require"
)

func TestRouteAdmissionSlow(t *testing.T) {
	baseURL := ensureRouteContractServerRunning(t)

	type overloadedResponse struct {
		status int
		code   string
		reason string
	}
	results := make(chan overloadedResponse, 2)
	var wg sync.WaitGroup
	sendSlow := func() {
		defer wg.Done()
		req, err := http.NewRequest(http.MethodGet, baseURL+"/admission-slow", nil)
		if err != nil {
			t.Errorf("new request (GET /admission-slow): %v", err)
			return
		}
		req.Header.Set("Accept", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("GET /admission-slow: %v", err)
			return
		}
		defer resp.Body.Close()
		payload, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusServiceUnavailable {
			return
		}
		var env struct {
			Error struct {
				Code    string         `json:"code"`
				Details map[string]any `json:"details"`
			} `json:"error"`
		}
		if err := json.Unmarshal(payload, &env); err != nil {
			t.Errorf("decode overload envelope: %v", err)
			return
		}
		reason, _ := env.Error.Details["reason"].(string)
		results <- overloadedResponse{
			status: resp.StatusCode,
			code:   env.Error.Code,
			reason: reason,
		}
	}

	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		req, err := http.NewRequest(http.MethodGet, baseURL+"/admission-slow", nil)
		if err != nil {
			t.Errorf("new request (first /admission-slow): %v", err)
			return
		}
		req.Header.Set("Accept", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("first /admission-slow: %v", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("first /admission-slow: got %d want 200 body=%s", resp.StatusCode, string(body))
		}
	}()
	time.Sleep(20 * time.Millisecond)

	wg.Add(2)
	go sendSlow()
	go sendSlow()
	wg.Wait()
	<-firstDone
	close(results)

	require.Equal(t, 2, len(results), "expected 2 overload responses")
	reasons := map[string]int{}
	for r := range results {
		require.Equal(t, http.StatusServiceUnavailable, r.status, "overload status")
		require.Equal(t, string(rstf.ErrorCodeOverloaded), r.code, "overload code")
		reasons[r.reason]++
	}
	require.Equal(t, 1, reasons["queue_full"], "overload reason queue_full")
	require.Equal(t, 1, reasons["queue_timeout"], "overload reason queue_timeout")
}
