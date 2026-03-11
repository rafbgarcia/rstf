# `rstf dev`

`rstf dev` runs the local development loop from the app root. In a scaffolded app, use it through `npm run dev` or `bun run dev`.

## Usage

```bash
npm run dev
npm run dev -- --port 4000
```

## What It Does

On startup, `rstf dev`:

1. generates the `rstf/` tree
2. bundles client hydration entries into `rstf/static/`
3. bundles per-route SSR entries into `rstf/ssr/`
4. builds `main.css` when present
5. starts the generated Go server
6. watches `.go`, `.tsx`, and `main.css`

The default HTTP port is `3000`.

## Runtime Ownership

The dev runtime is app-owned:

- React and React DOM are bundled from the app's dependencies
- the `rstf` executable comes from the app's local `@rstf/cli` package, which installs the matching macOS/Linux binary during `npm install`
- generated files live in the app's `rstf/` directory
- the embedded renderer loads SSR bundles from `rstf/ssr/`

## Generated Output

During development, `rstf dev` keeps these areas up to date:

- `rstf/generated`
- `rstf/types`
- `rstf/entries`
- `rstf/ssr_entries`
- `rstf/routes`
- `rstf/ssr`
- `rstf/static`
- `rstf/server_gen.go`

Do not edit those files directly.
