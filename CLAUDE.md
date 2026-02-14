# rstf framework

A Go + React framework with file-based routing and end-to-end type safety.

Developers write Go route handlers that return typed data, and React components that receive it as props.

The framework generates TypeScript types from Go structs, handles SSR via a Bun sidecar, and hydrates on the client.

The CLI (`rstf init`, `rstf dev`) orchestrates codegen, bundling, and live reloading.

## Specification-driven development

This project uses `*-spec.md` files co-located with their modules as the source of truth.
Search for `*-spec.md` to find all specifications.

**Workflow for any code change:**

1. Read the relevant spec(s) first.
2. Update the spec to reflect the intended change.
3. Implement the code to match the spec.

Note: if code and spec disagree, the code must be updated to match the spec.
