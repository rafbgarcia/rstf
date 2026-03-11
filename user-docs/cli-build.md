# `rstf build`

`rstf build` creates a deployable `dist/` directory from the app root. In a scaffolded app, use it through `npm run build` or `bun run build`.

## Usage

```bash
npm run build
```

## What It Produces

The build writes `dist/` with:

- the Go server binary
- `rstf/` generated files, client bundles, and SSR bundles
- client bundles and built CSS

The binary name matches the app directory name.

## Startup

From the app root:

```bash
npm run build
cd dist
./my-app
```

The production startup command is executing the Go binary from `dist/`.

## Build Steps

`rstf build` currently:

1. regenerates `rstf/`
2. bundles client assets
3. bundles per-route SSR entries for the embedded renderer
4. builds CSS when `main.css` exists
5. copies `rstf/` into `dist/`
6. builds the Go binary from `rstf/server_gen.go`

This is a deployable-directory workflow, not a single-binary workflow.
