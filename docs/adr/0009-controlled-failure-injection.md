# ADR 0009: Controlled failure injection

## Status

Accepted

## Context

The detection path has to be proven with a real failure. Deleting pods, running shell in a container, or applying arbitrary manifests would break the safety boundary and the laptop cluster.

## Decision

`payment-api` has an in-process fault. `POST /internal/fault` accepts `degraded` or `off`, requires header `X-OpsPilot-Fault-Token`, and rejects an expiry more than five minutes away. Degraded mode adds 500ms and returns HTTP 500 for about 40% of `/pay` requests. The process clears the fault when `until` passes.

OpsPilot reaches that endpoint only through an allow-listed API-server service proxy: namespace `demo-shop`, service `payment-api`, path `/internal/fault`. The browser cannot choose a namespace, a cluster, a shell command, or a manifest.

The allowlist is `payment-api-degraded` on service id `k8s_demo-shop_payment-api`, durations 30, 60, or 120 seconds. Production (`OPSPILOT_ENV=production`) disables the controller. Stopping the fault is recorded as `lab-fault-cleared`. It is not a Deployment rollback.

## Consequences

- The fault token is a local demo value (`opspilot-local-fault` unless `OPSPILOT_FAULT_TOKEN` is set). It is not printed in traces or API responses.
- Experiments cannot target PostgreSQL, `opspilot-system`, or any other namespace.
- A real Kubernetes rollback remains a later, separate action.
