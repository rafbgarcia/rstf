# Workfront 04: Auth, Sessions, and CSRF

## Overview
Ship built-in authentication foundations:
- session lifecycle and storage abstraction
- cookie strategy (signing, expiry, rotation)
- CSRF protections for state-changing operations

## Why This Workfront Exists
Most web apps require auth/security primitives immediately. This layer depends on stable action and validation contracts.

## Key Outcomes
- First-party session API with middleware integration.
- Secure defaults for cookies and CSRF.
- Authorization hooks usable from SSR/action/API handlers.

