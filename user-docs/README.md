# rstf User Docs

`rstf` is a Go-first framework for server-rendered web apps with React islands, typed server data, and live queries.

These docs describe the framework as it exists in this repository today. The workflow is real, but it is still greenfield and intentionally evolving.

## Start Here

- [Getting Started](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/getting-started.md)
- [CLI: init](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/cli-init.md)
- [CLI: dev](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/cli-dev.md)
- [CLI: build](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/cli-build.md)
- [Routing and Server Data](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/routing-and-server-data.md)
- [Live Queries](/Users/rafa/github.com/rafbgarcia/rstf/user-docs/live-queries.md)

## Current Contract

- `rstf init <name>` creates a full app with app-owned runtime dependencies.
- Generated apps install a local `@rstf/cli` package and prefer `npm run dev` / `npm run build`.
- Generated artifacts live in `rstf/`.
- Dynamic route params use `_name` on disk, for example `routes/users._id/index.go`.
- `rstf dev` runs the app from the app's own dependencies and generated files.
- `rstf build` writes a deployable `dist/` directory.
- Production startup is executing the Go binary inside `dist/`.

## Status

- Go `1.24` is the baseline.
- Node.js and npm are required for app runtime dependencies and CSS processing.
- The generated app currently points back to your local framework checkout through both `go.mod` and the local `@rstf/cli` package dependency.
- The repo includes `packages/create-rstf` and `packages/cli` as the npm-native bootstrap path; the intended published entrypoint is `npm create rstf@latest`.
- APIs, conventions, and generated output may still change.
