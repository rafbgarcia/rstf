# Routing and Server Data

`rstf` uses file-based routing for page components and Go functions for server behavior.

## Route Directories

A route lives under `routes/` and is identified by its directory name.

Examples:

- `routes/index` -> `/`
- `routes/get-vs-ssr` -> `/get-vs-ssr`
- `routes/users._id` -> `/users/{id}`

Dynamic segments use `_name` in the directory name.

## Route Files

A route can have:

- `index.tsx`: the React view
- `index.go`: Go server functions for the route

## SSR Data

To server-render typed data into a route, export `SSR` from the route's Go file and return a struct.

```go
package dashboard

import rstf "github.com/rafbgarcia/rstf"

type ServerData struct {
	Message string `json:"message"`
}

func SSR(ctx *rstf.Context) ServerData {
	return ServerData{Message: "Welcome"}
}
```

Then consume it in `index.tsx`:

```tsx
import { serverData } from "@rstf/routes/dashboard";

export function View() {
  const { message } = serverData();
  return <h1>{message}</h1>;
}
```

The `serverData()` function is generated from the Go `SSR` return type.

## JSON Handlers

Routes can also export HTTP verb handlers:

```go
func GET(ctx *rstf.Context) error
func POST(ctx *rstf.Context) error
func PUT(ctx *rstf.Context) error
func PATCH(ctx *rstf.Context) error
func DELETE(ctx *rstf.Context) error
```

Example:

```go
type APIResponse struct {
	Source string `json:"source"`
	Route  string `json:"route"`
}

func GET(ctx *rstf.Context) error {
	return ctx.JSON(200, APIResponse{
		Source: "get",
		Route:  "/dashboard",
	})
}
```

These handlers are for normal request/response HTTP behavior. They are separate from the newer live query RPC model.

## Layouts and Shared Components

`main.go` and `main.tsx` define the app layout.

Shared UI can also have Go-backed server data. For example, a shared component under `shared/ui/user-avatar` can export `SSR` in Go and consume `serverData()` in TSX.

That allows typed server data to flow into both routes and shared components.

## Route Helpers

Type-safe route helpers are generated in TypeScript and Go.

TypeScript:

```tsx
import { routes } from "@rstf/routes";

routes["users._id"].url({ id: "123" });
```

Go:

```go
import "example.com/my-app/rstf/routes"

url := routes.UsersDotId.URL(routes.UsersDotIdParams{Id: "123"})
```

## App Startup Hooks

The layout Go package can also export:

```go
func OnServerStart(app *rstf.App)
func AroundRequest() []rstf.Middleware
```

Use `OnServerStart` to configure the app at startup, for example:

- database setup
- request body limit
- admission control settings

Use `AroundRequest` for request middleware.
