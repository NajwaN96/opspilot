# ADR 0018 — SLO-gated promotion and the constrained canary executor

## Status

Accepted

## Context

Elapsed time is not evidence that a candidate is safe. The browser must not choose a namespace, image, or query.

## Decision

Each stage waits at least 45 seconds and then reads server-built PromQL for one allow-listed payment-api version. The canary window is `2m`. Five percent of the local generator is about 0.25 requests per second, so a `1m` increase never reaches 20 requests. Callers still cannot choose the range: `BuildVersionQueries` accepts only `1m`, `2m`, and `15m`, and only for versions `1.4.2`, `1.5.0`, `1.5.0-bad`, and `1.6.0-bad`. The gate passes only when the candidate has at least 20 requests, an error rate at or below 5%, a known p95 at or below 300ms, and a Ready pod. Otherwise the result is `INSUFFICIENT_DATA` or `FAIL`. Missing telemetry is not success.

Passing 5% and 25% advances the weight. Passing 50% on `1.5.0` proposes `PROMOTE_PAYMENT_API_CANARY`. Any fail proposes `ABORT_PAYMENT_API_CANARY`. Both mutations wait for a human approval. The executor can promote only `opspilot-demo:1.5.0` and can remove only `payment-api-canary`. `1.6.0-bad` cannot be promoted. Verification uses the version-scoped Prometheus window and must succeed twice before `SUCCEEDED` or `ABORTED`.

Reset Demo restores stable `1.4.2`, scales the canary to zero, and closes the active rollout. The lab start endpoints are refused when `OPSPILOT_ENV=production`.

## Consequences

- A Ready pod with a 25% error rate fails the gate. Kubernetes readiness is not the promotion signal.
- Alertmanager and Grafana are read-only additions. They do not approve rollouts.
