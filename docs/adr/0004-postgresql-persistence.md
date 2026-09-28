# ADR 0004 — PostgreSQL for operational state

## Status

Accepted

## Context

The Phase 1 repository kept clusters, incidents, and approvals in process memory. Restarting the API discarded an approval and rewound `INC-142`.

## Decision

PostgreSQL is the system of record for durable operational state. The schema is applied by versioned SQL files embedded in the API. Repositories use explicit SQL through pgx. There is no ORM.

The simulated incident engine still computes telemetry in memory, because those series are a function of the remediation clock. On startup the API reloads `demo_state.epoch` and `remediation_started_at`, so a restart continues the same story instead of reseeding a second `INC-142`.

Approvals, proposals, executions, timeline rows, and an append-only `audit_events` table are written when a remediation is approved. Kubernetes discovery writes separate rows with `source = kubernetes` and never updates the simulated `payment-api` metrics.

`POST /api/v1/demo/reset` is refused when `OPSPILOT_ENV=production`. In development it deletes only `INC-142` approval, execution, proposal, and audit rows, then writes one `demo-reset` audit record. It does not truncate the database.

## Consequences

- Local development needs PostgreSQL. Docker Compose provides it with development-only credentials in `.env.example`.
- Integration tests skip unless `OPSPILOT_TEST_DATABASE_URL` is set, so CI does not require a database.
- Without `OPSPILOT_DATABASE_URL` the API still boots on the memory repository. That mode does not survive a restart.
