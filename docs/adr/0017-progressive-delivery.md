# ADR 0017 — Progressive delivery and canary routing

## Status

Accepted

## Context

Phase 4 can replace the whole payment-api Deployment after a person approves. A new release should be observable at a small traffic share before it becomes the stable version. A service mesh or Argo Rollouts would add a control plane this local cluster does not need.

## Decision

`demo-shop/payment-api` stays the stable Deployment. `payment-api-canary` is a second Deployment and Service. `checkout-api` sends 0, 5, 25, 50, or 100 percent of `/pay` calls to the canary. The weight is stored in the `payment-routing` ConfigMap and applied through the existing allow-listed checkout endpoint. There is no generic patch API and no mesh.

The rollout state machine is persisted in PostgreSQL and resumed by a worker after an API restart. Only one payment-api rollout may be active. A Phase 4 rollback is rejected while a canary is active.

## Consequences

- Five percent of the local traffic generator is about one request every two seconds, so the first stage waits until Prometheus has seen 20 candidate requests.
- Checkout must be the rebuilt demo image. An older checkout binary ignores the weight.
