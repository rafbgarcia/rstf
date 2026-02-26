# Workfront 11: Rate Limiting and Policy

## Overview
Define first-class rate limiting primitives as middleware-level policy controls,
separate from request admission/backpressure.

## Why This Workfront Exists
Admission control protects server capacity. Rate limiting protects app/domain
fairness and abuse boundaries. These are related but distinct concerns and need
independent, explicit contracts.

## Key Outcomes
- Global and per-route rate limiting hooks.
- Stable keying model (IP, user ID, org ID, custom key).
- Clear strategy primitives (fixed window / token bucket) with explicit burst behavior.
- Deterministic error contract for throttled requests (`429`).
- Observability for throttling decisions and hot keys.

## Contract Decisions (Proposed)
- Enforcement point is middleware (before handler execution).
- Default response is `429 Too Many Requests` with standard error envelope.
- Start with in-memory limiter for single-instance deployments.
- Define pluggable store interface for distributed backends (e.g. Redis) in v2.
- Framework-managed policies should be declarative config, not ad-hoc handler calls.

## Proposed Policy Model (v1)
- Global policy:
  - requests per period,
  - burst size,
  - key selector.
- Route policy overrides:
  - optional per-route limits,
  - inherit global defaults when not specified.
- Key selectors:
  - built-ins: `IP`, `UserID`, `OrganizationID`,
  - custom function for advanced use cases.

## Error Contract
- Status: `429`
- Envelope:

```json
{
  "error": {
    "code": "rate_limited",
    "message": "too many requests",
    "details": {
      "retryAfterSeconds": 10
    }
  }
}
```

## Production Hardening Criteria
- Bounded memory behavior for in-memory limiter.
- Deterministic behavior under clock drift and burst traffic.
- Explicit behavior for missing key identity (fallback strategy).
- Metrics for throttles, top keys, per-route throttle rate.

## Out of Scope
- Server admission/backpressure (`run`/`enqueue`/`drop`) belongs to Workfront 01.
