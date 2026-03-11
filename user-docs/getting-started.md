# Getting Started

This guide gets a new `rstf` app running with the current standalone-app workflow.

## Prerequisites

- Go `1.24`
- Node.js with npm
- a local checkout of this repository

Install the CLI from the framework repo root:

```bash
go install ./cmd/rstf
```

## Create An App

```bash
rstf init my-app
```

That command:

- creates `./my-app`
- writes `go.mod`, `package.json`, `tsconfig.json`, `main.go`, `main.tsx`, and demo routes
- installs the app's npm dependencies
- installs a local `@rstf/cli` package so the app can run `rstf` through npm scripts
- generates the initial `rstf/` tree
- leaves the app ready for `npm run dev`

By default the app's Go module matches the directory name. To override it:

```bash
rstf init my-app --module github.com/acme/my-app
```

The generated `go.mod` includes a local `replace` directive pointing back to this framework checkout, and `package.json` includes a local `@rstf/cli` dependency from this repo. That is the current intended workflow while `rstf` is still evolving.

The repo also includes the npm bootstrap packages at [packages/create-rstf](/Users/rafa/github.com/rafbgarcia/rstf/packages/create-rstf) and [packages/cli](/Users/rafa/github.com/rafbgarcia/rstf/packages/cli). The intended published entrypoint is `npm create rstf@latest`, but until those packages are published the supported local-checkout flow remains `rstf init`.

## Start The Dev Server

```bash
cd my-app
npm run dev
```

The generated demo includes:

- `/` for typed SSR and a shared server-data component
- `/live-chat/studio` for a live query, mutation, and action flow
- `/users/ada` for a dynamic route

The default server port is `3000`.

## Route Naming

Route folders are flat and dot-separated. Dynamic params use a leading underscore.

Examples:

- `routes/index/index.go` -> `/`
- `routes/users._id/index.go` -> `/users/{id}`
- `routes/users._id.edit/index.go` -> `/users/{id}/edit`
- `routes/dashboard.active-users/index.go` -> `/dashboard/active-users`

## Generated Files

`rstf/` is generated output. Important files and directories include:

- `rstf/generated/routes.ts`: TypeScript route helpers and live RPC descriptors
- `rstf/generated/<path>.ts`: generated runtime modules for route and shared server data
- `rstf/types/*.d.ts`: generated TypeScript types from Go data contracts
- `rstf/routes/routes_gen.go`: generated Go route helper package, imported as `your-module/rstf/routes`
- `rstf/server_gen.go`: generated Go server entrypoint
- `rstf/static/*`: client bundles and built CSS

Do not edit generated files directly.

## Styling

`rstf init` installs Tailwind v4 and generates a light default theme in `main.css`.

When `main.css` exists:

- `npm run dev` builds and serves `rstf/static/main.css`
- `npm run build` includes the built CSS in `dist/rstf/static/main.css`

If `postcss.config.mjs` exists, `rstf` runs PostCSS. Otherwise `main.css` is copied as-is.

## Build For Deployment

From the app root:

```bash
npm run build
cd dist
./my-app
```

`rstf build` creates a deployable directory that contains:

- the Go binary
- the generated `rstf/` tree
- client assets
- the app source files the runtime still expects
- the app's `node_modules`

The startup command is executing the Go binary from `dist/`.

## Next Steps

- [CLI: init](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/cli-init.md)
- [CLI: dev](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/cli-dev.md)
- [CLI: build](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/cli-build.md)
- [Routing and Server Data](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/routing-and-server-data.md)
- [Live Queries](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/live-queries.md)
