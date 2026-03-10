# Minimum Lovable Product

## Product Identity

A Go-first web framework for building server-rendered apps with React islands, typed server data, and a tight local-to-production workflow.

## Target User

The first lovable user is a Go developer who wants:

- server rendering by default
- React for UI and interactivity
- minimal frontend/backend glue code
- a simple mental model for routes, data loading, and hydration
- a deployment artifact they can actually ship

This is a narrower and stronger target than "anyone building any full-stack app."

## The Split

The product should be split by user outcome, not by internal subsystem.

### Slice 1: Read-only app from zero to deploy

This is the actual minimum lovable product.

User outcome:

> I can create a new `rstf` app, add a few pages with server data, run it locally, build it, and deploy one binary.

Must include:

- `rstf init`
- `rstf dev`
- `rstf build`
- file-based routing
- SSR `GET` handlers
- typed server data into React components
- shared layouts/components
- CSS support with a clear default path
- a production-ready build artifact and startup path

Must not include:

- mutations/actions
- auth/session primitives
- caching/revalidation
- background jobs
- rate limiting
- mobile promises
- a separate RPC/procedure abstraction

Lovable release criteria:

- empty folder to running app is smooth
- route/data conventions are obvious
- local dev loop is reliable
- production build and deploy story is boring and predictable

Without this slice, the framework may be technically interesting but not lovable.

### Slice 2: Mutating apps

User outcome:

> I can build CRUD flows without inventing my own action, validation, redirect, and error conventions.

Must include:

- first-class actions or mutation handlers
- forms integration
- validation flow
- typed success/error results
- redirect semantics
- stable error model for user-facing failures

Why this comes next:

- once people can write data safely, the framework becomes useful for real app work
- this is the smallest step from "content app" to "product app"

This slice should absorb the current validation/forms and error-model work.

### Slice 3: Authenticated multi-user apps

User outcome:

> I can build a real account-based web app with secure defaults.

Must include:

- sessions
- cookie strategy
- CSRF for state-changing operations
- authorization hooks
- middleware integration with the request lifecycle

Why this comes after Slice 2:

- auth without a solid action/error model creates weak framework contracts
- security primitives should attach to a stable write-path story, not precede it

This slice should absorb the current auth/session/CSRF work.

## Explicit Non-Goals For The First Lovable Release

The first lovable release should not try to solve:

- mobile app generation or mobile runtime concerns
- general-purpose RPC between client and server
- revalidation/caching strategy
- queues, cron, or jobs
- rate limiting and abuse controls
- full observability platform concerns

Those are valid later workfronts, but they do not define the first product users will love.

## Why This Split Is The Right One

It forces the framework to complete a full user loop before broadening scope.

Bad split:

- routing
- renderer
- codegen
- validation
- auth

That is an implementation roadmap, not a product roadmap.

Good split:

- read-only app
- mutating app
- authenticated app

That maps directly to what a developer can accomplish and whether the framework is actually lovable.

## Current Recommendation

Treat Slice 1 as the release gate.

If a feature does not improve the "zero to deployed SSR app" story, it should not block the first lovable release. If a missing capability breaks that story, it is core and should move forward immediately.
