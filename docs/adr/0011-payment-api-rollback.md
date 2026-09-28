# ADR 0011 — Constrained payment-api rollback

## Status

Accepted

## Context

Phase 3 can open a real incident from Prometheus and traces, then stop at a recommendation. The next proof is one live remediation. A general Kubernetes write API would let the browser choose a namespace, Deployment, or image, which breaks the safety boundary.

The Reliability Lab fault is an in-process switch. It must stay separate from a bad Deployment, or a rollback would not change the thing that is failing.

## Decision

The only live remediation is `rollback-payment-api`.

The target is fixed in the API:

- cluster `opspilot-dev`
- namespace `demo-shop`
- Deployment and container `payment-api`
- bad release `1.5.0-bad`, image `opspilot-demo:1.5.0-bad`
- good release `1.4.2`, image `opspilot-demo:1.4.2`

The bad image is the same demo binary built with a compiled-in fault: about 30% HTTP 500 and 500ms of latency. It stays Ready and keeps emitting metrics and traces. The good image has no compiled-in fault. The browser cannot submit either name.

`POST /api/v1/rollouts/payment-api/bad` is development-only. It accepts no image, namespace, or manifest. It is not the Reliability Lab experiment.

Detection proposes the rollback only when the observed workload version is `1.5.0-bad`. Every other version keeps the non-executable `stop-experiment` recommendation. `get` recomputes that proposal from the stored version so a client cannot flip `allowed`.

Approval is written to `approvals` and `audit_events` before the Deployment update. The mutator then changes that one container image, `SERVICE_VERSION`, and the version labels. It reads the live object first and refuses any other image. The detection engine receives the read-only cluster view, not the mutator.

After the rollout is Ready, a verifier polls the existing payment-api Prometheus window. The incident is resolved only after two consecutive healthy evaluations on `opspilot-demo:1.4.2`. A timeout records a failed verification and leaves the incident open. `INC-142` still uses the simulated executor.

There is no `POST /kubectl`, `/exec`, `/apply`, or `/patch`.

## Consequences

- A bad deploy is a real image change. Recovery requires that image to roll back and Prometheus to agree.
- The action cannot repair a different service, even if that would be useful in a demo.
- Restarting the API resumes verification for executions still marked `verifying`. It does not mutate the cluster on startup.
