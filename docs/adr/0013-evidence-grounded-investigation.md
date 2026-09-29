# ADR 0013 — Evidence-grounded investigation

## Status

Accepted

## Context

A model that can query Prometheus, Jaeger, or Kubernetes on its own can invent a scope the server did not intend. Phase 4 also dropped traces that arrived a few seconds after an incident opened.

## Decision

The investigator receives one versioned evidence snapshot and nothing else. The snapshot is stored in `incident_evidence_snapshots`. Each item has a stable ID and a source: detection, Prometheus, OpenTelemetry/Jaeger, the Kubernetes API, deployment metadata, deterministic diagnosis, or a runbook.

The snapshot is bounded: a few failed or slow traces, metric summaries, the payment-api Deployment, a few pods and events, the known release pair, and a clipped runbook excerpt. Omitted counts are recorded.

After an incident opens, the runner keeps a short enrichment window. If relevant traces appear, it writes a new snapshot version and only then investigates. It does not poll the model on every detection cycle. One completed investigation is kept per snapshot version unless a person asks to run it again after a failure.

Retrieval of runbooks is by service and rule. There is no vector index.

## Consequences

- Citations point at rows the server stored.
- A late Jaeger export can still land in the incident.
- The model cannot widen the query after the snapshot is built.
