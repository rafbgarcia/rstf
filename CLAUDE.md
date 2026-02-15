# rstf framework

A Go + React framework with file-based routing and end-to-end type safety.

Developers write Go route handlers that return typed data, and React components that receive it as props.

The framework generates TypeScript types from Go structs and handles SSR via a Bun sidecar. Client-side hydration is planned but not yet implemented.

The CLI (`rstf dev`) orchestrates codegen and starts the server. `rstf init` and `rstf build` are not yet implemented.

## Understanding the codebase

Read `ARCHITECTURE.md` for a high-level overview of all components and how they connect, including a request flow diagram.

Key modules:

- `internal/codegen/` — Parses Go AST, generates TypeScript types (`.d.ts`), runtime modules (`serverData()` + `__setServerData()`), and server entry point (`.rstf/server_gen.go`)
- `internal/renderer/` — Go client + Bun sidecar for SSR via `renderToString`
- `internal/conventions/` — File conventions, route path resolution rules
- `internal/watcher/` — File change monitoring for live reload (not yet implemented)
- `cmd/rstf/` — CLI entry point: `dev` is implemented; `init` and `build` are not yet implemented
- `runtime/ssr.ts` — Bun HTTP server that executes React SSR
- `context.go`, `logger.go` — Request-scoped context and structured logging

## Design philosophy

**Low cognitive load**
**One obvious way to do it**
**Conventions over configuration**
**Opinionated design**

**Why Go for the server:**

- Static typing catches errors at compile time, which is especially valuable with AI-generated code.
- Go's small language surface and explicit style make it predictable for both LLMs and humans new to the codebase.
- Mature, consistent tooling (go fmt, go vet, go test, gopls) with no ecosystem fragmentation.

**Why SSR (via Bun sidecar):**

- Server-side auth checks before any HTML is sent — no flash of unauthorized content, no client-side auth loading states.
- Most business applications are read-heavy; SSR gives faster first paint with less client-side complexity.
- The Bun sidecar runs as a separate process. This adds operational complexity but avoids CGO dependencies and gives full React/JSX compatibility.

## Specification-driven development

This project uses `*-spec.md` files co-located with their modules as the source of truth.
Search for `*-spec.md` to find all specifications.

**Rules for spec files:**

- Each spec documents how its own component works — it must not duplicate information from other specs.
- When a spec needs to reference a concept owned by another spec, it cross-references with a "see `path/to/other-spec.md`" link.
- Specs document the current state of the code. Planned/unimplemented features must be clearly marked as such.

**Workflow for any code change:**

1. Read the relevant spec(s) first.
2. Update the spec to reflect the intended change.
3. Implement the code to match the spec.

Note: if code and spec disagree, the code must be updated to match the spec.
