# Architecture

## Project structure

```
myapp/
  go.mod                             # module github.com/user/myapp
  main.go                            # Layout SSR + types (package myapp)
  main.tsx                            # Layout component (wraps route via children)
  routes/
    index/
      index.tsx                      # GET /
      index.go                       # Server data for root page
    dashboard/
      index.tsx                      # GET /dashboard
      index.go                       # Server data
    users.$id.edit/
      index.tsx                      # GET /users/{id}/edit
      index.go                       # Server data
      UserEditForm.tsx               # Colocated component (not a route)
  shared/
    ui/
      user-avatar/
        index.tsx                    # Component with server data
        index.go
      button.tsx                     # Standalone component (no .go)
    hooks/
      some-hook.ts                   # Shared TypeScript code
```

See `internal/conventions/conventions-spec.md` for file conventions and route path resolution rules. See `internal/codegen/codegen-spec.md` for the generated `.rstf/` directory structure.

> **`main.go` must NOT use `package main`.** The user's `main.go` uses the app's package name (e.g. `package myapp`), making it importable by the generated `.rstf/server_gen.go` (which declares `package main` and contains `func main()`). Go prohibits importing `package main`, but any other package name works. The `rstf dev` CLI compiles `.rstf/server_gen.go` explicitly — Go's build tool skips dot-prefixed directories by default.

## Components

**Codegen** (`internal/codegen/`) — Produces all framework-generated files under `.rstf/`. Parses Go route files via AST, extracts `SSR` functions and their return structs, analyzes TypeScript imports for dependencies, and generates three outputs: `.d.ts` type declarations, `.ts` runtime modules (`serverData()` + `__setServerData()`), and `.rstf/server_gen.go` (the Go entry point that wires routes to handlers).

**Renderer** (`internal/renderer/`) — Two parts. The Go client (`renderer.go`) manages a Bun child process and sends HTTP render requests. The Bun sidecar (`runtime/ssr.ts`) receives those requests, calls `__setServerData()` on generated modules, then runs `ReactDOMServer.renderToString()` to produce HTML. The synchronous set-then-render block ensures concurrency safety on Bun's single-threaded event loop.

**Conventions** (`internal/conventions/`) — Defines the file conventions and rules that map the `routes/` directory structure to HTTP URL patterns. Each folder is a route, dots in folder names become path segments, `$param` becomes `{param}`. See `internal/codegen/codegen-spec.md` for how these rules are used to generate `.rstf/server_gen.go`.

**Watcher** (`internal/watcher/`) — Monitors `.go`, `.ts`, and `.tsx` files for changes during development. Triggers codegen + server restart for Go changes, and bundle rebuild + sidecar cache invalidation for TypeScript changes. Not yet implemented.

**CLI** (`cmd/rstf/`) — Developer-facing binary.

**Framework core** (`context.go`, `logger.go`) — Request-scoped `Context` with structured logging, passed to route handlers.

## How they connect

```
Developer writes:        Framework generates:         Request flow:

  main.go  ──┐                                    Browser
  main.tsx   │                                      │
             │         ┌─────────┐                  │ GET /dashboard
routes/      │         │         │                  ▼
  dashboard/ ├────────▶│ Codegen │        ┌──────────────────┐
    index.go │         │         │        │   Go HTTP server  │
    index.tsx│         └────┬────┘        │ (.rstf/server_gen)│
             │              │             └────────┬─────────┘
shared/      │              │                      │
  ui/        │              │               ctx := NewContext(req)
    user-    ─┘              │               data := dashboard.SSR(ctx)
    avatar.tsx               │                      │
                            ▼                      ▼
                ┌──────────────────┐    ┌─────────────────────┐
                │   .rstf/         │    │  Renderer (Go side)  │
                │                  │    │                      │
                │  types/          │    │  POST /render with   │
                │    dashboard.d.ts│    │  {component, layout, │
                │                  │    │   serverData}        │
                │  generated/      │    └──────────┬──────────┘
                │    dashboard.ts  │               │
                │    (serverData)  │               ▼
                │                  │    ┌─────────────────────┐
                │  server_gen.go   │    │  Bun sidecar (SSR)  │
                │                  │    │                      │
                └──────────────────┘    │  __setServerData()   │
                                        │  renderToString(     │
                                        │   <Layout>           │
                                        │     <Route />        │
                                        │   </Layout>)         │
                                        └──────────┬──────────┘
                                                   │
                                                   ▼
                                              HTML string
                                                   │
                                                   ▼
                                               Browser
```

## Data flow for a single request

1. Browser requests `GET /dashboard`
2. Go server (generated `server_gen.go`) matches the route
3. Handler calls `app.SSR(ctx)` (layout data) and `dashboard.SSR(ctx)` (route data)
4. Structs are converted to `map[string]any` via JSON marshal
5. Go renderer sends POST to Bun sidecar with component paths + data maps
6. Sidecar calls `__setServerData()` on each generated module (synchronous)
7. Sidecar calls `renderToString(<Layout><Route /></Layout>)` (synchronous)
8. HTML string returned to Go, written to response

## Specification files

Each module has a co-located `*-spec.md` file as the source of truth. Each concept is documented in one spec only — other specs cross-reference rather than duplicate.

| Spec                                       | Owns                                                                                                       |
| ------------------------------------------ | ---------------------------------------------------------------------------------------------------------- |
| `internal/codegen/codegen-spec.md`         | Go parsing, type mapping, generated module format, `@rstf/` alias, server code generation, `.rstf/` output |
| `internal/renderer/renderer-spec.md`       | SSR protocol, request/response format, sidecar lifecycle, concurrency model                                |
| `internal/conventions/conventions-spec.md` | File conventions, routing rules, route path resolution                                                     |
| `runtime/runtime-spec.md`                  | Runtime behavior of serverData(), layout composition, hydration (planned)                                  |
| `internal/watcher/watcher-spec.md`         | File watching, debouncing, rebuild triggers                                                                |
| `cmd/rstf/cli-spec.md`                     | CLI commands, startup sequence, process management                                                         |
