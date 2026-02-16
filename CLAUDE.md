# rstf framework

A Go + React framework with file-based routing and end-to-end type safety.

Developers write Go route handlers that return typed data, and React components that receive it as props.

The framework generates TypeScript types from Go structs and handles SSR via a Bun sidecar. Client-side hydration is planned but not yet implemented.

The CLI (`rstf dev`) orchestrates codegen and starts the server. `rstf init` and `rstf build` are not yet implemented.

## Understanding the codebase

See `ARCHITECTURE.md` for a request flow diagram showing how all components connect.

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

This project uses `*-spec.md` files co-located with their modules. Specs enable design-level review — teammates review spec diffs instead of code diffs.

Search for `*-spec.md` to find all specifications.

**What specs document:**

- Contracts between modules (request/response formats, expected inputs/outputs)
- Rules, constraints, and invariants (e.g. "SSR must return a single struct")
- Design rationale — the "why" behind non-obvious decisions
- Generated output formats — what codegen should produce

**What specs do NOT document:**

- Step-by-step descriptions of what code does — that's what the code is for
- Go/TypeScript interface definitions that are already in the source — tests verify correctness
- Full code examples that mirror the implementation

**Rules:**

- Each spec documents its own module only — no duplication across specs.
- Cross-reference other specs with "see `path/to/other-spec.md`" links.
- Keep specs concise: a spec diff should be reviewable at a glance.

**Workflow for any code change:**

1. Read the relevant spec(s) first.
2. Implement the code changes.
3. Update the relevant spec(s), if any.
