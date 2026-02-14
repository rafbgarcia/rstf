# Watcher Specification

## Overview

The watcher monitors the user's app directory for file changes and triggers the appropriate rebuild steps. It is used by `rstf dev` to provide live reloading during development.

## Interface

```go
type Event struct {
    Path string
    Kind string // "go", "ts", "tsx"
}

type Watcher struct { ... }

func New(appRoot string, onChange func(Event)) *Watcher
func (w *Watcher) Start() error
func (w *Watcher) Stop()
```

The watcher calls `onChange` whenever a relevant file is created, modified, or deleted. The caller (the CLI `dev` command) decides what to do based on the event kind.

## What triggers a rebuild

| File changed | Event kind | What the CLI does |
|-------------|------------|-------------------|
| `*.go`      | `"go"`     | Re-run codegen → regenerate TS types + Go server → restart Go server |
| `*.ts`      | `"ts"`     | Rebuild client bundles → signal Bun sidecar to clear module cache |
| `*.tsx`     | `"tsx"`    | Rebuild client bundles → signal Bun sidecar to clear module cache |

## Ignored paths

The watcher skips:
- `.rstf/` directory (framework output — includes generated types and runtime modules)
- Hidden files and directories (`.git`, `.DS_Store`, etc.)
- `node_modules/`

## Debouncing

Editors often trigger multiple file change events on a single save (write temp file, rename, delete backup). The watcher debounces with a ~100ms delay — it collects events during the window and fires `onChange` once with the most recent event per file.

## Implementation

Uses `fsnotify` for filesystem notifications (efficient, no polling). Falls back to polling with `filepath.WalkDir` if `fsnotify` is not available on the platform.
