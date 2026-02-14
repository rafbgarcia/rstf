# SSR Renderer Specification

## Overview

The SSR renderer is responsible for turning a React component + props into an HTML string on the server. It consists of two parts:

1. **Bun sidecar** — a small HTTP server running in a Bun process that executes `renderToString`
2. **Go renderer client** — sends render requests to the Bun sidecar and assembles the full HTML page

## Bun sidecar (`runtime/ssr.ts`)

A Bun HTTP server that accepts render requests and returns HTML strings.

### Request format

```
POST http://localhost:{port}/render
Content-Type: application/json

{
  "component": "dashboard/dashboard",
  "props": {
    "posts": [{"title": "Hello", "published": true}],
    "author": {"name": "Rafa", "email": "rafa@example.com"}
  }
}
```

### What it does

1. Receives the component path and props.
2. Imports the component module (e.g. `dashboard/dashboard.tsx`).
3. Finds the exported `View` function.
4. Calls `ReactDOMServer.renderToString(<View {...props} />)`.
5. Returns the HTML string.

### Response format

```json
{
  "html": "<div><p>Hello</p></div>"
}
```

### Error handling

If rendering fails, the sidecar returns a 500 with:

```json
{
  "error": "Component not found: dashboard/dashboard"
}
```

### Startup

- The sidecar is started by the dev command as a child process.
- It listens on a random available port and prints the port to stdout.
- The Go process reads the port from stdout to know where to send requests.

### Component resolution

The sidecar needs to know where the app root is (the directory containing route folders). It receives this as a command-line argument:

```
bun run runtime/ssr.ts --app-root ./example
```

Component paths in render requests are relative to this root. So `"component": "dashboard/dashboard"` resolves to `{app-root}/dashboard/dashboard.tsx`.

## Go renderer client (`internal/renderer/renderer.go`)

The Go side manages the Bun sidecar process and sends render requests.

### Interface

```go
type Renderer struct {
    port int
    cmd  *exec.Cmd
}

func New() *Renderer
func (r *Renderer) Start(appRoot string) error
func (r *Renderer) Stop() error
func (r *Renderer) RenderSSR(w http.ResponseWriter, component string, props map[string]any) error
```

### `RenderSSR` behavior

1. Serializes `props` to JSON.
2. Sends a POST request to `http://localhost:{port}/render`.
3. Receives the HTML string from the sidecar.
4. Wraps it in a full HTML page (see HTML shell below).
5. Writes the response to `w`.

### HTML shell

The renderer wraps the component HTML in a full page:

```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
</head>
<body>
  <div id="root">{component HTML here}</div>

  <script>
    window.__RSTF_PROPS__ = {serialized props JSON};
  </script>
  <script src="/.rstf/static/{route}/bundle.js"></script>
</body>
</html>
```

- `window.__RSTF_PROPS__` passes the props to the client for hydration.
- The `bundle.js` script is the client-side hydration bundle for this route.

## Why Bun (not embedded V8)

- Full React/JSX compatibility — no gaps in API support.
- No CGO dependency — keeps the Go build simple.
- Fast startup (~10ms) and built-in TypeScript/JSX transpilation.
- Can be swapped for Node or Deno later without changing the Go side.
