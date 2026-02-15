# SSR Renderer Specification

## Overview

The SSR renderer turns a React component tree into an HTML string on the server. It is a low-level primitive — no HTML shell, no hydration scripts. Page assembly is the caller's responsibility.

It consists of two parts:

1. **Bun sidecar** — a small HTTP server running in a Bun process that executes `renderToString`
2. **Go renderer client** — manages the sidecar process and sends render requests

## Server data model

The render request carries a `serverData` map keyed by component path. Before rendering, the sidecar imports each component's generated module and calls `__setServerData(data)`. See `internal/codegen/codegen-spec.md` for the generated module format and `runtime/runtime-spec.md` for how components consume server data.

## Bun sidecar (`runtime/ssr.ts`)

A Bun HTTP server that accepts render requests and returns HTML strings.

### Request format

```
POST http://localhost:{port}/render
Content-Type: application/json

{
  "component": "routes/dashboard",
  "layout": "main",
  "serverData": {
    "main": {
      "session": {"isLoggedIn": true, "user": {"name": "Rafa"}}
    },
    "routes/dashboard": {
      "posts": [{"title": "Hello", "published": true}],
      "author": {"name": "Rafa", "email": "rafa@example.com"}
    },
    "shared/ui/user-avatar": {
      "userName": "Rafa",
      "avatarUrl": "/avatars/rafa.jpg"
    }
  }
}
```

The keys within each component's data come from the Go struct's `json` tags. The Go handler calls `json.Marshal` on the `SSR()` return value, so the serialization format is determined by the struct definition.

Fields:
- `component` (required) — path to the route component, relative to project root
- `layout` (required) — path to the layout component, relative to project root
- `serverData` (optional) — per-component server data, keyed by component path

### What it does

1. Receives the component path, layout, and server data.
2. For each key in `serverData`, imports the generated module at `{projectRoot}/.rstf/generated/{key}.ts` and calls `__setServerData(data)`. Skips gracefully if no generated module exists.
3. Imports the layout component module (e.g. `main.tsx`).
4. Imports the route component module (e.g. `routes/dashboard.tsx`).
5. Renders `<Layout><Route /></Layout>` via `ReactDOMServer.renderToString()`.
6. Returns the HTML string.

The layout component receives the route as standard React `children`. No special imports or `dangerouslySetInnerHTML` — the framework handles composition at the render level.

### Concurrency safety

`__setServerData` mutates a module-level `_data` variable. If two requests interleave — request A sets data, request B sets data, request A renders — request A's `serverData()` calls would return B's data.

This is safe as long as there is **no `await` between `__setServerData` and `renderToString`**. Since `renderToString` is synchronous and Bun runs on a single-threaded event loop, a synchronous set-then-render block cannot be interrupted by another request.

The request handler must resolve all async work (dynamic `import()` calls) **before** entering the synchronous set-then-render block:

```ts
// 1. Async phase: resolve all imports
const modules = await Promise.all(
  entries.map(([path]) => import(`.../${path}.ts`))
);
// 2. Synchronous phase: set data + render (no await — cannot be interrupted)
for (let i = 0; i < entries.length; i++) {
  modules[i].__setServerData(entries[i][1]);
}
const html = renderToString(createElement(Layout, null, createElement(Route)));
```

### Response format

```json
{
  "html": "<html><body>...</body></html>"
}
```

### Error handling

If rendering fails, the sidecar returns a 500 with:

```json
{
  "error": "Component not found: routes/dashboard"
}
```

### Startup

- The sidecar is started by the Go renderer as a child process.
- It listens on a random available port and prints the port to stdout.
- The Go process reads the port from stdout to know where to send requests.

### Component resolution

The sidecar receives the project root as a command-line argument:

```
bun run runtime/ssr.ts --project-root /path/to/myapp
```

Component paths in render requests are **directory paths** relative to the project root. The sidecar resolves them to actual files using the folder convention (see `internal/conventions/conventions-spec.md`):

- **Route components** use `index.tsx` inside the directory: `"routes/dashboard"` → `{project-root}/routes/dashboard/index.tsx`
- **Shared components** use a file matching the directory name: `"shared/ui/user-avatar"` → `{project-root}/shared/ui/user-avatar/user-avatar.tsx`
- **Layout** is a root-level file: `"main"` → `{project-root}/main.tsx`

The sidecar uses Bun's standard module resolution (`import()`) which resolves directory paths to `index.tsx` automatically. For shared components whose entry file matches the directory name rather than `index.tsx`, the sidecar appends the basename (e.g. `shared/ui/user-avatar` → `shared/ui/user-avatar/user-avatar.tsx`).

### Module cache

The sidecar caches imported modules for performance. A `POST /invalidate` endpoint clears both the component module cache and the generated module cache — called by the file watcher when files change during development.

## Go renderer client (`internal/renderer/renderer.go`)

The Go side manages the Bun sidecar process and sends render requests.

### Interface

```go
type Renderer struct {
    port int
    cmd  *exec.Cmd
}

type RenderRequest struct {
    Component  string                    // route component path (required)
    Layout     string                    // layout component path (required)
    ServerData map[string]map[string]any // per-component server data, keyed by component path
}

func New() *Renderer
func (r *Renderer) Start(projectRoot string) error
func (r *Renderer) Stop() error
func (r *Renderer) Render(req RenderRequest) (string, error)
```

### `Render` behavior

1. Serializes the `RenderRequest` to JSON.
2. Sends a POST request to `http://localhost:{port}/render`.
3. Receives the HTML string from the sidecar.
4. Returns the HTML string to the caller.

The renderer returns raw component HTML only. It does not wrap it in a page shell, add hydration scripts, or write to an `http.ResponseWriter`. The caller composes the full page (prepends DOCTYPE, injects script tags before `</body>`).

### `Start` behavior

1. Locates `runtime/ssr.ts` relative to the framework module root.
2. Spawns `bun run runtime/ssr.ts --project-root {projectRoot}`.
3. Reads the port number from the first line of stdout.
4. Stores the port for subsequent render requests.
5. Times out after 10 seconds if no port is received.

### `Stop` behavior

1. Sends SIGINT to the Bun process.
2. Waits for the process to exit.

## Why Bun (not embedded V8)

- Full React/JSX compatibility — no gaps in API support.
- No CGO dependency — keeps the Go build simple.
- Fast startup (~10ms) and built-in TypeScript/JSX transpilation.
- Can be swapped for Node or Deno later without changing the Go side.
