# ADR 0008: Prometheus query boundary

## Status

Accepted

## Context

The console needs request rate, error rate, and latency. A browser-supplied PromQL parameter would be an arbitrary query API against the cluster.

## Decision

`internal/telemetry.BuildQueries` is the only place that creates PromQL. It accepts a service name and namespace that match `^[a-z0-9][a-z0-9-]{0,62}$` and a window of `1m` or `15m`. Callers pass the name stored for a discovered service, never a query string from the request.

The HTTP API has no `/query` route. The Kubernetes service proxy used to reach Prometheus allows only `GET /api/v1/query`, `GET /-/ready`, and `GET /-/healthy`.

Empty or `NaN` results are unavailable telemetry. Detection does not treat them as healthy or as a breach.

## Consequences

- Adding a metric means changing the server-side builder and its tests.
- Label sets stay low-cardinality: `service`, `namespace`, `version`, `route`, and `status_code` on the counter. The histogram omits `status_code`.
