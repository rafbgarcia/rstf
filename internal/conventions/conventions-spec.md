# Conventions Specification

## Overview

The conventions module defines the file conventions and rules that map the project's directory structure to HTTP routes. See `internal/codegen/codegen-spec.md` for how these conventions are used to generate the server entry point (`.rstf/server_gen.go`).

For the overall project structure (where `main.go`, `routes/`, `shared/` live), see `ARCHITECTURE.md`.

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
func SSR(ctx *rstf.Context) Session {
    // Check auth, load session
    return Session{}
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

Any `.tsx` file can have a paired `.go` file that provides server data. The `.go` file must be in the **same directory** as the `.tsx` file and export `SSR`.

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

## Handler functions

The framework recognizes the exported function name `SSR` and maps it to `GET` requests. The generated server calls `SSR()`, passes the returned struct to the Bun sidecar, and returns the rendered HTML.

| Go function | HTTP method | Behavior |
|-------------|-------------|----------|
| `SSR` | GET | Calls function, renders React component via Bun sidecar, returns HTML |

Additional HTTP method handlers (`GET` for JSON, `POST`, `PUT`, `DELETE`) are planned but not yet designed. They will be specified in a separate document when implemented.
