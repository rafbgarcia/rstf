# Runtime Specification

The `runtime/` directory contains the JavaScript/TypeScript code that runs outside of Go — both server-side (Bun sidecar) and client-side (hydration in the browser).

## Server-side: SSR sidecar (`runtime/ssr.ts`)

A Bun HTTP server that the Go renderer communicates with. See `internal/renderer/renderer-spec.md` for the full protocol (request/response format, startup, component resolution, concurrency model, and module caching).

## Server data mechanism

Server data is the bridge between Go handlers and React components. The Go side calls `SSR()` handlers, serializes the returned structs, and sends them to the Bun sidecar which calls `__setServerData()` before rendering (see `internal/codegen/codegen-spec.md` for the full pipeline and generated module format, and `internal/renderer/renderer-spec.md` for the render protocol).

This section documents how components consume server data at runtime.

### Why `serverData()` is a function

The generated module is cached by the Bun sidecar across requests. A function call ensures the component reads the current request's data at render time, not stale data from import time. Internally, `__setServerData()` updates the module's `_data` variable before each `renderToString` call. See `internal/renderer/renderer-spec.md` for concurrency safety details.

### Scoping

Each generated module is independent. `@rstf/shared/ui/user-avatar` only contains user avatar data. `@rstf/routes/dashboard` only contains dashboard data. A component imports `serverData` from its own generated module — no cross-access.

### Mixed data sources

A component can use both server data AND React props:

- **Server data** (from `.go` file): accessed via `serverData()` from `@rstf/{path}` — for auth, database queries, session info
- **React props** (from parent component): passed as regular JSX attributes — for component-to-component communication

```tsx
// shared/ui/user-avatar/index.tsx
import { serverData } from "@rstf/shared/ui/user-avatar";

export function View({ notificationCount }: { notificationCount?: number }) {
  const { userName, avatarUrl } = serverData();
  return (
    <div>
      <img src={avatarUrl} alt={userName} />
      <span>{userName}</span>
      {notificationCount && <span>{notificationCount}</span>}
    </div>
  );
}
```

`dashboard.tsx` uses it as `<UserAvatar notificationCount={3} />` — self-contained, composable.

## Layout composition

The layout component (`main.tsx`) receives route content as standard React `children` (see `internal/conventions/conventions-spec.md` for the layout file convention):

```tsx
// main.tsx
import { serverData } from "@rstf/main";
import type { ReactNode } from "react";

export function View({ children }: { children: ReactNode }) {
  const { session } = serverData();
  return (
    <html>
      <head><meta charSet="utf-8" /><title>My App</title></head>
      <body>
        {session.isLoggedIn ? <NavBar /> : <LoginPrompt />}
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

The generated `server_gen.go` calls `assemblePage()` after rendering to assemble the full HTML page. This includes:

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

On the client, generated modules initialize `_data` from `window.__RSTF_SERVER_DATA__` at import time (instead of waiting for `__setServerData()`). The codegen produces a dual-mode version of the generated module (see `internal/codegen/codegen-spec.md` for the module format) that detects the environment via `typeof window !== "undefined"`.

On the server, `_data` starts as an empty object and is set by the sidecar before rendering. On the client, `_data` is initialized from the serialized data embedded in the HTML. Either way, `serverData()` reads from `_data` at call time, returning the current request's values.

### Hydration sequence

1. Browser receives HTML — page is visible immediately.
2. Browser parses `__RSTF_SERVER_DATA__` — server data is in memory.
3. Browser loads `bundle.js` — React + component code loads.
4. Generated modules initialize `_data` from `__RSTF_SERVER_DATA__`.
5. `hydrateRoot` runs — React attaches to existing DOM, components call `serverData()` which reads from `_data`.
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

## Page metadata (not yet implemented)

Routes will be able to export a `meta` value to set per-page `<title>`, description, and other head tags. The layout reads it via its own `serverData()`.

### Static meta (no server data needed)

```tsx
// routes/settings.tsx
export const meta = {
  title: "Settings",
  description: "Manage your account settings",
};
```

### Dynamic meta (depends on server data)

Export `meta` as a function. The sidecar calls it after `__setServerData`, so `serverData()` returns current values:

```tsx
// routes/settings.tsx
import { serverData } from "@rstf/routes/settings";

export const meta = () => ({
  title: `Editing ${serverData().userName}`,
});
```

### How the sidecar uses meta

After importing the route module, the sidecar reads `mod.meta`:
- If it's a function, call it (after `__setServerData`) to get the meta object.
- If it's a plain object, use it directly.
- Merge the result into the layout's server data so the layout can render `<title>` and `<meta>` tags.

## Out of scope (MVP)

- Automatic dependency discovery (which `.go` files to call for a given route tree)
- Code splitting / shared React bundle across routes
- CSS extraction and injection
- Streaming SSR (`renderToPipeableStream`)
- Selective hydration / React Server Components
