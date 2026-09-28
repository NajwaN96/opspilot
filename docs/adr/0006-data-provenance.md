# ADR 0006 — Data provenance

## Status

Accepted

## Context

OpsPilot now shows two kinds of operational data at once: a simulated production incident and a real local cluster. Mixing them would make a demo look like a live outage, or make a healthy nginx pod look like `INC-142`.

## Decision

Every service has `source` and `telemetry`.

- `source=simulation` and `telemetry=simulated` is the seeded `production-01` catalog, including `payment-api` and `INC-142`.
- `source=kubernetes` and `telemetry=none` is a Deployment discovered from `demo-shop`. The API does not copy availability, latency, or error rate onto it.

The console repeats the source on the overview, the service page, the infrastructure page, and the incident page. Infrastructure renders only the Kubernetes read. The simulated node table is not labeled as the local cluster.

Audit rows are control-plane history, not Kubernetes events. Kubernetes events stay on the infrastructure page and are labeled as such.

## Consequences

- The same short name can exist twice: simulated `payment-api` and discovered `k8s_demo-shop_payment-api`. Identifiers do not collide.
- Later collectors (Prometheus, OpenTelemetry) must add their own source value instead of filling the simulated series.
