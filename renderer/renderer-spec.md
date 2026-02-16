# SSR Renderer Specification

## Overview

The SSR renderer turns a React component tree into an HTML string on the server. It consists of:

1. **Bun sidecar** (`runtime/ssr.ts`) — HTTP server running in Bun that executes `renderToString`
2. **Go renderer client** (`renderer/renderer.go`) — manages the sidecar process and sends render requests

The renderer returns raw component HTML only — no HTML shell, no hydration scripts. Page assembly (DOCTYPE, script injection) is the caller's responsibility.

## Render request contract

The Go client sends `POST /render` to the sidecar with:

- `component` (required) — route component path relative to project root (e.g. `"routes/dashboard"`)
- `layout` (required) — layout component path (e.g. `"main"`)
- `serverData` (optional) — per-component data, keyed by component path. Keys and values come from Go structs serialized via `json.Marshal`.

The sidecar returns `{"html": "..."}` on success, or HTTP 500 with `{"error": "..."}` on failure.

## Component resolution

Component paths are **directory paths**. The sidecar resolves them using the folder convention (see `internal/conventions/conventions-spec.md`):

- Route/shared components: `"routes/dashboard"` → `{project-root}/routes/dashboard/index.tsx`
- Layout: `"main"` → `{project-root}/main.tsx`

The project root is passed as a CLI argument: `bun run runtime/ssr.ts --project-root /path/to/app`.

## Render sequence

1. For each key in `serverData`, import the generated module at `.rstf/generated/{key}.ts` and call `__setServerData(data)`. Skip if no module exists.
2. Import the layout and route component modules.
3. Render `<Layout><Route /></Layout>` via `renderToString()`.

## Concurrency safety

`__setServerData` mutates a module-level variable. This is safe because `renderToString` is synchronous and Bun runs a single-threaded event loop. **Critical rule: no `await` between `__setServerData` and `renderToString`.** All async work (dynamic `import()` calls) must complete before the synchronous set-then-render block.

## Sidecar lifecycle

**Start:** The Go renderer spawns `bun run runtime/ssr.ts`. The sidecar listens on a random port and prints the port to stdout. The Go side reads it (10-second timeout).

**Stop:** Go sends SIGINT to the Bun process and waits for exit.

**Cache:** The sidecar caches imported modules for performance. A `POST /invalidate` endpoint clears all caches — called by the file watcher during development.

## Why Bun (not embedded V8)

- Full React/JSX compatibility with no API gaps.
- No CGO dependency — keeps the Go build simple.
- Fast startup (~10ms) and built-in TypeScript/JSX transpilation.
- Can be swapped for Node or Deno without changing the Go side.
