# SSR Renderer Specification

## Overview

The SSR renderer turns a React component tree into an HTML string on the server. It is a low-level primitive — no HTML shell, no hydration scripts. Page assembly is the caller's responsibility.

It consists of two parts:

1. **Bun sidecar** — a small HTTP server running in a Bun process that executes `renderToString`
2. **Go renderer client** — manages the sidecar process and sends render requests

## Server data model

Components don't receive server data as React props. Instead, each `.go` file's public functions are called on the server, and their return values are made available to the paired `.tsx` file via ES module live bindings.

The render request carries a `serverData` map keyed by component path. Before rendering, the sidecar imports each component's generated module (at `.rstf/generated/{path}.ts`) and calls its `__setServerData()` function. Components import typed values directly from `@rstf/{path}`.

This means:
- Server data flows through ES module live bindings (scoped per generated module)
- React props are used for component-to-component communication (parent → child)
- A component can use both: server data from its `.go` file AND React props from its parent

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
      "Session": {"isLoggedIn": true, "user": {"name": "Rafa"}}
    },
    "routes/dashboard": {
      "Posts": [{"title": "Hello", "published": true}]
    },
    "shared/ui/user-avatar": {
      "UserName": "Rafa",
      "AvatarUrl": "/avatars/rafa.jpg"
    }
  }
}
```

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

`__setServerData` mutates global module-level variables. If two requests interleave — request A sets data, request B sets data, request A renders — request A would render with B's data.

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

Component paths in render requests are relative to this root. So `"component": "routes/dashboard"` resolves to `{project-root}/routes/dashboard.tsx`.

This supports any component location — routes, shared components, or the layout entrypoint:

- `"routes/dashboard"` → `{project-root}/routes/dashboard.tsx`
- `"shared/ui/user-avatar"` → `{project-root}/shared/ui/user-avatar.tsx`
- `"main"` → `{project-root}/main.tsx`

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
