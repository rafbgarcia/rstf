package rstf

import (
	"testing"
	"time"
)

func TestAppRequestBodyLimit_Default(t *testing.T) {
	app := NewApp()
	if got, want := app.RequestBodyLimitBytes(), DefaultBodyLimit; got != want {
		t.Fatalf("RequestBodyLimitBytes() = %d, want %d", got, want)
	}
}

func TestAppRequestBodyLimit_Set(t *testing.T) {
	app := NewApp()
	if err := app.SetRequestBodyLimitBytes(2048); err != nil {
		t.Fatalf("SetRequestBodyLimitBytes: %v", err)
	}
	if got, want := app.RequestBodyLimitBytes(), int64(2048); got != want {
		t.Fatalf("RequestBodyLimitBytes() = %d, want %d", got, want)
	}
}

func TestAppRequestBodyLimit_SetInvalid(t *testing.T) {
	app := NewApp()
	if err := app.SetRequestBodyLimitBytes(0); err == nil {
		t.Fatal("expected error for limit=0, got nil")
	}
}

func TestAppAdmissionDefaults(t *testing.T) {
	app := NewApp()
	if got, want := app.MaxConcurrentRequests(), DefaultMaxConcurrentRequests; got != want {
		t.Fatalf("MaxConcurrentRequests() = %d, want %d", got, want)
	}
	if got, want := app.MaxQueuedRequests(), DefaultMaxQueuedRequests; got != want {
		t.Fatalf("MaxQueuedRequests() = %d, want %d", got, want)
	}
	if got, want := app.QueueTimeout(), DefaultQueueTimeout; got != want {
		t.Fatalf("QueueTimeout() = %s, want %s", got, want)
	}
}

func TestAppAdmissionSetters(t *testing.T) {
	app := NewApp()
	if err := app.SetMaxConcurrentRequests(3); err != nil {
		t.Fatalf("SetMaxConcurrentRequests: %v", err)
	}
	if err := app.SetMaxQueuedRequests(4); err != nil {
		t.Fatalf("SetMaxQueuedRequests: %v", err)
	}
	if err := app.SetQueueTimeout(150 * time.Millisecond); err != nil {
		t.Fatalf("SetQueueTimeout: %v", err)
	}

	if got, want := app.MaxConcurrentRequests(), 3; got != want {
		t.Fatalf("MaxConcurrentRequests() = %d, want %d", got, want)
	}
	if got, want := app.MaxQueuedRequests(), 4; got != want {
		t.Fatalf("MaxQueuedRequests() = %d, want %d", got, want)
	}
	if got, want := app.QueueTimeout(), 150*time.Millisecond; got != want {
		t.Fatalf("QueueTimeout() = %s, want %s", got, want)
	}
}

func TestAppAdmissionSettersRejectInvalid(t *testing.T) {
	app := NewApp()
	if err := app.SetMaxConcurrentRequests(0); err == nil {
		t.Fatal("expected error for max concurrent requests=0")
	}
	if err := app.SetMaxQueuedRequests(0); err == nil {
		t.Fatal("expected error for max queued requests=0")
	}
	if err := app.SetQueueTimeout(0); err == nil {
		t.Fatal("expected error for queue timeout=0")
	}
}
