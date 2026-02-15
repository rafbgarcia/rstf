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

#### Startup sequence (current MVP)

1. **Run codegen** (`codegen.Generate(".")`) — parse route directories, generate `.rstf/types/{route}.d.ts` type declarations, `.rstf/generated/{path}.ts` dual-mode runtime modules, `.rstf/entries/{name}.entry.tsx` hydration entries, and `.rstf/server_gen.go`.
2. **Bundle client JS** — for each SSR route, run `bun build` on the hydration entry to produce `.rstf/static/{name}/bundle.js`.
3. **Start Go HTTP server** — `go run ./.rstf/server_gen.go`, which itself starts the Bun sidecar and listens on `:3000`. The generated server serves static bundles from `/.rstf/static/` and assembles full HTML pages with `<!DOCTYPE html>`, server data injection, and bundle script tags.

#### Startup sequence (planned)

Steps 3-5 below are not yet implemented — the current MVP delegates sidecar management to the generated server.

1. **Run codegen** — same as above.
2. **Bundle client JS** — same as above.
3. **Start Bun sidecar** — launch `runtime/ssr.ts` as a child process, read port from stdout.
4. **Start Go HTTP server** — compile and run `.rstf/server_gen.go`, listening on `:3000`.
5. **Start file watcher** — watch for changes (see `internal/watcher/watcher-spec.md`).

#### Process management

Manages two child processes:

```
rstf dev (orchestrator)
  ├── Bun sidecar (runtime/ssr.ts) — long-running
  └── Go HTTP server (.rstf/server_gen.go) — restarted on .go changes
```

#### Graceful shutdown (Ctrl+C)

Current MVP: forwards SIGINT to the `go run` child process, which exits and takes the Bun sidecar with it.

Planned:

1. Stop the file watcher.
2. SIGTERM the Go HTTP server.
3. SIGTERM the Bun sidecar.
4. Wait for both to exit.

#### CLI output (current MVP)

```
rstf dev
  Codegen ......... done (2 routes)
  Client bundles .. done
  HTTP server ..... starting on :3000
```

#### CLI output (planned)

```
rstf dev
  Codegen ......... done (2 routes)
  Client bundles .. done
  Bun sidecar ..... running on :41234
  HTTP server ..... running on :3000

  Watching for changes...

[12:01:05] routes/dashboard/index.go changed → codegen + restart
[12:01:06] Server restarted
```

### `rstf build` (future)

Production build.

- Runs codegen.
- Bundles client JS with minification.
- Compiles Go server into a single binary.

## Project structure

See `ARCHITECTURE.md` for the user's project structure, `internal/conventions/conventions-spec.md` for file conventions, and `internal/codegen/codegen-spec.md` for the generated `.rstf/` directory.
