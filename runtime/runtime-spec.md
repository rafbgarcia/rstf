# Runtime Specification

The `runtime/` directory contains the JavaScript/TypeScript code that runs outside of Go — server-side (Bun sidecar) and client-side (hydration in the browser).

## Server-side: SSR sidecar (`runtime/ssr.ts`)

See `renderer/renderer-spec.md` for the full protocol.

## Server data consumption

### Why `serverData()` is a function

The generated module is cached by the Bun sidecar across requests. A function call ensures the component reads the current request's data at render time, not stale data from import time. See `renderer/renderer-spec.md` for concurrency safety.

### Scoping

Each generated module is independent. `@rstf/shared/ui/user-avatar` only contains user avatar data. A component imports `serverData` from its own generated module — no cross-access between modules.

### Mixed data sources

A component can use both server data (from its paired `.go` file, via `serverData()`) and React props (from its parent). Server data is for backend concerns (auth, DB queries); props are for component-to-component communication.

## Layout composition

The layout component (`main.tsx`) receives the route component as standard React `children`. The sidecar renders `<Layout><Route /></Layout>` — single render tree, single hydration root. See `internal/conventions/conventions-spec.md` for the layout file convention.

## Client-side hydration

### Page assembly

The generated `server_gen.go` assembles the full HTML page after rendering:

1. Prepends `<!DOCTYPE html>`
2. Injects `<script>window.__RSTF_SERVER_DATA__ = {...}</script>` before `</body>`
3. Injects the hydration bundle `<script src="..."></script>` before `</body>`

### Hydration entries

For each SSR route, codegen generates a hydration entry (`.rstf/entries/{name}.entry.tsx`) that imports the layout, route, all generated modules, and calls `hydrateRoot`. These are bundled by `bun build` into `.rstf/static/{route}/bundle.js`.

### Client-side module initialization

On the client, generated modules initialize `_data` from `window.__RSTF_SERVER_DATA__` at import time (detected via `typeof window !== "undefined"`). This means `serverData()` works identically on both server and client — it reads from `_data` at call time.

### Hydration sequence

1. Browser receives HTML — page is visible immediately
2. Browser parses `__RSTF_SERVER_DATA__`
3. Browser loads `bundle.js`
4. Generated modules initialize `_data` from `__RSTF_SERVER_DATA__`
5. `hydrateRoot` runs — React attaches to existing DOM
6. `useEffect`, `useState`, `onClick`, etc. become active

### Client-only APIs during SSR

| API         | Server             | Client                   |
| ----------- | ------------------ | ------------------------ |
| `useEffect` | Ignored            | Runs after hydration     |
| `useState`  | Initial value only | Becomes interactive      |
| `useRef`    | Ignored            | Attached after hydration |
| `onClick`   | Not in HTML        | Attached after hydration |

### Static file serving

The Go server serves `.rstf/static/` at `GET /.rstf/static/` for hydration bundles.

## Page metadata (not yet implemented)

Routes will export a `meta` value (object or function) to set per-page `<title>` and description. If `meta` is a function, the sidecar calls it after `__setServerData` so `serverData()` returns current values. The result is merged into layout server data for the layout to render `<head>` tags.

## Out of scope (MVP)

- Code splitting / shared React bundle across routes
- CSS extraction and injection
- Streaming SSR (`renderToPipeableStream`)
- Selective hydration / React Server Components
