_This project is under active development and not recommended for production use. APIs and conventions may change without notice._

# rstf

`rstf` is a Go-first framework for building server-rendered web apps with React islands, typed server data, and a tight local-to-production workflow.

## Current Workflow

Today the intended public workflow is:

1. Run `npm create rstf@latest <name>` to scaffold a full app.
2. Use `npm run dev` for the local loop.
3. Use `npm run build` to produce a deployable `dist/` directory.
4. Start production by executing the Go binary inside `dist/`.

## Prerequisites

- Go `1.24.x`
- Node.js `24.x`

## Quick Start

```bash
npm create rstf@latest my-app
cd my-app
npm run dev
```

`bunx create-rstf@latest my-app` also works, and scaffolded apps run through `bun run dev` / `bun run build` because they use local package scripts.

The generated app includes:

- a typed SSR home route
- a dynamic route using the `_id` convention
- a live query, mutation, and action demo
- a shared server-data component
- Tailwind v4 with a light default theme

Build output goes to `dist/`:

```bash
npm run build
cd dist
./my-app
```

## Docs

- [User Docs](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/README.md)
- [Getting Started](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/getting-started.md)
- [CLI: init](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/cli-init.md)
- [CLI: dev](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/cli-dev.md)
- [CLI: build](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/cli-build.md)
- [Routing and Server Data](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/routing-and-server-data.md)
- [Live Queries](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/live-queries.md)
