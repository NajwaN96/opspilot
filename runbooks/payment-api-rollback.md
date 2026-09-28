# Runbook: roll back payment-api

Two different rollbacks share this name. They do not share an executor.

## INC-142, simulated

This section matches the simulated action for INC-142. It is not wired to a cluster. Do not treat the commands below as something OpsPilot runs.

### When to use it

- Incident: payment-api latency and error rate jumped in the same minute as a rollout.
- Current version `v1.8.2`, previous healthy version `v1.8.1`.
- Logs show `database connection pool exhausted`.
- Trace time is in the postgres span, not in the checkout edge.
- Policy class for this rollback is LOW, and a person has approved it.

### What OpsPilot does

The console sends:

```json
{ "action": "rollback" }
```

to `POST /api/v1/incidents/INC-142/remediations`.

The service checks that this is the allowed proposal, then the simulated executor records:

- namespace `payments`
- deployment `payment-api`
- from `v1.8.2`
- to `v1.8.1`

The process then reports the workflow steps and the recovered metrics. It does not call the Kubernetes API.

### Do not

- Give an investigator or a chat tool a cluster-admin kubeconfig.
- Auto-approve a rollback because confidence is high.
- Run `kubectl` from the demo button.

## demo-shop, real local rollback

This section is for cluster `opspilot-dev` only. It does not apply to a cloud account.

### What the bad release does

`payment-api` `1.5.0-bad` (`opspilot-demo:1.5.0-bad`) stays Ready. About 30% of `GET /pay` responses are HTTP 500, and successful and failed calls both wait 500ms. Metrics and traces continue.

The good release is `1.4.2` (`opspilot-demo:1.4.2`). The Reliability Lab fault is a different control. Stopping an experiment does not change this image.

### Deploy the bad release

In the console, open Reliability Lab and choose **Deploy Bad payment**. That calls `POST /api/v1/rollouts/payment-api/bad` with an empty object. The API ignores any image name and updates only `demo-shop/payment-api`.

The control is off when `OPSPILOT_ENV=production`.

### What detection should do

Rule `PAYMENT_API_RELIABILITY_DEGRADATION` still requires 20 requests in one minute and two bad evaluations. Error rate above 5% or p95 above 300ms opens or resumes one `INC-REAL-…` incident.

When the workload version is `1.5.0-bad`, the proposal is `rollback-payment-api` from `1.5.0-bad` to `1.4.2`. Other versions cannot be approved. Action `rollback` is still rejected for these incidents.

### Approve

On the incident, **Approve & Execute** sends `{ "action": "rollback-payment-api" }`. The API records the approval, then sets the image to `opspilot-demo:1.4.2`. No other Deployment changes.

The response means the rollout is Ready and Prometheus verification has started. It does not mean the incident is resolved.

### Verify

The verifier waits for two consecutive healthy one-minute windows, up to three minutes. On success the incident becomes `resolved` and the audit trail gains approval, rollout, and verification rows. If Prometheus stays degraded, verification is `failed` and the incident stays `mitigating`.

A second approval of an already-running or resolved rollback returns the existing record and does not change the Deployment again.
