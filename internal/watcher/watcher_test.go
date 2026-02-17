package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// waitBatch waits up to timeout for a batch of events on ch. Returns the batch
// and true, or nil and false if the timeout expires.
func waitBatch(ch <-chan []Event, timeout time.Duration) ([]Event, bool) {
	select {
	case batch := <-ch:
		return batch, true
	case <-time.After(timeout):
		return nil, false
	}
}

func TestGoFileChange(t *testing.T) {
	dir := t.TempDir()

	events := make(chan []Event, 10)
	w := New(dir, func(batch []Event) { events <- batch })
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	// Write a .go file.
	path := filepath.Join(dir, "main.go")
	os.WriteFile(path, []byte("package main"), 0644)

	batch, ok := waitBatch(events, 2*time.Second)
	if !ok {
		t.Fatal("expected event for .go file, got none")
	}
	if batch[0].Kind != "go" {
		t.Fatalf("expected kind %q, got %q", "go", batch[0].Kind)
	}
}

func TestTsxFileChange(t *testing.T) {
	dir := t.TempDir()

	events := make(chan []Event, 10)
	w := New(dir, func(batch []Event) { events <- batch })
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	path := filepath.Join(dir, "App.tsx")
	os.WriteFile(path, []byte("export function View() {}"), 0644)

	batch, ok := waitBatch(events, 2*time.Second)
	if !ok {
		t.Fatal("expected event for .tsx file, got none")
	}
	if batch[0].Kind != "tsx" {
		t.Fatalf("expected kind %q, got %q", "tsx", batch[0].Kind)
	}
}

func TestCssFileChange(t *testing.T) {
	dir := t.TempDir()

	events := make(chan []Event, 10)
	w := New(dir, func(batch []Event) { events <- batch })
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	path := filepath.Join(dir, "main.css")
	os.WriteFile(path, []byte("body { color: red; }"), 0644)

	batch, ok := waitBatch(events, 2*time.Second)
	if !ok {
		t.Fatal("expected event for .css file, got none")
	}
	if batch[0].Kind != "css" {
		t.Fatalf("expected kind %q, got %q", "css", batch[0].Kind)
	}
}

func TestTsFileIgnored(t *testing.T) {
	dir := t.TempDir()

	events := make(chan []Event, 10)
	w := New(dir, func(batch []Event) { events <- batch })
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	// Write a .ts file — should NOT produce an event.
	path := filepath.Join(dir, "utils.ts")
	os.WriteFile(path, []byte("export const x = 1"), 0644)

	_, ok := waitBatch(events, 500*time.Millisecond)
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

	events := make(chan []Event, 10)
	w := New(dir, func(batch []Event) { events <- batch })
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	// Write .go files inside ignored directories.
	for _, name := range []string{".rstf", ".git", "node_modules"} {
		path := filepath.Join(dir, name, "file.go")
		os.WriteFile(path, []byte("package x"), 0644)
	}

	_, ok := waitBatch(events, 500*time.Millisecond)
	if ok {
		t.Fatal("expected no event for files in ignored directories, but got one")
	}
}

func TestNewSubdirectoryWatched(t *testing.T) {
	dir := t.TempDir()

	events := make(chan []Event, 10)
	w := New(dir, func(batch []Event) { events <- batch })
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

	batch, ok := waitBatch(events, 2*time.Second)
	if !ok {
		t.Fatal("expected event for .go file in new subdirectory, got none")
	}
	if batch[0].Kind != "go" {
		t.Fatalf("expected kind %q, got %q", "go", batch[0].Kind)
	}
}
