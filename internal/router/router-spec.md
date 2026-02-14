# Router Specification

## Overview

The router maps file paths to HTTP routes and dispatches requests to Go handler functions. It uses file-based routing under a `routes/` directory, with optional `.go` files providing server-side data to React components.

Because Go is statically typed, the framework generates Go source code (`.rstf/server_gen.go`) that imports the user's root package and route packages, wraps their handler functions, and registers HTTP handlers. The generated file is the Go entry point (`func main()`).

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
        user-avatar.tsx              # Component with server data
        user-avatar.go
      button.tsx                     # Standalone component (no .go)
    hooks/
      some-hook.ts                   # Shared TypeScript code
  .rstf/                             # Generated (gitignored)
    server_gen.go                    # Generated entry point (package main)
    types/
      index.d.ts
      dashboard.d.ts
      users-id-edit.d.ts
    static/
      index/bundle.js
      dashboard/bundle.js
```

The user's `main.go` uses the app's package name (e.g. `package myapp`), not `package main`. This makes it importable by the generated `.rstf/server_gen.go`, which declares `package main` and contains `func main()`. Go prohibits importing `package main`, but any other package name works. The `rstf dev` CLI compiles `.rstf/server_gen.go` explicitly (Go's build tool skips dot-prefixed directories by default).

## File conventions

### Layout (`main.go` + `main.tsx`)

`main.go` provides layout-level server data (e.g. session, auth) available to `main.tsx` on every request. It uses the app's package name derived from `go.mod`.

`main.tsx` is the root React component. It wraps all route components via `children` and can switch between layouts based on the server data (e.g. logged-in vs logged-out).

```go
package myapp

import rstf "github.com/rafbgarcia/rstf"

type Session struct {
    IsLoggedIn bool   `json:"isLoggedIn"`
    UserName   string `json:"userName"`
}

// SSR provides layout data to main.tsx on every request.
// Can also handle redirects via ctx.
func SSR(ctx *rstf.Context) (session Session) {
    // Check auth, load session
    return
}
```

### Routes (`routes/` directory)

All routes use the **folder convention** — each route is a folder under `routes/` containing at least an `index.tsx`. This enforces a single way to define routes and gives each route its own Go package when a `.go` file is present.

```
routes/
  index/
    index.tsx                   # GET /
    index.go                    # Server data (package index)
  dashboard/
    index.tsx                   # GET /dashboard
    index.go                    # Server data (package dashboard)
  users.$id.edit/
    index.tsx                   # GET /users/{id}/edit
    index.go                    # Server data (package useredit)
    UserEditForm.tsx            # Colocated component (not a route)
  about/
    index.tsx                   # GET /about (no .go — no server data)
```

Rules:

- Each route folder contains `index.tsx` (required) and optionally `index.go`.
- The folder name determines the URL path (see Route path resolution).
- Additional `.tsx` files in the folder are colocated components, NOT separate routes.
- **No nested folder routes.** URL nesting is expressed with dots in the folder name: `routes/users.$id.edit/`, NOT `routes/users/$id/edit/`.

> **Why folder-only?** Go requires all `.go` files in a directory to share the same package name. Multiple `.go` files in `routes/` would all be `package routes` and couldn't each export `func SSR()`. Folders give each route its own package. Using folders for ALL routes (even those without `.go`) keeps one consistent convention.

### Dynamic segments

`$` in a folder name denotes a dynamic URL parameter:

```
routes/users.$id/                -> /users/{id}
routes/users.$id.edit/           -> /users/{id}/edit
routes/posts.$slug/              -> /posts/{slug}
```

Dynamic parameters are accessed via `ctx.Request.PathValue("id")` in the Go handler, using Go 1.22+ ServeMux pattern matching.

### Shared components

Components outside `routes/` are shared — they can be imported by any route or other component.

Shared components that need server data must be in their own directory (same Go package constraint):

```
shared/ui/user-avatar/
  user-avatar.tsx              # Component
  user-avatar.go               # Server data (package useravatar)
