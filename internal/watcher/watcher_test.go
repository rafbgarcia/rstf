package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, w.Start())
	defer w.Stop()

	// Write a .go file.
	path := filepath.Join(dir, "main.go")
	os.WriteFile(path, []byte("package main"), 0644)

	batch, ok := waitBatch(events, 2*time.Second)
	require.True(t, ok, "expected event for .go file, got none")
	assert.Equal(t, "go", batch[0].Kind)
}

func TestTsxFileChange(t *testing.T) {
	dir := t.TempDir()

	events := make(chan []Event, 10)
	w := New(dir, func(batch []Event) { events <- batch })
	require.NoError(t, w.Start())
	defer w.Stop()

	path := filepath.Join(dir, "App.tsx")
	os.WriteFile(path, []byte("export function View() {}"), 0644)

	batch, ok := waitBatch(events, 2*time.Second)
	require.True(t, ok, "expected event for .tsx file, got none")
	assert.Equal(t, "tsx", batch[0].Kind)
}

func TestCssFileChange(t *testing.T) {
	dir := t.TempDir()

	events := make(chan []Event, 10)
	w := New(dir, func(batch []Event) { events <- batch })
	require.NoError(t, w.Start())
	defer w.Stop()

	path := filepath.Join(dir, "main.css")
	os.WriteFile(path, []byte("body { color: red; }"), 0644)

	batch, ok := waitBatch(events, 2*time.Second)
	require.True(t, ok, "expected event for .css file, got none")
	assert.Equal(t, "css", batch[0].Kind)
}

func TestTsFileIgnored(t *testing.T) {
	dir := t.TempDir()

	events := make(chan []Event, 10)
	w := New(dir, func(batch []Event) { events <- batch })
	require.NoError(t, w.Start())
	defer w.Stop()

	// Write a .ts file — should NOT produce an event.
	path := filepath.Join(dir, "utils.ts")
	os.WriteFile(path, []byte("export const x = 1"), 0644)

	_, ok := waitBatch(events, 500*time.Millisecond)
	assert.False(t, ok, "expected no event for .ts file, but got one")
}

func TestIgnoredDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create ignored directories before starting the watcher.
	for _, name := range []string{".rstf", ".git", "node_modules"} {
		os.MkdirAll(filepath.Join(dir, name), 0755)
	}

	events := make(chan []Event, 10)
	w := New(dir, func(batch []Event) { events <- batch })
	require.NoError(t, w.Start())
	defer w.Stop()

	// Write .go files inside ignored directories.
	for _, name := range []string{".rstf", ".git", "node_modules"} {
		path := filepath.Join(dir, name, "file.go")
		os.WriteFile(path, []byte("package x"), 0644)
	}

	_, ok := waitBatch(events, 500*time.Millisecond)
	assert.False(t, ok, "expected no event for files in ignored directories, but got one")
}

func TestNewSubdirectoryWatched(t *testing.T) {
	dir := t.TempDir()

	events := make(chan []Event, 10)
	w := New(dir, func(batch []Event) { events <- batch })
	require.NoError(t, w.Start())
	defer w.Stop()

	// Create a new subdirectory, then write a .go file inside it.
	subdir := filepath.Join(dir, "routes", "dashboard")
	os.MkdirAll(subdir, 0755)

	// Give the watcher time to register the new directory.
	time.Sleep(200 * time.Millisecond)

	path := filepath.Join(subdir, "index.go")
	os.WriteFile(path, []byte("package dashboard"), 0644)

	batch, ok := waitBatch(events, 2*time.Second)
	require.True(t, ok, "expected event for .go file in new subdirectory, got none")
	assert.Equal(t, "go", batch[0].Kind)
}
