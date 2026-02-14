# Runtime Specification

The `runtime/` directory contains the JavaScript/TypeScript code that runs outside of Go — both server-side (Bun sidecar) and client-side (hydration in the browser).

## Server-side: SSR sidecar (`runtime/ssr.ts`)

A Bun HTTP server that the Go renderer communicates with. See `internal/renderer/renderer-spec.md` for the full protocol (request/response format, startup, component resolution).

In summary:
- Receives `{ component, layout, serverData? }` via POST.
- Sets server data on generated modules via `__setServerData()`.
- Nests the route inside the layout via React `children`.
- Calls `ReactDOMServer.renderToString()`.
- Returns the raw HTML string (no page shell, no hydration scripts).

### Module cache

The sidecar caches imported modules for performance. During development, the watcher signals the sidecar to clear its cache via `POST /invalidate` when files change, so the next render picks up the new code.

## Server data mechanism

Server data is the bridge between Go functions and React components. Each `.go` file's public functions are called on the server, and their return values are made available to the paired `.tsx` file as direct imports.

### How it works

1. Codegen reads the `.go` file and generates a `.ts` module under `.rstf/generated/` with `export let` bindings and a `__setServerData()` function.
2. The Go handler calls each relevant `.go` file's functions, collecting return values into a `serverData` map keyed by component path.
3. The sidecar imports each generated module and calls `__setServerData(data)` before rendering.
4. During `renderToString` (synchronous), components read the live bindings, which reflect the current request's data.

### Generated modules (codegen responsibility)

For each `.go` file with public functions, codegen generates a module under `.rstf/generated/`:

```go
// shared/ui/user-avatar.go
func UserName(ctx rstf.Context) string {
    return ctx.Session().User.FirstName + " " + ctx.Session().User.LastName
}
func AvatarUrl(ctx rstf.Context) string {
    return ctx.Session().User.AvatarURL
}
```

Generates:

```typescript
// .rstf/generated/shared/ui/user-avatar.ts (generated — do not edit)
export let UserName: string = "";
export let AvatarUrl: string = "";

export function __setServerData(data: Record<string, any>) {
  UserName = data.UserName ?? "";
  AvatarUrl = data.AvatarUrl ?? "";
}
```

A `tsconfig.json` path alias maps `@rstf/*` to `.rstf/generated/*`, so components import as:

```tsx
import { UserName, AvatarUrl } from "@rstf/shared/ui/user-avatar";
```

### Why this works (ES module live bindings)

ES module `export let` creates live bindings — importers hold a reference to the binding, not a copy of the value. When `__setServerData()` reassigns `UserName`, all importers see the new value on their next read.

Since `renderToString` is synchronous, there's no concurrency. The sidecar sets data, renders, returns — one request at a time.

### Scoping

Each generated module is independent. `@rstf/shared/ui/user-avatar` only contains `UserName` and `AvatarUrl`. `@rstf/routes/dashboard` only contains `Posts`. A component can only import from its own generated module — no cross-access.

### Mixed data sources

A component can use both server data AND React props:

- **Server data** (from `.go` file): imported from `@rstf/{path}` — for auth, database queries, session info
- **React props** (from parent component): passed as regular JSX attributes — for component-to-component communication

```tsx
// shared/ui/user-avatar.tsx
import { UserName, AvatarUrl } from "@rstf/shared/ui/user-avatar";

export function View({ notificationCount }: { notificationCount?: number }) {
  return (
    <div>
      <img src={AvatarUrl} alt={UserName} />
      <span>{UserName}</span>
      {notificationCount && <span>{notificationCount}</span>}
    </div>
  );
}
```

`dashboard.tsx` uses it as `<UserAvatar notificationCount={3} />` — self-contained, composable.

## Layout composition

The layout (`main.tsx`) is mandatory. It renders `<html><head><body>` and receives route content as standard React `children`:

