# ADR 0007: Local OpenTelemetry architecture

## Status

Accepted

## Context

demo-shop needs real traces for a request chain without a SaaS exporter, a service mesh, or a large tracing distribution.

## Decision

Applications export OTLP/HTTP to an OpenTelemetry Collector in `opspilot-system`. The collector batches spans, sets `deployment.environment=local`, and forwards OTLP to Jaeger all-in-one. Jaeger is the trace backend because one container exposes a stable JSON query API (`/api/traces`) that OpsPilot can read without adding another query service.

Prometheus is separate and scrapes `/metrics` directly. The collector does not export metrics.

Nothing leaves the laptop. There is no external exporter.

## Consequences

- Trace IDs in the UI come from Jaeger.
- Jaeger memory storage is ephemeral. Incident rows keep trace IDs and a short summary, not the full span tree.
- CoreDNS must be running. In this nested Docker environment its image is imported locally because registry pulls time out.
