# `rstf init`

`rstf init <name>` creates a new `rstf` app in a new directory.

## Usage

```bash
rstf init my-app
rstf init my-app --module github.com/acme/my-app
```

`<name>` is the destination directory and default app/module name.

## What It Creates

The scaffold is not a toy. It includes:

- `main.go` and `main.tsx`
- `routes/index`
- `routes/live-chat._id`
- `routes/users._id`
- `shared/ui/app-badge`
- `main.css` with Tailwind v4 setup
- `postcss.config.mjs`
- `package.json`, `tsconfig.json`, and `go.mod`
- an initial generated `rstf/` tree

The generated app is wired to the current local framework checkout with a `replace github.com/rafbgarcia/rstf => /path/to/rstf` entry in `go.mod`.
It also installs a local `@rstf/cli` package in `package.json` so the app can run `rstf` through npm scripts.

## Flags

- `--module`: override the Go module path
- `--skip-install`: write files and generate `rstf/`, but skip `npm install` and `go mod tidy`

## Result

After a normal `rstf init my-app`, the app is ready for:

```bash
cd my-app
npm run dev
```
