# Watcher Specification

## Overview

The watcher monitors the user's app directory for file changes and triggers the appropriate rebuild steps. It is used by `rstf dev` to provide live reloading during development.

## Interface

```go
type Event struct {
    Path string
    Kind string // "go", "tsx"
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
| `*.go`      | `"go"`     | Re-run codegen → re-bundle client JS → kill server (sidecar dies with it) → restart |
| `*.tsx`     | `"tsx"`    | Re-bundle all client bundles → signal Bun sidecar to clear module cache (no restart) |

`.ts` files are not watched — they don't trigger any rebuild since they don't have Go companions and aren't route components.

## Ignored paths

The watcher skips:
- `.rstf/` directory (framework output — includes generated types and runtime modules)
- Hidden files and directories (`.git`, `.DS_Store`, etc.)
- `node_modules/`

## New directories

When a new directory is created (e.g. a new route), the watcher walks it and adds all subdirectories to the watch list.

## Implementation

Uses `fsnotify` for filesystem notifications (efficient, no polling).
