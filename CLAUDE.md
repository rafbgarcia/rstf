# rstf framework

A Go + React framework with file-based routing and end-to-end type safety.

Developers write Go route handlers that return typed data, and React components that receive it as props.

The framework generates TypeScript types from Go structs, handles SSR via a Bun sidecar, and hydrates on the client.

The CLI (`rstf init`, `rstf dev`) orchestrates codegen, bundling, and live reloading.

## Design philosophy

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

**Workflow for any code change:**

1. Read the relevant spec(s) first.
2. Update the spec to reflect the intended change.
3. Implement the code to match the spec.

Note: if code and spec disagree, the code must be updated to match the spec.
