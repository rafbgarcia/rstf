# `rstf dev`

`rstf dev` runs the local development loop from the app root.

## Usage

```bash
rstf dev
rstf dev --port 4000
```

## What It Does

On startup, `rstf dev`:

1. generates the `rstf/` tree
2. bundles client hydration entries into `rstf/static/`
3. builds `main.css` when present
4. starts the generated Go server
5. starts the React SSR sidecar using the app's own `node_modules`
6. watches `.go`, `.tsx`, and `main.css`

The default HTTP port is `3000`.

## Runtime Ownership

The dev runtime is app-owned:

- the SSR sidecar resolves `tsx` from the app's `node_modules`
- React and React DOM come from the app
- generated files live in the app's `rstf/` directory

That means a scaffolded app can run without relying on this repo's `node_modules`.

## Generated Output

During development, `rstf dev` keeps these areas up to date:

- `rstf/generated`
- `rstf/types`
- `rstf/entries`
- `rstf/routes`
- `rstf/static`
- `rstf/server_gen.go`

Do not edit those files directly.
