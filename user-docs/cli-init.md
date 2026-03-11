# `rstf init`

`rstf init <name>` creates a new `rstf` app in a new directory. The public entrypoint is `npm create rstf@latest`.

## Usage

```bash
npm create rstf@latest my-app
npm create rstf@latest my-app -- --module github.com/acme/my-app
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

The generated app pins a released `github.com/rafbgarcia/rstf` Go module version and a matching local `@rstf/cli` package version.

## Flags

- `--module`: override the Go module path
- `--skip-install`: write files and generate `rstf/`, but skip `npm install` and `go mod tidy`

## Result

After a normal `npm create rstf@latest my-app`, the app is ready for:

```bash
cd my-app
npm run dev
```
