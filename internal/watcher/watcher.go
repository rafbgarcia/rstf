package watcher

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Event represents a file change detected by the watcher.
type Event struct {
	Path string // Absolute path of the changed file
	Kind string // "go" or "tsx"
}

// Watcher monitors an app directory for .go, .tsx, and .css file changes.
type Watcher struct {
	appRoot  string
	onChange func(Event)
	fsw      *fsnotify.Watcher
	done     chan struct{}
}

// New creates a Watcher that monitors appRoot for file changes.
// onChange is called for each relevant file event.
func New(appRoot string, onChange func(Event)) *Watcher {
	return &Watcher{
		appRoot:  appRoot,
		onChange: onChange,
		done:     make(chan struct{}),
	}
}

// Start begins watching the directory tree. It walks appRoot to add all
// non-ignored directories, then starts a goroutine to process events.
func (w *Watcher) Start() error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.fsw = fsw

	// Walk the tree and add all non-ignored directories.
	err = filepath.WalkDir(w.appRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable dirs
		}
		if d.IsDir() && shouldIgnoreDir(w.appRoot, path) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return fsw.Add(path)
		}
		return nil
	})
	if err != nil {
		fsw.Close()
		return err
	}

	go w.loop()
	return nil
}

// Stop terminates the watcher.
func (w *Watcher) Stop() {
	if w.fsw != nil {
		w.fsw.Close()
	}
	<-w.done
}

func (w *Watcher) loop() {
	defer close(w.done)

	// Debounce: collect events, then fire after 50ms of quiet.
	const debounce = 50 * time.Millisecond
	pending := make(map[string]Event) // keyed by path
	timer := time.NewTimer(0)
	timer.Stop() // don't fire immediately

	for {
		select {
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if e, ok := w.toEvent(ev); ok {
				pending[e.Path] = e
				timer.Reset(debounce)
			}

		case <-timer.C:
			for _, e := range pending {
				w.onChange(e)
			}
			pending = make(map[string]Event)

		case _, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			// Ignore watcher errors — not much we can do during dev.
		}
	}
}

// toEvent converts an fsnotify event into a watcher Event, if relevant.
// As a side effect, newly created directories are added to the watch list.
func (w *Watcher) toEvent(ev fsnotify.Event) (Event, bool) {
	// Only care about writes, creates, and renames (which may create new files).
	if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
		return Event{}, false
	}

	// If a new directory was created, walk it and watch all subdirectories.
	if ev.Op&fsnotify.Create != 0 {
		if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
			filepath.WalkDir(ev.Name, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() && shouldIgnoreDir(w.appRoot, path) {
					return filepath.SkipDir
				}
				if d.IsDir() {
					w.fsw.Add(path)
				}
				return nil
			})
			return Event{}, false
		}
	}

	kind := fileKind(ev.Name)
	if kind == "" {
		return Event{}, false
	}

	return Event{Path: ev.Name, Kind: kind}, true
}

// fileKind returns "go", "tsx", or "css" for watched extensions, "" otherwise.
func fileKind(path string) string {
	if strings.HasSuffix(path, ".go") {
		return "go"
	}
	if strings.HasSuffix(path, ".tsx") {
		return "tsx"
	}
	if strings.HasSuffix(path, ".css") {
		return "css"
	}
	return ""
}

// shouldIgnoreDir returns true if the directory should not be watched.
func shouldIgnoreDir(appRoot, path string) bool {
	name := filepath.Base(path)

	// Hidden directories (.git, .rstf, .DS_Store, etc.)
	if strings.HasPrefix(name, ".") && path != appRoot {
		return true
	}

	if name == "node_modules" {
		return true
	}

	return false
}
