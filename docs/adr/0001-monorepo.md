# ADR 0001 — One repository for the console and the control plane

## Status

Accepted

## Context

OpsPilot has a TypeScript console and a Go API. They share the incident, service, and remediation shapes. The product is early: the API contract will change as the simulated world is replaced, and a split repository would make those changes two reviews.

## Decision

Keep `apps/web` and `apps/api` in one Git repository. Each app has its own toolchain (`package.json`, `go.mod`). There is no shared code generator and no workspace tool beyond the directory layout. CI builds them as separate jobs.

Documentation, ADRs, runbooks, and the demo script live at the repository root because they describe the system, not one app.

## Consequences

A console change and an API change can land together, which matches how the incident payload is evolving.

The repository will contain two dependency graphs. That is acceptable. We will not introduce a JavaScript monorepo orchestrator until a third app needs it.

Contract drift is possible because the TypeScript types are written by hand. While the payload is small, review is enough. A generated client becomes worth it when the API grows past the current resources.

## Alternatives

Separate repositories would let the API version independently. We do not have external consumers yet, so the coordination cost is not paying for anything.

A single TypeScript process (Next.js route handlers) would remove a network hop. The control plane is expected to own policy, execution, and later cluster credentials. That code should not live in the UI server. Go keeps that boundary obvious.
