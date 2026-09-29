# ADR 0019 — AI recommendation versus the rollout gate

## Status

Accepted

## Context

The Phase 5 investigator can read a snapshot and recommend a semantic action. A canary is another place that recommendation could be mistaken for permission.

## Decision

When a gate reaches promote or abort, the investigator may see a bounded snapshot: rollout id, versions, weight, candidate metrics, readiness, and a runbook excerpt. It may recommend only `NO_ACTION`, `CONTINUE_INVESTIGATION`, `CONTINUE_CANARY`, `PROMOTE_PAYMENT_API_CANARY`, or `ABORT_PAYMENT_API_CANARY`.

The deterministic proposal is computed first. If the model says promote and the gate failed, the rollout records the mismatch and the proposal stays abort. The model does not receive the mutator, a shell, PromQL, or an image name to choose. A provider failure leaves the gate in force.

## Consequences

- The Rollouts page shows the SLO gate and the model recommendation as separate sections.
- An OpenAI quota or timeout does not promote or remove a candidate.
