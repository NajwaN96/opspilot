# ADR 0010: Deterministic incident detection

## Status

Accepted

## Context

OpsPilot needs to open an incident from live telemetry without a model, and it must not open a new incident on every poll.

## Decision

Rule `PAYMENT_API_RELIABILITY_DEGRADATION` evaluates `payment-api` in `demo-shop` every 15 seconds.

It breaches only when Prometheus data is present, the one-minute request count is at least 20, and either the error rate is above 5% or p95 latency is above 300ms. Two consecutive breaches open an incident. Two consecutive healthy evaluations record telemetry recovered. Missing data or low volume holds the streak and does not open or recover.

The fingerprint is `rule|service|namespace|cluster`. A partial unique index allows one non-resolved incident per fingerprint. Recovery does not resolve the incident. The streak is in memory and resets when the API restarts; the database still prevents a second active incident.

Diagnosis is `detection.Diagnose`. It lists a likely cause, supporting evidence, and contradicting evidence. It does not invent a confidence percentage.

## Consequences

- A single request cannot open an incident.
- Stale Prometheus data is not used: a failed query is unavailable.
- The incident stays open after recovery so an operator can still read the evidence.
