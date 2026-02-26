# Workfront 05: Data Layer and Migrations

## Overview
Establish the database developer experience:
- migrations and seeding
- transaction and unit-of-work conventions
- DB configuration patterns for local/dev/test/prod

## Why This Workfront Exists
`*sql.DB` access exists, but batteries-included frameworks provide an opinionated lifecycle for schema evolution and consistent data access workflows.

## Key Outcomes
- Migration command flow and file conventions.
- Seed pipeline for deterministic local/test setup.
- Transaction helpers integrated with request context.