```tsx
// main.tsx
import { Session } from "@rstf/main";
import type { ReactNode } from "react";

export function View({ children }: { children: ReactNode }) {
  return (
    <html>
      <head><meta charSet="utf-8" /><title>My App</title></head>
      <body>
        {Session.isLoggedIn ? <NavBar /> : <LoginPrompt />}
        <main>{children}</main>
      </body>
    </html>
  );
}
```

The sidecar renders `<Layout><Route /></Layout>` — single render tree, single hydration root.

## Client-side: Hydration

After the server sends HTML, the browser hydrates it — React attaches event handlers and state to the existing DOM.

### How hydration works

The caller of `Renderer.Render` (not the renderer itself) is responsible for assembling the full HTML page. This includes:

1. Prepending `<!DOCTYPE html>` to the rendered HTML.
2. Injecting `<script>window.__RSTF_SERVER_DATA__ = {...}</script>` before `</body>`.
3. Injecting the hydration bundle `<script src="..."></script>` before `</body>`.

### Generated hydration entry

For each SSR route, the framework generates a hydration entry file:

```typescript
// Generated: .rstf/entries/dashboard.entry.tsx
import { hydrateRoot } from "react-dom/client";
import { View as Layout } from "../../main";
import { View as Route } from "../../routes/dashboard";

// Initialize server data for all generated modules used by this route.
// Each generated module reads its data from window.__RSTF_SERVER_DATA__.
import "@rstf/main";
import "@rstf/routes/dashboard";
import "@rstf/shared/ui/user-avatar";

hydrateRoot(
  document,
  <Layout>
    <Route />
  </Layout>
);
```

These are bundled by Bun into `.rstf/static/{route}/bundle.js`.

### Client-side initialization of generated modules

On the client, generated modules initialize from `window.__RSTF_SERVER_DATA__` at import time:

```typescript
// .rstf/generated/shared/ui/user-avatar.ts (generated — do not edit)
const _isServer = typeof window === "undefined";
const _initData = _isServer
  ? {}
  : ((window as any).__RSTF_SERVER_DATA__?.["shared/ui/user-avatar"] ?? {});

export let UserName: string = _initData.UserName ?? "";
export let AvatarUrl: string = _initData.AvatarUrl ?? "";

export function __setServerData(data: Record<string, any>) {
  UserName = data.UserName ?? "";
  AvatarUrl = data.AvatarUrl ?? "";
}
```

On the server, exports start as empty strings and are set by the sidecar before rendering. On the client, exports are initialized from the serialized data embedded in the HTML.

### Hydration sequence

1. Browser receives HTML — page is visible immediately.
2. Browser parses `__RSTF_SERVER_DATA__` — server data is in memory.
3. Browser loads `bundle.js` — React + component code loads.
4. Generated modules initialize from `__RSTF_SERVER_DATA__`.
5. `hydrateRoot` runs — React attaches to existing DOM.
6. `useEffect`, `useState`, `onClick`, etc. become active.

### Client-only APIs during SSR

React handles these automatically:

| API | Server | Client |
|-----|--------|--------|
| `useEffect` | Ignored | Runs after hydration |
| `useState` | Initial value only | Becomes interactive |
| `useRef` | Ignored | Attached after hydration |
| `onClick` | Not in HTML | Attached after hydration |

The full React ecosystem works — any library following React's hooks contract is SSR-compatible.

### Bundle generation

```
bun build .rstf/entries/dashboard.entry.tsx --outdir .rstf/static/dashboard/
```

Output includes React, ReactDOM, the component, and the hydration bootstrap.

### Static file serving

The Go server serves `.rstf/static/` so browsers can load bundles:

```
GET /.rstf/static/dashboard/bundle.js -> .rstf/static/dashboard/bundle.js
```

## Out of scope (MVP)

- Automatic dependency discovery (which `.go` files to call for a given route tree)
- Code splitting / shared React bundle across routes
- CSS extraction and injection
- Streaming SSR (`renderToPipeableStream`)
- Selective hydration / React Server Components
