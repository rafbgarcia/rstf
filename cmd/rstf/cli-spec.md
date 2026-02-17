# CLI Specification

## Overview

`rstf` is the developer-facing CLI for the framework. It is distributed as a standalone binary. The developer installs it, runs `rstf init` to scaffold a project, and `rstf dev` to start developing.

## Commands

### `rstf init` (not yet implemented)

Scaffolds a new project in the current directory.

Creates:

- `go.mod` — Go module (prompts for module path or infers from directory)
- `package.json` — with `react`, `react-dom`, `@types/react` dependencies
- `tsconfig.json` — configured with `@rstf/*` path alias and type includes (see `internal/codegen/codegen-spec.md` for full tsconfig)
- `.gitignore` — ignoring `.rstf/`, `node_modules/`
- `.rstf/` — empty framework output directory

#### Prerequisites check

Before scaffolding, `rstf init` verifies:

- Go is installed (`go version`)
- Bun is installed (`bun --version`)

If either is missing, it prints instructions and exits.

### `rstf dev`

Starts the development server.

#### Flags

- `--port <port>` — HTTP server port (default: `3000`). Passed through to the generated server binary.

#### Startup sequence

1. **Run codegen** (`codegen.Generate(".")`) — parse route directories, generate `.rstf/types/{route}.d.ts` type declarations, `.rstf/generated/{path}.ts` dual-mode runtime modules, `.rstf/entries/{name}.entry.tsx` hydration entries, and `.rstf/server_gen.go`.
2. **Bundle client JS** — for each SSR route, run `bun build` on the hydration entry to produce `.rstf/static/{name}/bundle.js`.
3. **Build CSS** (if `main.css` exists) — if `postcss.config.mjs` is present, process `main.css` through PostCSS via a generated build script; otherwise copy `main.css` as-is. Output goes to `.rstf/static/main.css`. The generated server detects this file at startup and injects a `<link>` tag.
4. **Start Go HTTP server** — `go run ./.rstf/server_gen.go --port {port}`, which itself starts the Bun sidecar and listens on the specified port. The generated server serves static assets from `/.rstf/static/` and assembles full HTML pages with `<!DOCTYPE html>`, optional CSS link, server data injection, and bundle script tags.
5. **Start file watcher** — watch for `.go`, `.tsx`, and `.css` changes (see `internal/watcher/watcher-spec.md`).

#### Process management

The Go server owns the Bun sidecar in both dev and production — same architecture, one code path. The CLI manages a single child process (`go run`), which internally starts the sidecar.

Bun starts in milliseconds, so restarting the sidecar on Go changes has negligible cost. The slow part of a restart is `go run` recompilation, not sidecar startup.

#### File watcher behavior

- **`.go` change** — re-run codegen, kill the server (sidecar dies with it), re-bundle JS, rebuild CSS, restart.
- **`.tsx` change** — re-bundle all entries, rebuild CSS (Tailwind scans TSX for class names), hit the sidecar's cache invalidation endpoint via `.rstf/sidecar.port`. No restart.
- **`.css` change** — rebuild CSS only. No JS rebundle, no sidecar invalidation (CSS is served statically).

See `internal/watcher/watcher-spec.md` for details.

#### Graceful shutdown (Ctrl+C)

Forwards SIGINT to the `go run` child process. The generated server handles SIGINT/SIGTERM by calling `renderer.Stop()` to terminate the Bun sidecar before exiting. This prevents orphaned sidecar processes.

#### CLI output

```
rstf dev
  Codegen ......... done (2 routes)
  Client bundles .. done
  CSS ............. done
  HTTP server ..... starting on :3000
```

The CSS line only appears if `main.css` exists at the project root.

With file watcher:

```
rstf dev
  Codegen ......... done (2 routes)
  Client bundles .. done
  CSS ............. done
  HTTP server ..... starting on :3000

  Watching for changes...

  [change] routes/dashboard/index.go
  Codegen ......... done (2 routes)
  Client bundles .. done
  HTTP server ..... restarting on :3000

  [change] main.css
  CSS ............. done
```

### `rstf build` (future)

Production build.

- Runs codegen.
- Bundles client JS with minification.
- Compiles Go server into a single binary.

## Project structure

See `ARCHITECTURE.md` for the user's project structure, `internal/conventions/conventions-spec.md` for file conventions, and `internal/codegen/codegen-spec.md` for the generated `.rstf/` directory.
