# ADR 0020 — Rollout state ownership

## Status

Accepted

## Context

Phase 6 stored the last canary stage in `rollouts.weight` and also used that number as if it were still the live split. A desired replica count of zero was mapped to degraded. Detection recorded telemetry recovery without resolving the incident. Alertmanager can keep a canary rule firing while old samples age out of the range.

## Decision

Each fact has one source:

- Kubernetes owns the Deployment image, replica count, and readiness.
- ConfigMap `demo-shop/payment-routing` owns the live canary weight.
- PostgreSQL owns the rollout lifecycle, stage weight, analyses, proposals, approvals, executions, and the audit log.
- Prometheus owns quantitative pass/fail evidence.
- Alertmanager owns whether a rule is firing.
- Jaeger owns trace evidence.
- The investigator owns a recommendation and nothing else.

`desired=0` and `ready=0` is `idle`. `desired>0` and `ready<desired` stays degraded. The API reports `stageWeight` and `liveWeight` separately. `stableVersion` stays the baseline recorded when the rollout started. `liveStableVersion` is read from the stable Deployment and is not written back over that baseline. Historical events are not rewritten.

`SUCCEEDED` requires the promoted image to be Ready, the canary scaled to zero, live weight 0, and a healthy verification. `ABORTED` requires stable `1.4.2` Ready, the same idle candidate, and a healthy verification. `FAILED` means verification or startup could not be completed safely. `NEEDS_ATTENTION` is used when a repair cannot be proven; the controller does not mark that rollout successful.

On each tick the controller re-applies the stored stage weight and candidate replicas while a rollout is pending, running, or awaiting approval. Promotion and abort are not reversed by that repair. A repeated tick that already matches writes no second reconciliation event.

A detection incident for `k8s_demo-shop_payment-api` is resolved only when the latest rollout is `ABORTED` or `SUCCEEDED`, the incident started during that rollout, detection has set `telemetry_recovered_at`, the candidate is idle, live weight is 0, the stable Deployment is Ready on the expected image, and the current one-minute Prometheus window is healthy. Missing telemetry does not resolve it. Simulated incidents are not selected. The investigator cannot close an incident.

If Alertmanager still reports a known payment alert as firing after a terminal rollout whose live weight and desired replicas are both 0, the API interpretation is `recovering` and the note says historical samples remain inside the evaluation window. The Alertmanager state itself stays `firing` until the rule clears.

## Consequences

- Reset Demo restores stable `1.4.2`, scales the canary to zero, and sets live weight to 0. It keeps the stage weight and the audit trail.
- The Rollouts page shows the last evaluated stage and the live percentage as different fields.
