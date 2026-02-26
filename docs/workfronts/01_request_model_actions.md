# Workfront 01: Request Model and Actions

## Overview

Define the core request handling model beyond SSR page rendering:

- page SSR handlers
- JSON/API handlers
- form/action handlers (POST/PUT/PATCH/DELETE)

This workfront establishes the framework's fundamental runtime contract for reads and writes.

## Why This Workfront Exists

The framework currently centers on `SSR` for GET-like rendering. A batteries-included framework needs first-class mutation and API primitives before higher-level features can be built safely.

## Key Outcomes

- [x] A clear handler convention for page, action, and API endpoints.
- [x] Typed request parsing and response helpers.
- [x] Consistent status/redirect semantics for actions.
- [x] Request size limits and payload validation contracts for SSR/action/API endpoints.
- [x] Standard oversized/invalid payload error envelope and status mapping.

## Production Hardening Criteria

- [x] Define default and configurable request body size limits.
- [ ] Define backpressure/admission behavior when request queues are saturated.
- [ ] Ensure request parsing failures are deterministic and safe for production responses (partially implemented).

## Contract Decisions (Current)

- Routes remain under `routes/` only (no separate `api/` root).
- Same path content negotiation:
  - `GET` with `Accept: text/html` uses `SSR`.
  - `GET` with non-HTML `Accept` uses `GET`.
  - If the selected handler is missing, return `406 Not Acceptable`.
- Action handlers are method-named exported functions in the route package:
  - `POST`, `PUT`, `PATCH`, `DELETE` (v1).
- `HEAD` and `OPTIONS` are framework-provided by default in v1:
  - `HEAD` mirrors `GET` headers/status without body.
  - `OPTIONS` returns method metadata/preflight baseline without requiring user code.
- End-user response model:
  - Handlers use `func METHOD(ctx *rstf.Context) error`.
  - Users can write directly to standard Go web primitives (`http.ResponseWriter` and `*http.Request`) via context.
  - Framework response helpers are optional sugar (`JSON`, `Text`, `Redirect`, `NoContent`).
- Body parsing and error contract defaults:
  - Default body limit: `1 MiB` (configurable via `app.SetRequestBodyLimitBytes`).
  - Status mapping: `400` invalid payload, `413` payload too large, `415` unsupported content type, `422` validation failure.
  - Standard error envelope:

```json
{
  "error": {
    "code": "invalid_payload",
    "message": "human-readable message",
    "details": {}
  }
}
```

## Implementation Status (Current)

- Parser now recognizes route handlers: `SSR`, `GET`, `POST`, `PUT`, `PATCH`, `DELETE`.
- Generated server now dispatches by HTTP method per route:
  - `GET`: Accept negotiation (`text/html` preference -> SSR page render, non-HTML -> `GET` handler, missing selected handler -> `406`).
  - `POST`/`PUT`/`PATCH`/`DELETE`: call route action handlers.
  - `HEAD`: framework-provided from `GET` behavior (same status/headers, no body).
  - `OPTIONS`: framework-generated `Allow` metadata per route.
- `rstf.Context` now includes request/response helpers:
  - `BindJSON` (with default 1 MiB body limit; respects app-configured request body limit),
  - `JSON`, `Text`, `Redirect`, `NoContent`.
- Standard error envelope writer is implemented for action/API errors via `WriteErrorEnvelope`.
- Integration test scenarios under `tests/integration/test_project/routes` now implement concrete handlers (no comment-only stubs).
- Added focused scenario route `routes/actions-exhaustive-supported-verbs` to exercise the full supported method set in one place.

## Progress Snapshot

- [x] Implemented:
  - parser + codegen + runtime dispatch for `SSR`, `GET`, `POST`, `PUT`, `PATCH`, `DELETE`
  - Accept negotiation with `406` when selected handler is missing
  - framework-provided `HEAD`/`OPTIONS`
  - context helpers (`BindJSON`, `JSON`, `Text`, `Redirect`, `NoContent`)
  - standard action/API error envelope writer
  - scenario route coverage in `tests/integration/test_project/routes`
- [ ] Missing:
  - admission/backpressure implementation (`run`/`enqueue`/`drop`)
  - full contract-test completion for all mappings and negotiation edge cases
  - unrestricted integration runtime verification (sandbox currently blocks port-binding tests)

## Recommended Direction

- Keep admission control deterministic and explicit:
  - run when in-flight budget is available,
  - enqueue when queue budget is available,
  - drop with `503` + standard envelope when saturated.
- Treat request admission and rate limiting as separate concerns:
  - Workfront 01 owns server admission/backpressure behavior.
  - Rate limiting policy and keying are tracked in Workfront 11.
- Keep v1 policy minimal and production-safe:
  - `MaxConcurrentRequests`,
  - `MaxQueuedRequests`,
  - `QueueTimeout`.
- Prefer middleware-level enforcement before handler execution.

## Remaining To Finish Workfront 01

- [x] Add configurable request body limits wired through app config (default remains `1 MiB`).
- [ ] Implement admission/backpressure runtime controls (`run`/`enqueue`/`drop`) and overload envelope behavior.
- [ ] Add exhaustive contract tests for:
  - `400`, `413`, `415`, `422` mappings,
  - Accept negotiation edge cases and `406`,
  - `HEAD`/`OPTIONS` framework defaults.
- [ ] Verify integration/runtime behavior in unrestricted environment (port-binding tests).
