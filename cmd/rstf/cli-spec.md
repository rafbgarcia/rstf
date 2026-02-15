# CLI Specification

## Overview

`rstf` is the developer-facing CLI for the framework. It is distributed as a standalone binary. The developer installs it, runs `rstf init` to scaffold a project, and `rstf dev` to start developing.

## Commands

### `rstf init`

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

Starts the development server with live reloading.

#### Startup sequence

1. **Run codegen** — parse route directories, generate `.rstf/types/{route}.d.ts` type declarations, `.rstf/generated/{path}.ts` runtime modules, and `.rstf/server_gen.go`.
2. **Bundle client JS** — for each SSR route, generate hydration entry and bundle with Bun into `.rstf/static/{route}/bundle.js`.
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

1. Stop the file watcher.
2. SIGTERM the Go HTTP server.
3. SIGTERM the Bun sidecar.
4. Wait for both to exit.

#### CLI output

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