```

Standalone components without server data can live anywhere:

```
shared/ui/button.tsx             # No .go file needed
shared/hooks/some-hook.ts        # Shared TypeScript code
```

### Go file pairing rules

Any `.tsx` file can have a paired `.go` file that provides server data. The `.go` file must be in the **same directory** as the `.tsx` file and export a recognized handler function (`SSR`, `GET`, etc.).

## Route path resolution

Folder names in `routes/` are converted to URL patterns:

1. Strip the `routes/` prefix.
2. Use the folder name (ignore files inside).
3. Split the folder name on `.` to get path segments.
4. Replace `$param` segments with `{param}` (Go 1.22 ServeMux syntax).
5. The folder name `index` maps to `/`.

| Folder | URL pattern |
|--------|-------------|
| `routes/index/` | `GET /` |
| `routes/dashboard/` | `GET /dashboard` |
| `routes/about/` | `GET /about` |
| `routes/users.$id/` | `GET /users/{id}` |
| `routes/users.$id.edit/` | `GET /users/{id}/edit` |
| `routes/posts.$slug/` | `GET /posts/{slug}` |

## Static import analysis

The framework needs to know which `.go` files to call for each route. A route's component may import shared components that have their own `.go` files. The framework discovers these dependencies by statically analyzing TSX/TS imports at codegen time.

### How it works

1. For each route, parse its `index.tsx` for `import` statements.
2. For each local import (relative paths like `./` or `../`, not bare specifiers like `react`), resolve the file path.
3. Check if a `.go` file exists in that file's directory (indicating server data).
4. If yes, record it as a server data dependency for this route.
5. Recursively scan the imported file's imports (with cycle detection via a visited-files set).
6. Always include `main.go` — the layout runs on every request.

### Import parsing

Imports are extracted with a regex scan of `from` clauses — no full TS parser needed:

```
from ['"](\./|\.\./)...['"]
```

Bare specifiers (`react`, `@rstf/...`) are skipped. Only local relative imports are followed.

### Output

The analysis produces a dependency map — for each route, the list of component paths that have `.go` files:

```
GET /                -> [main, routes/index]
GET /dashboard       -> [main, routes/dashboard, shared/ui/user-avatar]
GET /users/{id}/edit -> [main, routes/users.$id.edit, shared/ui/user-avatar]
```

This tells the generated code which `SSR` functions to call per request.

## Generated code (`.rstf/server_gen.go`)

The codegen produces `.rstf/server_gen.go` — the Go entry point. It declares `package main`, imports the user's root package and all route/shared-component packages, and wires HTTP handlers.

```go
// Code generated by rstf. DO NOT EDIT.
package main

import (
    "fmt"
    "net/http"

    rstf "github.com/rafbgarcia/rstf"
    "github.com/rafbgarcia/rstf/internal/renderer"

    app "github.com/user/myapp"
    routeindex "github.com/user/myapp/routes/index"
    dashboard "github.com/user/myapp/routes/dashboard"
    useredit "github.com/user/myapp/routes/users.$id.edit"
    useravatar "github.com/user/myapp/shared/ui/user-avatar"
)

func main() {
    r := renderer.New()
    r.Start(".")

    // GET / — layout + route server data
    http.HandleFunc("GET /", func(w http.ResponseWriter, req *http.Request) {
        ctx := rstf.NewContext(req)
        session := app.SSR(ctx)
        featured := routeindex.SSR(ctx)

        html, err := r.Render(renderer.RenderRequest{
            Component: "routes/index",
            Layout:    "main",
            ServerData: map[string]map[string]any{
                "main":         {"session": session},
                "routes/index": {"featured": featured},
            },
        })
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        fmt.Fprint(w, html)
    })

    // GET /dashboard — layout + route + shared component
    http.HandleFunc("GET /dashboard", func(w http.ResponseWriter, req *http.Request) {
        ctx := rstf.NewContext(req)
        session := app.SSR(ctx)
        posts, author := dashboard.SSR(ctx)
        userName, avatarUrl := useravatar.SSR(ctx)

        html, err := r.Render(renderer.RenderRequest{
            Component: "routes/dashboard",
            Layout:    "main",
            ServerData: map[string]map[string]any{
                "main":                  {"session": session},
                "routes/dashboard":      {"posts": posts, "author": author},
                "shared/ui/user-avatar": {"userName": userName, "avatarUrl": avatarUrl},
            },
        })
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        fmt.Fprint(w, html)
    })

    // GET /users/{id}/edit — layout + route + shared component
    http.HandleFunc("GET /users/{id}/edit", func(w http.ResponseWriter, req *http.Request) {
        ctx := rstf.NewContext(req)
        session := app.SSR(ctx)
        user, roles := useredit.SSR(ctx)
        userName, avatarUrl := useravatar.SSR(ctx)

        html, err := r.Render(renderer.RenderRequest{
            Component: "routes/users.$id.edit",
            Layout:    "main",
            ServerData: map[string]map[string]any{
                "main":                  {"session": session},
                "routes/users.$id.edit": {"user": user, "roles": roles},
                "shared/ui/user-avatar": {"userName": userName, "avatarUrl": avatarUrl},
            },
        })
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        fmt.Fprint(w, html)
    })

    http.ListenAndServe(":3000", nil)
}
```

### Key details

- **User's package is importable**: `main.go` uses the app's package name (e.g. `package myapp`), not `package main`. The generated code imports it (e.g. `app "github.com/user/myapp"`) and calls `app.SSR(ctx)` for layout data.
- **Generated file is the entry point**: `.rstf/server_gen.go` has `package main` and `func main()`. The `rstf dev` CLI compiles and runs it.
- **Layout always runs**: `app.SSR()` is called on every request, before route-specific handlers.
- **Go 1.22+ ServeMux**: Uses method-and-pattern routing (e.g. `"GET /users/{id}/edit"`).
- **Import analysis drives wiring**: Only `.go` files discovered via static import analysis (plus the route's own `.go` and the layout) are called.
- **Named return values -> `map[string]any`**: Keys match Go param names, which match generated TypeScript prop names.
- **`ServerData` keyed by component path**: Matches the renderer's ES module live binding model (see `renderer-spec.md`).
- **Dynamic params**: Accessed via `ctx.Request.PathValue("id")` in user code.
- **Raw HTML response**: The renderer returns HTML, the handler writes it directly. Page shell assembly (DOCTYPE, hydration scripts) is the caller's responsibility.

## How the codegen produces this

The `codegen` package (`internal/codegen/codegen.go`) already parses Go files and extracts:

- Relative directory path (`RouteFile.Dir`)
- Go package name (`RouteFile.Package`)
- Function names and signatures (`RouteFile.Funcs` — each `RouteFunc` has `Name`, `Params`, `HasContext`)
- Named return parameters (`RouteParam` with `Name`, `Type`, `IsSlice`)
- Referenced struct definitions (`RouteFile.Structs`)

A `GenerateServer` function (not yet implemented) will combine:

1. The module path from the user's `go.mod`
2. The parsed `[]RouteFile` from `codegen.ParseDir` (for route and shared-component `.go` files, plus `main.go`)
3. The dependency map from the static import analysis
4. Route path resolution (folder name -> URL pattern)

to produce `.rstf/server_gen.go`.

## Handler function dispatch

The framework recognizes these exported function names and maps them to HTTP methods:

| Go function | HTTP method | Behavior |
|-------------|-------------|----------|
| `SSR` | GET | Calls function, renders React component via Bun sidecar, returns HTML |
| `GET` | GET | Calls function, returns JSON (future) |
| `POST` | POST | Calls function, returns JSON (future) |
| `PUT` | PUT | Calls function, returns JSON (future) |
| `DELETE` | DELETE | Calls function, returns JSON (future) |

For the MVP, only `SSR` is implemented.
