_This project is under active development and not recommended for production use. APIs and conventions may change without notice._

# rstf

`rstf` is a Go-first framework for building server-rendered web apps with React islands, typed server data, and a tight local-to-production workflow.

## Current Workflow

Today the framework is designed to be used from a local checkout of this repository.

1. Install the `rstf` CLI from this repo.
2. Run `rstf init <name>` to create a full app.
3. Use `rstf dev` for the local loop.
4. Use `rstf build` to produce a deployable `dist/` directory.
5. Start production by executing the Go binary inside `dist/`.

Generated apps are currently wired back to your local checkout with a `replace github.com/rafbgarcia/rstf => /path/to/rstf` entry in `go.mod`. That is intentional while the framework is still greenfield.

## Prerequisites

- Go `1.24`
- Node.js with npm
- a local checkout of this repository

From the framework repo root:

```bash
go install ./cmd/rstf
```

## Quick Start

```bash
rstf init my-app
cd my-app
rstf dev
```

The generated app includes:

- a typed SSR home route
- a dynamic route using the `_id` convention
- a live query, mutation, and action demo
- a shared server-data component
- Tailwind v4 with a light default theme

Build output goes to `dist/`:

```bash
rstf build
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
