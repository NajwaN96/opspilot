# ADR 0002 — Go for the control plane

## Status

Accepted

## Context

The API is a small set of reads plus two commands: approve a remediation, start a simulated experiment. It needs clear package boundaries, structured logs, and a place to put policy that is not the browser. Later it will sit next to Kubernetes clients and long-running intake.

## Decision

Implement the control plane in Go, standard library HTTP (`net/http` ServeMux, `log/slog`, `encoding/json`). No web framework and no third-party router.

Layout:

- handlers translate HTTP to service calls
- the service authorizes commands
- the executor performs an action
- the repository loads and saves state

The first repository is in-memory and rebuilds the world on each read from a clock plus the remediation timestamp. That keeps the demo deterministic and avoids a database in the MVP. The `Catalog` interface is the seam for PostgreSQL.

## Consequences

The binary has no runtime dependencies. Tests use `httptest` and run in a few milliseconds.

JSON field names are the contract. The UI copies them into `src/lib/types.ts`.

`slog` JSON logs each request with method, path, status, and duration. There is no tracing SDK yet. When OpenTelemetry arrives, it should wrap the same handler chain rather than replace it.

The memory store holds approval state in the process. A restart replays `INC-142` from the beginning. That is correct for the demo and wrong for a real on-call tool, which is why persistence is the next milestone.

## Alternatives

Node route handlers would share types with the UI. They would also put the future executor in the same process that renders the console. We want the credential boundary to be the API process.

An immediate PostgreSQL dependency would make the demo heavier and would not change the UI. The interface is in place so the store can be swapped without a handler rewrite.
