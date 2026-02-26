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
- A clear handler convention for page, action, and API endpoints.
- Typed request parsing and response helpers.
- Consistent status/redirect semantics for actions.

