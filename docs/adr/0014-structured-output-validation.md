# ADR 0014 — Structured output validation

## Status

Accepted

## Context

Free-form model text cannot be shown as an investigation, and it cannot be allowed to name an action the server does not understand.

## Decision

The OpenAI call uses a strict JSON schema (`response_format.json_schema`, `strict: true`) for summary, likely causes, observations, one recommended action, missing information, and risk notes. The server unmarshals that object and validates it again.

Validation rejects malformed JSON, an empty summary, an action outside `NO_ACTION`, `CONTINUE_INVESTIGATION`, `ROLLBACK_PAYMENT_API`, and `STOP_RELIABILITY_EXPERIMENT`, evidence IDs that are not in the snapshot, versions, trace IDs, and deployment names that do not appear in the snapshot, and confidence outside 0 to 1. A likely cause must cite supporting evidence. Confidence is labeled model confidence, not a calibrated probability.

Invalid output is stored as `invalid_output` and is not shown as an accepted investigation. `real` is true only when the provider is OpenAI, the status is completed, and validation accepted the output.

## Consequences

- A hallucinated Deployment or trace ID fails the investigation instead of becoming a fact.
- The action enum cannot grow because a model invented a string.
