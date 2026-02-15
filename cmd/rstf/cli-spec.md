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

#### Startup sequence

1. **Run codegen** (`codegen.Generate(".")`) — parse route directories, generate `.rstf/types/{route}.d.ts` type declarations, `.rstf/generated/{path}.ts` dual-mode runtime modules, `.rstf/entries/{name}.entry.tsx` hydration entries, and `.rstf/server_gen.go`.
2. **Bundle client JS** — for each SSR route, run `bun build` on the hydration entry to produce `.rstf/static/{name}/bundle.js`.
3. **Start Go HTTP server** — `go run ./.rstf/server_gen.go`, which itself starts the Bun sidecar and listens on `:3000`. The generated server serves static bundles from `/.rstf/static/` and assembles full HTML pages with `<!DOCTYPE html>`, server data injection, and bundle script tags.
4. **Start file watcher** — watch for `.go` and `.tsx` changes (see `internal/watcher/watcher-spec.md`).

#### Process management

The Go server owns the Bun sidecar in both dev and production — same architecture, one code path. The CLI manages a single child process (`go run`), which internally starts the sidecar.

Bun starts in milliseconds, so restarting the sidecar on Go changes has negligible cost. The slow part of a restart is `go run` recompilation, not sidecar startup.

#### File watcher behavior

- **`.go` change** — re-run codegen, kill the server (sidecar dies with it), re-bundle, restart.
- **`.tsx` change** — re-bundle all entries, hit the sidecar's cache invalidation endpoint via `.rstf/sidecar.port`. No restart.

See `internal/watcher/watcher-spec.md` for details.

#### Graceful shutdown (Ctrl+C)

Forwards SIGINT to the `go run` child process, which exits and takes the Bun sidecar with it.

#### CLI output

```
rstf dev
  Codegen ......... done (2 routes)
  Client bundles .. done
  HTTP server ..... starting on :3000
```

With file watcher:

```
rstf dev
  Codegen ......... done (2 routes)
  Client bundles .. done
  HTTP server ..... starting on :3000

  Watching for changes...

  [change] routes/dashboard/index.go
  Codegen ......... done (2 routes)
  Client bundles .. done
  HTTP server ..... restarting on :3000
```

### `rstf build` (future)

Production build.

- Runs codegen.
- Bundles client JS with minification.
- Compiles Go server into a single binary.

## Project structure

See `ARCHITECTURE.md` for the user's project structure, `internal/conventions/conventions-spec.md` for file conventions, and `internal/codegen/codegen-spec.md` for the generated `.rstf/` directory.
