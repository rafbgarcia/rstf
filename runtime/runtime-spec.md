# Runtime Specification

The `runtime/` directory contains the JavaScript/TypeScript code that runs outside of Go — both server-side (Bun sidecar) and client-side (hydration in the browser).

## Server-side: SSR sidecar (`runtime/ssr.ts`)

A Bun HTTP server that the Go renderer communicates with. See `internal/renderer/renderer-spec.md` for the full protocol (request/response format, startup, component resolution).

In summary:
- Receives `{ component, props }` via POST.
- Dynamically imports the component `.tsx` file.
- Calls `ReactDOMServer.renderToString(<View {...props} />)`.
- Returns the HTML string.

### Module cache

The sidecar caches imported modules for performance. During development, the watcher signals the sidecar to clear its cache when `.ts`/`.tsx` files change, so the next render picks up the new code.

## Client-side: Hydration

After the server sends HTML, the browser needs to hydrate it — React attaches event handlers and state to the existing DOM.

### Generated hydration entry

For each SSR route, the framework generates a hydration entry file:

```typescript
// Generated: .rstf/entries/dashboard.entry.tsx
import { hydrateRoot } from "react-dom/client";
import { View } from "../../dashboard/dashboard";

const props = (window as any).__RSTF_PROPS__;
hydrateRoot(document.getElementById("root")!, <View {...props} />);
```

These are bundled by Bun into `.rstf/static/{route}/bundle.js`.

### What the browser receives

```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
</head>
<body>
  <div id="root">
    <!-- Server-rendered HTML -->
    <div><p>Hello World</p><p>Draft Post</p></div>
  </div>

  <script>
    window.__RSTF_PROPS__ = {"posts":[...]};
  </script>
  <script src="/.rstf/static/dashboard/bundle.js"></script>
</body>
</html>
```

### Hydration sequence

1. Browser receives HTML — page is visible immediately.
2. Browser parses `__RSTF_PROPS__` — props are in memory.
3. Browser loads `bundle.js` — React + component code loads.
4. `hydrateRoot` runs — React attaches to existing DOM.
5. `useEffect`, `useState`, `onClick`, etc. become active.

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

- Code splitting / shared React bundle across routes
- CSS extraction and injection
- Streaming SSR (`renderToPipeableStream`)
- Selective hydration / React Server Components
