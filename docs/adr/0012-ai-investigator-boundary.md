# ADR 0012 — AI investigator boundary

## Status

Accepted

## Context

Phase 4 can roll `demo-shop/payment-api` from `1.5.0-bad` back to `1.4.2` after a person approves a server-built proposal. An investigator that can explain that incident is useful only if it cannot also change the cluster.

## Decision

The investigator lives in `internal/investigate`. That package may read a prepared evidence snapshot and return a structured recommendation. It does not import the executor, the Kubernetes client, `os/exec`, `database/sql`, or a container runtime.

The provider interface is `InvestigatorProvider`:

- `OpenAIProvider` calls the chat completions API with the snapshot JSON.
- `FixtureProvider` returns a labeled fixture and is never treated as a real model.
- `DisabledProvider` reports that investigation is unavailable.

`OPSPILOT_AI_PROVIDER=openai` and `OPSPILOT_AI_API_KEY` are read from the process environment. Local development may load a gitignored repo-root `.env`, and that loader does not override variables that are already set. The key is not written to logs, the frontend, Kubernetes manifests, Compose, or `.env.example`.

If the provider is unconfigured, unauthorized, timed out, or unavailable, detection, diagnosis, policy, approval, and the constrained rollback keep working. The incident page shows that AI investigation is unavailable. A failed real call is not replaced with a fixture result.

## Consequences

- A model outage cannot roll a Deployment.
- Tests can run the investigator without a network by selecting the fixture or disabled provider explicitly.
- Adding a new provider does not require giving it cluster credentials.
