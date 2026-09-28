# Runbook: roll back payment-api

This runbook matches the simulated action for INC-142. It is not wired to a cluster. Do not treat the commands below as something OpsPilot runs.

## When to use it

- Incident: payment-api latency and error rate jumped in the same minute as a rollout.
- Current version `v1.8.2`, previous healthy version `v1.8.1`.
- Logs show `database connection pool exhausted`.
- Trace time is in the postgres span, not in the checkout edge.
- Policy class for this rollback is LOW, and a person has approved it.

## What OpsPilot does in the MVP

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

## What a future executor would do

After the same approval, a constrained client would roll the Deployment back to the previous ReplicaSet and stop. It would not delete the namespace, scale to zero, or edit any other object.

Verification, still read-only:

- error rate near the pre-deploy baseline (demo target 0.3%)
- p95 near 240ms
- postgres connection usage off the saturation line (demo target 37%)
- no new surge in checkout deadlines

If verification fails, the incident stays open. The executor does not try a second, different action on its own.

## Do not

- Give an investigator or a chat tool a cluster-admin kubeconfig.
- Auto-approve a rollback because confidence is high.
- Run `kubectl` from the demo button.
