package route_tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	rstf "github.com/rafbgarcia/rstf"
	"github.com/stretchr/testify/require"
)

func TestRouteActionsReturnJSON(t *testing.T) {
	baseURL := ensureRouteContractServerRunning(t)

	reqBody := strings.NewReader(`{"title":"hello"}`)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/actions-return-json", reqBody)
	require.NoErrorf(t, err, "new request (POST success)")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "POST /actions-return-json")
	payload, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode, "POST /actions-return-json")
	require.Contains(t, resp.Header.Get("Content-Type"), "application/json", "POST /actions-return-json")
	var created struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	require.NoErrorf(t, json.Unmarshal(payload, &created), "POST /actions-return-json decode: body=%s", string(payload))
	require.Equal(t, "post_123", created.ID, "POST /actions-return-json id")
	require.Equal(t, "created", created.Status, "POST /actions-return-json status")

	assertErrorEnvelope(
		t,
		baseURL,
		http.MethodPost,
		"/actions-return-json",
		strings.NewReader("title=plain"),
		"text/plain",
		http.StatusUnsupportedMediaType,
		string(rstf.ErrorCodeUnsupportedContentType),
	)

	assertErrorEnvelope(
		t,
		baseURL,
		http.MethodPost,
		"/actions-return-json",
		strings.NewReader(`{"title":`),
		"application/json",
		http.StatusBadRequest,
		string(rstf.ErrorCodeInvalidPayload),
	)

	huge := `{"title":"` + strings.Repeat("a", 2048) + `"}`
	details := assertErrorEnvelope(
		t,
		baseURL,
		http.MethodPost,
		"/actions-return-json",
		bytes.NewBufferString(huge),
		"application/json",
		http.StatusRequestEntityTooLarge,
		string(rstf.ErrorCodePayloadTooLarge),
	)
	require.Equal(t, float64(1024), details["limitBytes"], "POST /actions-return-json payload_too_large")

	assertErrorEnvelope(
		t,
		baseURL,
		http.MethodPost,
		"/actions-return-json",
		strings.NewReader(`{"title":""}`),
		"application/json",
		http.StatusUnprocessableEntity,
		string(rstf.ErrorCodeValidationFailed),
	)
}
