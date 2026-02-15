package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// waitEvent waits up to timeout for an event on ch. Returns the event and true,
// or a zero Event and false if the timeout expires.
func waitEvent(ch <-chan Event, timeout time.Duration) (Event, bool) {
	select {
	case ev := <-ch:
		return ev, true
	case <-time.After(timeout):
		return Event{}, false
	}
}

func TestGoFileChange(t *testing.T) {
	dir := t.TempDir()

	events := make(chan Event, 10)
	w := New(dir, func(e Event) { events <- e })
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	// Write a .go file.
	path := filepath.Join(dir, "main.go")
	os.WriteFile(path, []byte("package main"), 0644)

	ev, ok := waitEvent(events, 2*time.Second)
	if !ok {
		t.Fatal("expected event for .go file, got none")
	}
	if ev.Kind != "go" {
		t.Fatalf("expected kind %q, got %q", "go", ev.Kind)
	}
}

func TestTsxFileChange(t *testing.T) {
	dir := t.TempDir()

	events := make(chan Event, 10)
	w := New(dir, func(e Event) { events <- e })
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	path := filepath.Join(dir, "App.tsx")
	os.WriteFile(path, []byte("export function View() {}"), 0644)

	ev, ok := waitEvent(events, 2*time.Second)
	if !ok {
		t.Fatal("expected event for .tsx file, got none")
	}
	if ev.Kind != "tsx" {
		t.Fatalf("expected kind %q, got %q", "tsx", ev.Kind)
	}
}

func TestTsFileIgnored(t *testing.T) {
	dir := t.TempDir()

	events := make(chan Event, 10)
	w := New(dir, func(e Event) { events <- e })
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	// Write a .ts file — should NOT produce an event.
	path := filepath.Join(dir, "utils.ts")
	os.WriteFile(path, []byte("export const x = 1"), 0644)

	_, ok := waitEvent(events, 500*time.Millisecond)
	if ok {
		t.Fatal("expected no event for .ts file, but got one")
	}
}

func TestIgnoredDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create ignored directories before starting the watcher.
	for _, name := range []string{".rstf", ".git", "node_modules"} {
		os.MkdirAll(filepath.Join(dir, name), 0755)
	}

	events := make(chan Event, 10)
	w := New(dir, func(e Event) { events <- e })
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	// Write .go files inside ignored directories.
	for _, name := range []string{".rstf", ".git", "node_modules"} {
		path := filepath.Join(dir, name, "file.go")
		os.WriteFile(path, []byte("package x"), 0644)
	}

	_, ok := waitEvent(events, 500*time.Millisecond)
	if ok {
		t.Fatal("expected no event for files in ignored directories, but got one")
	}
}

func TestNewSubdirectoryWatched(t *testing.T) {
	dir := t.TempDir()

	events := make(chan Event, 10)
	w := New(dir, func(e Event) { events <- e })
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	// Create a new subdirectory, then write a .go file inside it.
	subdir := filepath.Join(dir, "routes", "dashboard")
	os.MkdirAll(subdir, 0755)

	// Give the watcher time to register the new directory.
	time.Sleep(200 * time.Millisecond)

	path := filepath.Join(subdir, "index.go")
	os.WriteFile(path, []byte("package dashboard"), 0644)

	ev, ok := waitEvent(events, 2*time.Second)
	if !ok {
		t.Fatal("expected event for .go file in new subdirectory, got none")
	}
	if ev.Kind != "go" {
		t.Fatalf("expected kind %q, got %q", "go", ev.Kind)
	}
}
