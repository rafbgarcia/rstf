package rstf

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAdmissionMiddleware_DropsWhenQueueFull(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		w.WriteHeader(http.StatusNoContent)
	})

	mw := NewAdmissionMiddleware(AdmissionControlConfig{
		MaxConcurrentRequests: 1,
		MaxQueuedRequests:     1,
		QueueTimeout:          time.Second,
	})
	h := mw(handler)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		req := httptest.NewRequest(http.MethodGet, "/slow", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}()

	<-started

	wg.Add(1)
	go func() {
		defer wg.Done()
		req := httptest.NewRequest(http.MethodGet, "/slow", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}()

	time.Sleep(20 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	errBody, ok := envelope["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, string(ErrorCodeOverloaded), errBody["code"])
	details, ok := errBody["details"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "queue_full", details["reason"])

	close(release)
	wg.Wait()
}

func TestAdmissionMiddleware_QueueTimeout(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		w.WriteHeader(http.StatusNoContent)
	})

	mw := NewAdmissionMiddleware(AdmissionControlConfig{
		MaxConcurrentRequests: 1,
		MaxQueuedRequests:     1,
		QueueTimeout:          50 * time.Millisecond,
	})
	h := mw(handler)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		req := httptest.NewRequest(http.MethodGet, "/slow", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}()
	<-started

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	errBody, ok := envelope["error"].(map[string]any)
	require.True(t, ok)
	details, ok := errBody["details"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "queue_timeout", details["reason"])

	close(release)
	wg.Wait()
}
