# CLI Specification

## Overview

`rstf` is the developer-facing CLI for the framework. It is distributed as a standalone binary. The developer installs it, runs `rstf init` to scaffold a project, and `rstf dev` to start developing.

## Commands

### `rstf init`

Scaffolds a new project in the current directory.

Creates:

- `go.mod` — Go module (prompts for module path or infers from directory)
- `package.json` — with `react`, `react-dom`, `@types/react` dependencies
- `tsconfig.json` — configured with `"include": [".rstf/types", "**/*.ts", "**/*.tsx"]`
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

## Generated output (`.rstf/` directory)

All framework-generated files go in `.rstf/` to keep the developer's project clean:

```
.rstf/
  server_gen.go              # Generated Go server with route handlers
  types/
    dashboard.d.ts           # Global types for /dashboard
    settings.d.ts            # Global types for /settings
  generated/
    main.ts                  # Runtime module for layout (serverData + __setServerData)
    routes/
      dashboard.ts           # Runtime module for /dashboard
      settings.ts            # Runtime module for /settings
  entries/
    dashboard.entry.tsx      # Hydration entry for /dashboard
  static/
    dashboard/
      bundle.js              # Hydration bundle for /dashboard
    settings/
      bundle.js              # Hydration bundle for /settings
```

## User's project structure

After `rstf init` and creating some routes:

```
my-app/
  .rstf/                     # gitignored, framework output
    types/
      dashboard.d.ts
      settings.d.ts
    generated/
      main.ts
      routes/
        dashboard.ts
        settings.ts
    server_gen.go
    static/
  main.go                    # Layout SSR handler (package myapp)
  main.tsx                   # Layout component
  routes/
    dashboard/
      index.go               # Server data (package dashboard)
      index.tsx               # GET /dashboard
    settings/
      index.go               # Server data (package settings)
      index.tsx               # GET /settings
  go.mod
  package.json
  tsconfig.json
  .gitignore
```
