package rstf

import "testing"

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
