package route_tests

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	rstf "github.com/rafbgarcia/rstf"
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

	if len(results) != 2 {
		t.Fatalf("expected 2 overload responses, got %d", len(results))
	}
	reasons := map[string]int{}
	for r := range results {
		if r.status != http.StatusServiceUnavailable {
			t.Fatalf("overload status = %d, want 503", r.status)
		}
		if r.code != string(rstf.ErrorCodeOverloaded) {
			t.Fatalf("overload code = %q, want %q", r.code, rstf.ErrorCodeOverloaded)
		}
		reasons[r.reason]++
	}
	if reasons["queue_full"] != 1 || reasons["queue_timeout"] != 1 {
		t.Fatalf("expected one queue_full and one queue_timeout; got %+v", reasons)
	}
}
