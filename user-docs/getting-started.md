# Getting Started

This guide gets a small `rstf` app running in dev mode with:

- a Go layout
- a React layout component
- a server-rendered route
- a live query route

## What Exists Today

`rstf` is still greenfield. The current user-facing workflow is:

1. create the project structure manually
2. install the app's React dependencies
3. run `rstf dev`

Today, `rstf` assumes you are working from a local checkout of the framework repository. The development runtime uses the framework repo's own `node_modules` for the SSR sidecar and bundling pipeline.

## Prerequisites

- Go `1.24.6`
- Node.js
- npm
- a local checkout of this repository with its root dependencies installed

From the framework repo root:

```bash
npm install
go install ./cmd/rstf
```

That gives you the local `rstf` CLI used in the rest of this guide.

## Minimal Project Layout

A minimal app looks like this:

```text
my-app/
  go.mod
  package.json
  tsconfig.json
  main.go
  main.tsx
  routes/
    hello/
      index.go
      index.tsx
```

## go.mod

```go
module example.com/my-app

go 1.24.6

require github.com/rafbgarcia/rstf v0.0.0

replace github.com/rafbgarcia/rstf => /absolute/path/to/rstf
```

Use a local `replace` while the framework is still moving quickly.

## package.json

Your app needs React. A minimal `package.json` is:

```json
{
  "private": true,
  "dependencies": {
    "react": "^19.1.0",
    "react-dom": "^19.1.0"
  },
  "devDependencies": {
    "@types/react": "^19.1.0"
  }
}
```

## tsconfig.json

`rstf` generates `.rstf/generated/*` modules for route helpers and typed server data. Add the generated paths and includes:

```json
{
  "compilerOptions": {
    "jsx": "react-jsx",
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "node",
    "lib": ["DOM", "DOM.Iterable", "ES2022"],
    "noImplicitAny": true,
    "paths": {
      "@rstf/*": ["./.rstf/generated/*"]
    }
  },
  "include": [".rstf/types", ".rstf/generated/**/*.ts", "**/*.ts", "**/*.tsx"]
}
```

## Layout Files

`main.go` is the Go layout module.

```go
package myapp

import rstf "github.com/rafbgarcia/rstf"

type ServerData struct {
	AppName string `json:"appName"`
}

func SSR(ctx *rstf.Context) ServerData {
	return ServerData{AppName: "My App"}
}
```

`main.tsx` is the React layout component.

```tsx
import { serverData } from "@rstf/main";
import type { ReactNode } from "react";

export function View({ children }: { children: ReactNode }) {
  const { appName } = serverData();

  return (
    <html>
      <head>
        <title>{appName}</title>
      </head>
      <body>
        <main>{children}</main>
      </body>
    </html>
  );
}
```

## First Route

`routes/hello/index.go`

```go
package hello

import rstf "github.com/rafbgarcia/rstf"

type ServerData struct {
	Message string `json:"message"`
}

func SSR(ctx *rstf.Context) ServerData {
	return ServerData{Message: "Hello from Go"}
}
```

`routes/hello/index.tsx`

```tsx
import { serverData } from "@rstf/routes/hello";

export function View() {
  const { message } = serverData();
  return <h1>{message}</h1>;
}
```

## Run The App

From your app root:

```bash
rstf dev
```

That does the current dev loop:

- generates `.rstf/`
- bundles route entries
- starts the Go server
- starts the React SSR sidecar
- watches `.go`, `.tsx`, and `main.css`

The default server port is `3000`.

## CSS

If the app has a `main.css`, `rstf dev` will build and serve it automatically.

- if `postcss.config.mjs` exists, `rstf` runs PostCSS
- otherwise `main.css` is copied as-is to `.rstf/static/main.css`

## Generated Files

The `.rstf/` directory is generated output. Important pieces are:

- `.rstf/generated/routes.ts`: TypeScript route helpers and live RPC descriptors
- `.rstf/generated/<path>.ts`: generated runtime modules
- `.rstf/types/*.d.ts`: generated TypeScript namespaces from Go types
- `.rstf/routes/routes_gen.go`: generated Go route and query helper package, imported as `your-module/.rstf/routes`
- `.rstf/server_gen.go`: generated Go server entrypoint

Do not edit generated files directly.

## Next Steps

- [Routing and Server Data](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/routing-and-server-data.md)
- [Live Queries](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/live-queries.md)
