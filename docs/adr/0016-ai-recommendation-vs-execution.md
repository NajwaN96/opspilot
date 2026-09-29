# ADR 0016 — AI recommendation versus deterministic execution

## Status

Accepted

## Context

`ROLLBACK_PAYMENT_API` is a useful sentence about a payment-api incident. It is not a description of which cluster, namespace, Deployment, image, or revision to change. Those values are already fixed by the Phase 4 proposal builder.

## Decision

An accepted investigation stores a semantic action. It does not create an approval and it does not call the mutator.

The deterministic proposal is still `release.Propose(observed version)`. Only version `1.5.0-bad` produces `rollback-payment-api` with a fixed target of `1.4.2`. Policy and a human approval remain mandatory. The constrained executor is still the only component that updates the Deployment. Prometheus still has to show recovery before the incident resolves.

The semantic action `ROLLBACK_PAYMENT_API` is a different string from the executor action `rollback-payment-api`. Passing the model string to `release.Validate` is a denial.

## Consequences

- A model can recommend a rollback on a healthy `1.4.2` snapshot and the proposal builder still refuses it.
- The incident page shows deterministic findings and the AI synthesis as separate sections.
