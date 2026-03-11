# `rstf build`

`rstf build` creates a deployable `dist/` directory from the app root.

## Usage

```bash
rstf build
```

## What It Produces

The build writes `dist/` with:

- the Go server binary
- `rstf/` generated files
- client bundles and built CSS
- the app files the runtime expects at startup
- the app's `node_modules`

The binary name matches the app directory name.

## Startup

From the app root:

```bash
rstf build
cd dist
./my-app
```

The production startup command is executing the Go binary from `dist/`.

## Build Steps

`rstf build` currently:

1. regenerates `rstf/`
2. bundles client assets
3. builds CSS when `main.css` exists
4. copies the project into `dist/`
5. builds the Go binary from `rstf/server_gen.go`

This is a deployable-directory workflow, not a single-binary workflow.
