# Workfront 03: Error Model and Resilience

## Overview
Define framework-level error behavior:
- not found and internal error handling
- safe production responses
- structured error boundaries between handler, renderer, and transport layers

## Why This Workfront Exists
Error behavior must be consistent early. It influences handler contracts, response types, observability, and security posture.

## Key Outcomes
- Standard error types and mapping rules.
- Default 404/500 handling behavior.
- Clear separation between user-facing and internal errors.
- SSR timeout budget and timeout/error mapping policy.
- End-to-end request cancellation propagation across handler -> renderer client -> sidecar.
- Sidecar supervision/restart policy with explicit failure-mode handling.

## Production Hardening Criteria
- Define renderer timeout defaults, override model, and status-code semantics.
- Define cancellation guarantees when clients disconnect or handlers are canceled.
- Define sidecar crash/restart behavior, retry boundaries, and failure escalation rules.
