# Workfront 08: Assets and Production Build

## Overview
Complete production asset/tooling pipeline:
- `rstf build`
- minification and manifest generation
- asset fingerprinting and static serving strategy

## Why This Workfront Exists
A batteries-included framework needs an end-to-end production artifact flow, not only `dev` mode.

## Key Outcomes
- Deterministic production build command.
- Hashed assets and manifest-driven injection.
- Packaging strategy for deployment-ready outputs.
- Generated server runtime hardening profile for production deployments.

## Production Hardening Criteria
- Generated production server uses explicit HTTP timeouts (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`).
- Timeout values are configurable via framework config with safe defaults.
- Build output documents runtime hardening assumptions and deployment expectations.
