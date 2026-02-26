package rstf

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
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

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	var envelope map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	errBody := envelope["error"].(map[string]any)
	if got := errBody["code"]; got != string(ErrorCodeOverloaded) {
		t.Fatalf("error.code = %v, want %q", got, ErrorCodeOverloaded)
	}
	details := errBody["details"].(map[string]any)
	if got := details["reason"]; got != "queue_full" {
		t.Fatalf("details.reason = %v, want queue_full", got)
	}

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

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	var envelope map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	errBody := envelope["error"].(map[string]any)
	details := errBody["details"].(map[string]any)
	if got := details["reason"]; got != "queue_timeout" {
		t.Fatalf("details.reason = %v, want queue_timeout", got)
	}

	close(release)
	wg.Wait()
}
