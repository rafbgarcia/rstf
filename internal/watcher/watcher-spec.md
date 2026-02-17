# Watcher Specification

## Overview

The watcher monitors the user's app directory for file changes and triggers the appropriate rebuild steps. It is used by `rstf dev` to provide live reloading during development.

## Interface

```go
type Event struct {
    Path string
    Kind string // "go", "tsx", "css"
}

type Watcher struct { ... }

func New(appRoot string, onChange func([]Event)) *Watcher
func (w *Watcher) Start() error
func (w *Watcher) Stop()
```

The watcher debounces filesystem events (50ms quiet period), then calls `onChange` with the full batch. The caller (the CLI `dev` command) classifies the batch and decides what to do.

## What triggers a rebuild

| File changed | Event kind | What the CLI does |
|-------------|------------|-------------------|
| `*.go`      | `"go"`     | Incremental codegen (`Regenerate`) → re-bundle client JS → rebuild CSS → restart server |
| `*.tsx`     | `"tsx"`    | Incremental codegen (`Regenerate`) → re-bundle client JS → rebuild CSS → restart server only if `server_gen.go` changed, otherwise just invalidate sidecar |
| `*.css`     | `"css"`    | Rebuild CSS only (no JS rebundle, no sidecar invalidation — CSS is served statically) |

`.ts` files are not watched — they don't trigger any rebuild since they don't have Go companions and aren't route components.

## Batch events

The watcher fires a single callback per debounce cycle with all events in that batch. This allows the CLI to:

- Classify the entire batch (has Go? has TSX? has CSS?) in one pass
- Run a single incremental codegen covering all changes
- Make one restart decision instead of multiple

## Ignored paths

The watcher skips:
- `.rstf/` directory (framework output — includes generated types and runtime modules)
- Hidden files and directories (`.git`, `.DS_Store`, etc.)
- `node_modules/`

## New directories

When a new directory is created (e.g. a new route), the watcher walks it and adds all subdirectories to the watch list.

## Implementation

Uses `fsnotify` for filesystem notifications (efficient, no polling).
